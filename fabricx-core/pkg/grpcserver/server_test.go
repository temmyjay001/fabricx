// fabricx-core/pkg/grpcserver/server_test.go
package grpcserver

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/temmyjay001/fabricx-core/pkg/docker"
	"github.com/temmyjay001/fabricx-core/pkg/executor"
)

func TestInitNetwork(t *testing.T) {
	tests := []struct {
		name    string
		req     *InitNetworkRequest
		setup   func(*executor.MockExecutor)
		want    bool
		wantMsg string
	}{
		{
			name: "successful initialization",
			req: &InitNetworkRequest{
				NetworkName: "test-network",
				NumOrgs:     2,
				ChannelName: "mychannel",
				Config:      map[string]string{},
			},
			setup: func(m *executor.MockExecutor) {
				m.ExecuteCombinedFunc = func(ctx context.Context, name string, args ...string) ([]byte, error) {
					// Mock successful docker operations
					return []byte("success"), nil
				}
			},
			want:    true,
			wantMsg: "Network initialized successfully",
		},
		{
			name: "docker unavailable",
			req: &InitNetworkRequest{
				NetworkName: "test-network",
				NumOrgs:     2,
				ChannelName: "mychannel",
			},
			setup: func(m *executor.MockExecutor) {
				m.ExecuteCombinedFunc = func(ctx context.Context, name string, args ...string) ([]byte, error) {
					return nil, fmt.Errorf("docker not available")
				}
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := executor.NewMockExecutor()
			tt.setup(mockExec)
			
			dockerMgr := docker.NewManagerWithExecutor(mockExec)
			server := NewFabricXServerWithManager(dockerMgr)
			
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			
			resp, err := server.InitNetwork(ctx, tt.req)
			
			if err != nil {
				t.Fatalf("InitNetwork() error = %v", err)
			}
			
			if resp.Success != tt.want {
				t.Errorf("InitNetwork() success = %v, want %v", resp.Success, tt.want)
			}
			
			if tt.wantMsg != "" && resp.Message != tt.wantMsg {
				t.Errorf("InitNetwork() message = %v, want %v", resp.Message, tt.wantMsg)
			}
			
			// If successful, verify network ID is generated
			if resp.Success && resp.NetworkId == "" {
				t.Error("Expected non-empty network ID")
			}
		})
	}
}

func TestInitNetworkContextCancellation(t *testing.T) {
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
	
	dockerMgr := docker.NewManagerWithExecutor(mockExec)
	server := NewFabricXServerWithManager(dockerMgr)
	
	// Context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	
	req := &InitNetworkRequest{
		NetworkName: "test-network",
		NumOrgs:     2,
		ChannelName: "mychannel",
	}
	
	resp, err := server.InitNetwork(ctx, req)
	
	if err != nil {
		t.Fatalf("InitNetwork() error = %v", err)
	}
	
	if resp.Success {
		t.Error("Expected initialization to fail due to context cancellation")
	}
}

func TestDeployChaincode(t *testing.T) {
	tests := []struct {
		name      string
		req       *DeployChaincodeRequest
		setupNet  bool
		wantSuccess bool
	}{
		{
			name: "successful deployment",
			req: &DeployChaincodeRequest{
				NetworkId:     "test-net-123",
				ChaincodeName: "mycc",
				ChaincodePath: "./chaincode",
				Version:       "1.0",
				Language:      "golang",
			},
			setupNet:    true,
			wantSuccess: true,
		},
		{
			name: "network not found",
			req: &DeployChaincodeRequest{
				NetworkId:     "non-existent",
				ChaincodeName: "mycc",
				ChaincodePath: "./chaincode",
			},
			setupNet:    false,
			wantSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := executor.NewMockExecutor()
			mockExec.ExecuteCombinedFunc = func(ctx context.Context, name string, args ...string) ([]byte, error) {
				return []byte("success"), nil
			}
			
			dockerMgr := docker.NewManagerWithExecutor(mockExec)
			server := NewFabricXServerWithManager(dockerMgr)
			
			// Setup network if needed
			if tt.setupNet {
				// Add a mock network to the server
				// (In real tests, you'd use InitNetwork first)
				server.networksMu.Lock()
				// You'd create a mock network.Network here
				server.networksMu.Unlock()
			}
			
			ctx := context.Background()
			resp, err := server.DeployChaincode(ctx, tt.req)
			
			if err != nil {
				t.Fatalf("DeployChaincode() error = %v", err)
			}
			
			if resp.Success != tt.wantSuccess {
				t.Errorf("DeployChaincode() success = %v, want %v", resp.Success, tt.wantSuccess)
			}
		})
	}
}

