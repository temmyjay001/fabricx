// fabricx-core/pkg/docker/manager.go
package docker

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/temmyjay001/fabricx-core/pkg/errors"
	"github.com/temmyjay001/fabricx-core/pkg/executor"
	"github.com/temmyjay001/fabricx-core/pkg/types"
)

type Manager struct {
	mu       sync.Mutex
	networks map[string]*NetworkState
	exec     executor.Executor // Injected executor for testability
}

type NetworkState struct {
	ComposePath string
	ProjectName string
	Running     bool
}

// NewManager creates a new Docker manager
func NewManager() *Manager {
	return NewManagerWithExecutor(executor.NewRealExecutor())
}

// NewManagerWithExecutor creates a manager with custom executor (for testing)
func NewManagerWithExecutor(exec executor.Executor) *Manager {
	return &Manager{
		networks: make(map[string]*NetworkState),
		exec:     exec,
	}
}

// CheckDockerAvailable verifies Docker is installed and running
func (m *Manager) CheckDockerAvailable(ctx context.Context) error {
	_, err := m.exec.ExecuteCombined(ctx, "docker", "version")
	if err != nil {
		return errors.WrapWithContext("CheckDockerAvailable", errors.ErrDockerUnavailable, map[string]interface{}{
			"error": err.Error(),
		})
	}

	_, err = m.exec.ExecuteCombined(ctx, "docker-compose", "version")
	if err != nil {
		return errors.WrapWithContext("CheckDockerAvailable", errors.ErrBinaryMissing, map[string]interface{}{
			"binary": "docker-compose",
			"error":  err.Error(),
		})
	}

	return nil
}

// PullFabricImages pulls required Hyperledger Fabric Docker images
func (m *Manager) PullFabricImages(ctx context.Context) error {
	images := []string{
		"hyperledger/fabric-peer:2.5",
		"hyperledger/fabric-orderer:2.5",
		"hyperledger/fabric-ca:1.5",
		"hyperledger/fabric-tools:2.5",
		"couchdb:3.3",
	}

	for _, image := range images {
		fmt.Printf("📦 Pulling %s...\n", image)

		_, err := m.exec.ExecuteCombined(ctx, "docker", "pull", image)
		if err != nil {
			return errors.WrapWithContext("PullFabricImages", errors.ErrContainerFailed, map[string]interface{}{
				"image": image,
				"error": err.Error(),
			})
		}
	}

	fmt.Println("✅ All Fabric images pulled successfully")
	return nil
}

// StartNetwork starts all containers for a network
func (m *Manager) StartNetwork(ctx context.Context, net types.Network) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	composePath := filepath.Join(net.GetConfigPath(), "docker-compose.yaml")
	projectName := fmt.Sprintf("fabricx-%s", net.GetID())

	fmt.Println("🚀 Starting Fabric network containers...")

	output, err := m.exec.ExecuteCombined(ctx, "docker-compose",
		"-f", composePath,
		"-p", projectName,
		"up", "-d",
	)

	if err != nil {
		return errors.WrapWithContext("StartNetwork", errors.ErrContainerFailed, map[string]interface{}{
			"network_id": net.GetID(),
			"error":      err.Error(),
			"output":     string(output),
		})
	}

	m.networks[net.GetID()] = &NetworkState{
		ComposePath: composePath,
		ProjectName: projectName,
		Running:     true,
	}

	fmt.Println("✅ Network containers started successfully")
	return nil
}

// StopNetwork stops and optionally removes containers
func (m *Manager) StopNetwork(ctx context.Context, net types.Network, cleanup bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, exists := m.networks[net.GetID()]
	if !exists {
		return errors.WrapWithContext("StopNetwork", errors.ErrNetworkNotFound, map[string]interface{}{
			"network_id": net.GetID(),
		})
	}

	fmt.Println("🛑 Stopping Fabric network...")

	args := []string{"-f", state.ComposePath, "-p", state.ProjectName, "down"}
	if cleanup {
		args = append(args, "-v", "--remove-orphans")
	}

	output, err := m.exec.ExecuteCombined(ctx, "docker-compose", args...)
	if err != nil {
		return errors.WrapWithContext("StopNetwork", errors.ErrContainerFailed, map[string]interface{}{
			"network_id": net.GetID(),
			"error":      err.Error(),
			"output":     string(output),
		})
	}

	delete(m.networks, net.GetID())

	// Cleanup network directory if requested
	if cleanup {
		fmt.Println("🧹 Cleaning up network files...")
		if err := net.Cleanup(); err != nil {
			return errors.Wrap("StopNetwork.Cleanup", err)
		}
	}

	fmt.Println("✅ Network stopped successfully")
	return nil
}

