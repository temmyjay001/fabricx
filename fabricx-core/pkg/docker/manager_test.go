// core/pkg/docker/manager_test.go
package docker

import (
	"context"
	stdErr "errors"
	"testing"
	"time"

	"github.com/temmyjay001/core/pkg/errors"
	"github.com/temmyjay001/core/pkg/executor"
)

// MockNetwork implements types.Network for testing
type MockNetwork struct {
	id         string
	configPath string
	cleanupErr error
}

func (m *MockNetwork) GetID() string            { return m.id }
func (m *MockNetwork) GetConfigPath() string    { return m.configPath }
func (m *MockNetwork) GetOrgs() interface{}     { return nil }
func (m *MockNetwork) GetOrderers() interface{} { return nil }
func (m *MockNetwork) Cleanup() error           { return m.cleanupErr }

func TestCheckDockerAvailable(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*executor.MockExecutor)
		wantErr bool
		errType error
	}{
		{
			name: "docker and docker-compose available",
			setup: func(m *executor.MockExecutor) {
				m.ExecuteCombinedFunc = func(ctx context.Context, name string, args ...string) ([]byte, error) {
					return []byte("Docker version 20.10.0"), nil
				}
			},
			wantErr: false,
		},
		{
			name: "docker not available",
			setup: func(m *executor.MockExecutor) {
				m.ExecuteCombinedFunc = func(ctx context.Context, name string, args ...string) ([]byte, error) {
					if name == "docker" {
						return nil, errors.ErrDockerUnavailable
					}
					return []byte("ok"), nil
				}
			},
			wantErr: true,
			errType: errors.ErrDockerUnavailable,
		},
		{
			name: "docker-compose not available",
			setup: func(m *executor.MockExecutor) {
				callCount := 0
				m.ExecuteCombinedFunc = func(ctx context.Context, name string, args ...string) ([]byte, error) {
					callCount++
					if callCount == 1 {
						return []byte("Docker version"), nil
					}
					return nil, errors.ErrBinaryMissing
				}
			},
			wantErr: true,
			errType: errors.ErrBinaryMissing,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := executor.NewMockExecutor()
			tt.setup(mockExec)

			mgr := NewManager(mockExec)
			ctx := context.Background()

			err := mgr.CheckDockerAvailable(ctx)

			if (err != nil) != tt.wantErr {
				t.Errorf("CheckDockerAvailable() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && tt.errType != nil {
				if !stdErr.Is(err, tt.errType) {
					t.Errorf("Expected error type %v, got %v", tt.errType, err)
				}
			}
		})
	}
}