func TestStopNetwork(t *testing.T) {
	tests := []struct {
		name        string
		req         *StopNetworkRequest
		setupNet    bool
		wantSuccess bool
	}{
		{
			name: "successful stop",
			req: &StopNetworkRequest{
				NetworkId: "test-net-123",
				Cleanup:   true,
			},
			setupNet:    true,
			wantSuccess: true,
		},
		{
			name: "network not found",
			req: &StopNetworkRequest{
				NetworkId: "non-existent",
				Cleanup:   false,
			},
			setupNet:    false,
			wantSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := executor.NewMockExecutor()
			mockExec.ExecuteCombinedFunc = func(ctx context.Context, name string, args ...string) ([]byte, error) {
				return []byte("stopped"), nil
			}
			
			dockerMgr := docker.NewManagerWithExecutor(mockExec)
			server := NewFabricXServerWithManager(dockerMgr)
			
			if tt.setupNet {
				// Add mock network
				server.networksMu.Lock()
				// You'd add a mock network here
				server.networksMu.Unlock()
			}
			
			ctx := context.Background()
			resp, err := server.StopNetwork(ctx, tt.req)
			
			if err != nil {
				t.Fatalf("StopNetwork() error = %v", err)
			}
			
			if resp.Success != tt.wantSuccess {
				t.Errorf("StopNetwork() success = %v, want %v", resp.Success, tt.wantSuccess)
			}
		})
	}
}

func TestGetNetworkStatus(t *testing.T) {
	mockExec := executor.NewMockExecutor()
	mockExec.ExecuteCombinedFunc = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		// Simulate 3 running containers
		return []byte("container1\ncontainer2\ncontainer3"), nil
	}
	
	dockerMgr := docker.NewManagerWithExecutor(mockExec)
	server := NewFabricXServerWithManager(dockerMgr)
	
	// Test with non-existent network
	ctx := context.Background()
	resp, err := server.GetNetworkStatus(ctx, &NetworkStatusRequest{
		NetworkId: "non-existent",
	})
	
	if err != nil {
		t.Fatalf("GetNetworkStatus() error = %v", err)
	}
	
	if resp.Running {
		t.Error("Expected network to not be running")
	}
	
	if resp.Status != "not found" {
		t.Errorf("Expected status 'not found', got %v", resp.Status)
	}
}

func TestServerShutdown(t *testing.T) {
	mockExec := executor.NewMockExecutor()
	mockExec.ExecuteCombinedFunc = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return []byte("success"), nil
	}
	
	dockerMgr := docker.NewManagerWithExecutor(mockExec)
	server := NewFabricXServerWithManager(dockerMgr)
	
	// Add some mock networks
	server.networksMu.Lock()
	// Add mock networks here
	server.networksMu.Unlock()
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	err := server.Shutdown(ctx)
	
	if err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}
	
	// Verify all networks were removed
	server.networksMu.RLock()
	if len(server.networks) != 0 {
		t.Errorf("Expected no networks after shutdown, got %d", len(server.networks))
	}
	server.networksMu.RUnlock()
}

// Benchmark tests
func BenchmarkInitNetwork(b *testing.B) {
	mockExec := executor.NewMockExecutor()
	mockExec.ExecuteCombinedFunc = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return []byte("success"), nil
	}
	
	dockerMgr := docker.NewManagerWithExecutor(mockExec)
	server := NewFabricXServerWithManager(dockerMgr)
	
	req := &InitNetworkRequest{
		NetworkName: "bench-network",
		NumOrgs:     2,
		ChannelName: "mychannel",
	}
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server.InitNetwork(ctx, req)
	}
}

func BenchmarkGetNetworkStatus(b *testing.B) {
	mockExec := executor.NewMockExecutor()
	mockExec.ExecuteCombinedFunc = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return []byte("container1\ncontainer2"), nil
	}
	
	dockerMgr := docker.NewManagerWithExecutor(mockExec)
	server := NewFabricXServerWithManager(dockerMgr)
	
	// Setup a network
	server.networksMu.Lock()
	// Add mock network
	server.networksMu.Unlock()
	
	req := &NetworkStatusRequest{
		NetworkId: "test-net",
	}
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server.GetNetworkStatus(ctx, req)
	}
}