// GetNetworkStatus returns the status of all containers
func (m *Manager) GetNetworkStatus(ctx context.Context, net types.Network) (bool, string, error) {
	m.mu.Lock()
	state, exists := m.networks[net.GetID()]
	m.mu.Unlock()

	if !exists {
		return false, "not started", nil
	}

	output, err := m.exec.ExecuteCombined(ctx, "docker-compose",
		"-f", state.ComposePath,
		"-p", state.ProjectName,
		"ps", "-q",
	)

	if err != nil {
		return false, fmt.Sprintf("error checking status: %v", err), nil
	}

	containerIDs := strings.Split(strings.TrimSpace(string(output)), "\n")
	runningCount := 0
	for _, id := range containerIDs {
		if id != "" {
			runningCount++
		}
	}

	return runningCount > 0, fmt.Sprintf("%d containers running", runningCount), nil
}

// StreamLogs streams container logs in real-time
func (m *Manager) StreamLogs(ctx context.Context, net types.Network, containerName string) (<-chan string, <-chan error) {
	logChan := make(chan string, 100)
	errChan := make(chan error, 1)

	go func() {
		defer close(logChan)
		defer close(errChan)

		m.mu.Lock()
		state, exists := m.networks[net.GetID()]
		m.mu.Unlock()

		if !exists {
			errChan <- errors.WrapWithContext("StreamLogs", errors.ErrNetworkNotFound, map[string]interface{}{
				"network_id": net.GetID(),
			})
			return
		}

		args := []string{"-f", state.ComposePath, "-p", state.ProjectName, "logs", "-f"}
		if containerName != "" {
			args = append(args, containerName)
		}

		outChan, streamErrChan, err := m.exec.ExecuteStream(ctx, "docker-compose", args...)
		if err != nil {
			errChan <- errors.Wrap("StreamLogs", err)
			return
		}

		for {
			select {
			case line, ok := <-outChan:
				if !ok {
					return
				}
				select {
				case logChan <- line:
				case <-ctx.Done():
					return
				}
			case err := <-streamErrChan:
				if err != nil {
					errChan <- errors.Wrap("StreamLogs", err)
				}
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	return logChan, errChan
}

// ExecuteInContainer executes a command inside a running container
func (m *Manager) ExecuteInContainer(ctx context.Context, containerName string, command []string) ([]byte, error) {
	args := []string{"exec", containerName}
	args = append(args, command...)

	output, err := m.exec.ExecuteCombined(ctx, "docker", args...)
	if err != nil {
		return output, errors.WrapWithContext("ExecuteInContainer", errors.ErrContainerFailed, map[string]interface{}{
			"container": containerName,
			"command":   command,
			"error":     err.Error(),
			"output":    string(output),
		})
	}

	return output, nil
}

// CopyToContainer copies a file to a container
func (m *Manager) CopyToContainer(ctx context.Context, srcPath, containerName, dstPath string) error {
	_, err := m.exec.ExecuteCombined(ctx, "docker", "cp", srcPath, fmt.Sprintf("%s:%s", containerName, dstPath))
	if err != nil {
		return errors.WrapWithContext("CopyToContainer", errors.ErrContainerFailed, map[string]interface{}{
			"src":       srcPath,
			"container": containerName,
			"dst":       dstPath,
			"error":     err.Error(),
		})
	}
	return nil
}

// CopyFromContainer copies a file from a container
func (m *Manager) CopyFromContainer(ctx context.Context, containerName, srcPath, dstPath string) error {
	_, err := m.exec.ExecuteCombined(ctx, "docker", "cp", fmt.Sprintf("%s:%s", containerName, srcPath), dstPath)
	if err != nil {
		return errors.WrapWithContext("CopyFromContainer", errors.ErrContainerFailed, map[string]interface{}{
			"container": containerName,
			"src":       srcPath,
			"dst":       dstPath,
			"error":     err.Error(),
		})
	}
	return nil
}
