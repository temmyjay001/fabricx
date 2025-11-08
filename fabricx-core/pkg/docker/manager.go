// pkg/docker/manager.go
package docker

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/temmyjay001/fabricx-core/pkg/types"
)

type Manager struct {
	mu       sync.Mutex
	networks map[string]*NetworkState
}

type NetworkState struct {
	ComposePath string
	ProjectName string
	Running     bool
}

func NewManager() *Manager {
	return &Manager{
		networks: make(map[string]*NetworkState),
	}
}

// CheckDockerAvailable verifies Docker is installed and running
func (m *Manager) CheckDockerAvailable() error {
	cmd := exec.Command("docker", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker is not available: %w", err)
	}

	cmd = exec.Command("docker-compose", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker-compose is not available: %w", err)
	}

	return nil
}

// PullFabricImages pulls required Hyperledger Fabric Docker images
func (m *Manager) PullFabricImages() error {
	images := []string{
		"hyperledger/fabric-peer:2.5",
		"hyperledger/fabric-orderer:2.5",
		"hyperledger/fabric-ca:1.5",
		"hyperledger/fabric-tools:2.5",
		"couchdb:3.3",
	}

	for _, image := range images {
		fmt.Printf("📦 Pulling %s...\n", image)
		cmd := exec.Command("docker", "pull", image)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to pull %s: %w", image, err)
		}
	}

	fmt.Println("✅ All Fabric images pulled successfully")
	return nil
}

// StartNetwork starts all containers for a network
func (m *Manager) StartNetwork(net types.Network) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	composePath := filepath.Join(net.GetConfigPath(), "docker-compose.yaml")
	projectName := fmt.Sprintf("fabricx-%s", net.GetID())

	// Start containers
	fmt.Println("🚀 Starting Fabric network containers...")
	cmd := exec.Command("docker-compose",
		"-f", composePath,
		"-p", projectName,
		"up", "-d",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start network: %w\nOutput: %s", err, string(output))
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
func (m *Manager) StopNetwork(net types.Network, cleanup bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, exists := m.networks[net.GetID()]
	if !exists {
		return fmt.Errorf("network %s not found in manager", net.GetID())
	}

	// Stop containers
	fmt.Println("🛑 Stopping Fabric network...")
	args := []string{"-f", state.ComposePath, "-p", state.ProjectName, "down"}
	if cleanup {
		args = append(args, "-v", "--remove-orphans")
	}

	cmd := exec.Command("docker-compose", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to stop network: %w\nOutput: %s", err, string(output))
	}

	delete(m.networks, net.GetID())

	// Cleanup network directory if requested
	if cleanup {
		fmt.Println("🧹 Cleaning up network files...")
		return net.Cleanup()
	}

	fmt.Println("✅ Network stopped successfully")
	return nil
}

// GetNetworkStatus returns the status of all containers
func (m *Manager) GetNetworkStatus(net types.Network) (bool, string, error) {
	m.mu.Lock()
	state, exists := m.networks[net.GetID()]
	m.mu.Unlock()

	if !exists {
		return false, "not started", nil
	}

	// Check container status
	cmd := exec.Command("docker-compose",
		"-f", state.ComposePath,
		"-p", state.ProjectName,
		"ps", "-q",
	)
	output, err := cmd.CombinedOutput()
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
// Returns a channel that sends log lines
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
			errChan <- fmt.Errorf("network %s not found", net.GetID())
			return
		}

		// Stream logs from specific container
		args := []string{"-f", state.ComposePath, "-p", state.ProjectName, "logs", "-f"}
		if containerName != "" {
			args = append(args, containerName)
		}

		cmd := exec.CommandContext(ctx, "docker-compose", args...)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			errChan <- err
			return
		}

		if err := cmd.Start(); err != nil {
			errChan <- err
			return
		}

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			case logChan <- scanner.Text():
			}
		}

		if err := cmd.Wait(); err != nil {
			errChan <- err
		}
	}()

	return logChan, errChan
}

// ExecuteInContainer executes a command inside a running container
func (m *Manager) ExecuteInContainer(containerName string, command []string) ([]byte, error) {
	args := []string{"exec", containerName}
	args = append(args, command...)

	cmd := exec.Command("docker", args...)
	return cmd.CombinedOutput()
}

// CopyToContainer copies a file to a container
func (m *Manager) CopyToContainer(srcPath, containerName, dstPath string) error {
	cmd := exec.Command("docker", "cp", srcPath, fmt.Sprintf("%s:%s", containerName, dstPath))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to copy file: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// CopyFromContainer copies a file from a container
func (m *Manager) CopyFromContainer(containerName, srcPath, dstPath string) error {
	cmd := exec.Command("docker", "cp", fmt.Sprintf("%s:%s", containerName, srcPath), dstPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to copy file: %w\nOutput: %s", err, string(output))
	}
	return nil
}