func TestStartNetwork(t *testing.T) {
	tests := []struct {
		name    string
		network *MockNetwork
		setup   func(*executor.MockExecutor)
		wantErr bool
	}{
		{
			name: "successful start",
			network: &MockNetwork{
				id:         "test-net-123",
				configPath: "/tmp/fabricx/test-net-123/config",
			},
			setup: func(m *executor.MockExecutor) {
				m.ExecuteCombinedFunc = func(ctx context.Context, name string, args ...string) ([]byte, error) {
					return []byte("Creating network... done"), nil
				}
			},
			wantErr: false,
		},
		{
			name: "docker-compose fails",
			network: &MockNetwork{
				id:         "test-net-456",
				configPath: "/tmp/fabricx/test-net-456/config",
			},
			setup: func(m *executor.MockExecutor) {
				m.ExecuteCombinedFunc = func(ctx context.Context, name string, args ...string) ([]byte, error) {
					return []byte("Error"), errors.ErrContainerFailed
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := executor.NewMockExecutor()
			tt.setup(mockExec)

			mgr := NewManager(mockExec)
			ctx := context.Background()

			err := mgr.StartNetwork(ctx, tt.network)

			if (err != nil) != tt.wantErr {
				t.Errorf("StartNetwork() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Verify docker-compose was called
			if !mockExec.WasCalledWith("docker-compose") {
				t.Error("Expected docker-compose to be called")
			}
		})
	}
}

func TestStopNetwork(t *testing.T) {
	tests := []struct {
		name    string
		cleanup bool
		setup   func(*Manager, *MockNetwork)
		wantErr bool
	}{
		{
			name:    "stop without cleanup",
			cleanup: false,
			setup: func(mgr *Manager, net *MockNetwork) {
				// Simulate a running network
				mgr.networks[net.GetID()] = &NetworkState{
					ComposePath: "/tmp/test/docker-compose.yaml",
					ProjectName: "fabricx-test",
					Running:     true,
				}
			},
			wantErr: false,
		},
		{
			name:    "stop with cleanup",
			cleanup: true,
			setup: func(mgr *Manager, net *MockNetwork) {
				mgr.networks[net.GetID()] = &NetworkState{
					ComposePath: "/tmp/test/docker-compose.yaml",
					ProjectName: "fabricx-test",
					Running:     true,
				}
			},
			wantErr: false,
		},
		{
			name:    "network not found",
			cleanup: false,
			setup:   func(mgr *Manager, net *MockNetwork) {},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := executor.NewMockExecutor()
			mockExec.ExecuteCombinedFunc = func(ctx context.Context, name string, args ...string) ([]byte, error) {
				return []byte("Stopping containers... done"), nil
			}

			mgr := NewManager(mockExec)
			net := &MockNetwork{
				id:         "test-net-123",
				configPath: "/tmp/test",
			}

			tt.setup(mgr, net)
			ctx := context.Background()

			err := mgr.StopNetwork(ctx, net, tt.cleanup)

			if (err != nil) != tt.wantErr {
				t.Errorf("StopNetwork() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetNetworkStatus(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*Manager, *MockNetwork, *executor.MockExecutor)
		wantRunning bool
		wantStatus  string
	}{
		{
			name: "network running",
			setup: func(mgr *Manager, net *MockNetwork, mockExec *executor.MockExecutor) {
				mgr.networks[net.GetID()] = &NetworkState{
					ComposePath: "/tmp/test/docker-compose.yaml",
					ProjectName: "fabricx-test",
					Running:     true,
				}
				mockExec.ExecuteCombinedFunc = func(ctx context.Context, name string, args ...string) ([]byte, error) {
					return []byte("container1\ncontainer2\ncontainer3"), nil
				}
			},
			wantRunning: true,
			wantStatus:  "3 containers running",
		},
		{
			name: "network not started",
			setup: func(mgr *Manager, net *MockNetwork, mockExec *executor.MockExecutor) {
				// Don't add network to manager
			},
			wantRunning: false,
			wantStatus:  "not started",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := executor.NewMockExecutor()
			mgr := NewManager(mockExec)
			net := &MockNetwork{
				id:         "test-net-123",
				configPath: "/tmp/test",
			}

			tt.setup(mgr, net, mockExec)
			ctx := context.Background()

			running, status, err := mgr.GetNetworkStatus(ctx, net)

			if err != nil {
				t.Errorf("GetNetworkStatus() error = %v", err)
			}

			if running != tt.wantRunning {
				t.Errorf("GetNetworkStatus() running = %v, want %v", running, tt.wantRunning)
			}

			if status != tt.wantStatus {
				t.Errorf("GetNetworkStatus() status = %v, want %v", status, tt.wantStatus)
			}
		})
	}
}

func TestExecuteInContainer(t *testing.T) {
	tests := []struct {
		name          string
		containerName string
		command       []string
		setup         func(*executor.MockExecutor)
		wantErr       bool
		wantOutput    string
	}{
		{
			name:          "successful execution",
			containerName: "peer0.org1",
			command:       []string{"peer", "version"},
			setup: func(m *executor.MockExecutor) {
				m.ExecuteCombinedFunc = func(ctx context.Context, name string, args ...string) ([]byte, error) {
					return []byte("peer: Version: 2.5.0"), nil
				}
			},
			wantErr:    false,
			wantOutput: "peer: Version: 2.5.0",
		},
		{
			name:          "container not found",
			containerName: "invalid-container",
			command:       []string{"ls"},
			setup: func(m *executor.MockExecutor) {
				m.ExecuteCombinedFunc = func(ctx context.Context, name string, args ...string) ([]byte, error) {
					return []byte("Error: No such container"), errors.ErrContainerFailed
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := executor.NewMockExecutor()
			tt.setup(mockExec)

			mgr := NewManager(mockExec)
			ctx := context.Background()

			output, err := mgr.ExecuteInContainer(ctx, tt.containerName, tt.command)

			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteInContainer() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && string(output) != tt.wantOutput {
				t.Errorf("ExecuteInContainer() output = %v, want %v", string(output), tt.wantOutput)
			}
		})
	}
}

func TestContextCancellation(t *testing.T) {
	t.Run("start network respects context cancellation", func(t *testing.T) {
		mockExec := executor.NewMockExecutor()
		mockExec.ExecuteCombinedFunc = func(ctx context.Context, name string, args ...string) ([]byte, error) {
			// Simulate long-running operation
			select {
			case <-time.After(5 * time.Second):
				return []byte("done"), nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		mgr := NewManager(mockExec)
		net := &MockNetwork{
			id:         "test-net-123",
			configPath: "/tmp/test",
		}

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err := mgr.StartNetwork(ctx, net)

		if err == nil {
			t.Error("Expected error due to context cancellation")
		}

		if ctx.Err() == nil {
			t.Error("Expected context to be cancelled")
		}
	})
}
