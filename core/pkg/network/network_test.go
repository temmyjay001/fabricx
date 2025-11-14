// core/pkg/network/network_test.go
package network

import (
	"context"
	stdErr "errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/temmyjay001/core/pkg/errors"
	"github.com/temmyjay001/core/pkg/executor"
)

func TestBootstrap(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		setup   func(*executor.MockExecutor)
		wantErr bool
		errType error
	}{
		{
			name: "successful bootstrap",
			config: &Config{
				NetworkName: "test-network",
				NumOrgs:     2,
				ChannelName: "mychannel",
			},
			setup: func(m *executor.MockExecutor) {
				m.ExecuteCombinedFunc = func(ctx context.Context, name string, args ...string) ([]byte, error) {
					// Mock successful docker operations
					// Handle different commands
					if len(args) > 0 {
						switch args[0] {
						case "run":
							// Mock cryptogen, configtxgen
							return []byte("success"), nil
						case "exec":
							// Mock peer commands
							return []byte("success"), nil
						default:
							return []byte("success"), nil
						}
					}
					return []byte("success"), nil
				}
			},
			wantErr: false,
		},
		{
			name: "crypto generation fails",
			config: &Config{
				NetworkName: "test-network",
				NumOrgs:     2,
				ChannelName: "mychannel",
			},
			setup: func(m *executor.MockExecutor) {
				m.ExecuteCombinedFunc = func(ctx context.Context, name string, args ...string) ([]byte, error) {
					// Fail on cryptogen
					if len(args) > 1 && args[0] == "run" && contains(args, "cryptogen") {
						return nil, fmt.Errorf("cryptogen failed")
					}
					return []byte("success"), nil
				}
			},
			wantErr: true,
		},
		{
			name:   "with default config",
			config: &Config{
				// Leave all fields empty to test defaults
			},
			setup: func(m *executor.MockExecutor) {
				m.ExecuteCombinedFunc = func(ctx context.Context, name string, args ...string) ([]byte, error) {
					return []byte("success"), nil
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := executor.NewMockExecutor()
			tt.setup(mockExec)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			net, err := Bootstrap(ctx, tt.config, mockExec)

			if (err != nil) != tt.wantErr {
				t.Errorf("Bootstrap() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				// Verify network was created
				if net == nil {
					t.Fatal("Expected network to be created")
				}

				if net.ID == "" {
					t.Error("Expected network ID to be set")
				}

				// Verify defaults were applied
				if tt.config.NetworkName == "" && net.Name != "fabricx-network" {
					t.Errorf("Expected default network name, got %s", net.Name)
				}

				if tt.config.NumOrgs == 0 && len(net.Orgs) != 2 {
					t.Errorf("Expected 2 default orgs, got %d", len(net.Orgs))
				}

				if tt.config.ChannelName == "" && net.Channel.Name != "mychannel" {
					t.Errorf("Expected default channel name, got %s", net.Channel.Name)
				}

				// Verify paths exist
				if _, err := os.Stat(net.BasePath); os.IsNotExist(err) {
					t.Errorf("Expected base path to exist: %s", net.BasePath)
				}

				// Cleanup
				if err := net.Cleanup(); err != nil {
					t.Errorf("Cleanup failed: %v", err)
				}
			}
		})
	}
}

func TestBootstrapContextCancellation(t *testing.T) {
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

	config := &Config{
		NetworkName: "test-network",
		NumOrgs:     2,
		ChannelName: "mychannel",
	}

	// Context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	net, err := Bootstrap(ctx, config, mockExec)

	if err == nil {
		t.Error("Expected error due to context cancellation")
		if net != nil {
			if err := net.Cleanup(); err != nil {
				t.Errorf("Cleanup failed: %v", err)
			}
		}
	}

	if ctx.Err() == nil {
		t.Error("Expected context to be cancelled")
	}
}

func TestGenerateOrganizations(t *testing.T) {
	tests := []struct {
		name    string
		numOrgs int
		wantLen int
	}{
		{
			name:    "2 organizations",
			numOrgs: 2,
			wantLen: 2,
		},
		{
			name:    "3 organizations",
			numOrgs: 3,
			wantLen: 3,
		},
		{
			name:    "1 organization",
			numOrgs: 1,
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orgs := generateOrganizations(tt.numOrgs)

			if len(orgs) != tt.wantLen {
				t.Errorf("generateOrganizations() returned %d orgs, want %d", len(orgs), tt.wantLen)
			}

			// Verify organization structure
			for i, org := range orgs {
				expectedName := fmt.Sprintf("Org%d", i+1)
				if org.Name != expectedName {
					t.Errorf("Org %d: expected name %s, got %s", i, expectedName, org.Name)
				}

				expectedMSPID := fmt.Sprintf("%sMSP", expectedName)
				if org.MSPID != expectedMSPID {
					t.Errorf("Org %d: expected MSPID %s, got %s", i, expectedMSPID, org.MSPID)
				}

				if len(org.Peers) == 0 {
					t.Errorf("Org %d: expected at least one peer", i)
				}

				// Verify port ranges don't overlap
				expectedPort := 7051 + (i * 1000)
				if org.Peers[0].Port != expectedPort {
					t.Errorf("Org %d: expected port %d, got %d", i, expectedPort, org.Peers[0].Port)
				}
			}
		})
	}
}

func TestGenerateOrderers(t *testing.T) {
	orderers := generateOrderers()

	if len(orderers) != 1 {
		t.Errorf("Expected 1 orderer, got %d", len(orderers))
	}

	if orderers[0].Name != "orderer.example.com" {
		t.Errorf("Expected orderer name 'orderer.example.com', got %s", orderers[0].Name)
	}

	if orderers[0].Port != 7050 {
		t.Errorf("Expected orderer port 7050, got %d", orderers[0].Port)
	}
}

func TestWaitForReady(t *testing.T) {
	tests := []struct {
		name        string
		setupNet    func(*Network, *executor.MockExecutor)
		timeout     time.Duration
		wantErr     bool
		wantErrType error
	}{
		{
			name: "network becomes ready",
			setupNet: func(net *Network, mockExec *executor.MockExecutor) {
				// Mock successful readiness checks and channel operations
				mockExec.ExecuteCombinedFunc = func(ctx context.Context, name string, args ...string) ([]byte, error) {
					if len(args) > 0 {
						// Handle different commands
						if contains(args, "channel") && contains(args, "create") {
							return []byte("Channel created"), nil
						}
						if contains(args, "channel") && contains(args, "join") {
							return []byte("Joined channel"), nil
						}
						if contains(args, "channel") && contains(args, "update") {
							return []byte("Updated anchor"), nil
						}
					}
					return []byte("success"), nil
				}
			},
			timeout: 5 * time.Second,
			wantErr: false,
		},
		{
			name: "timeout waiting for ready",
			setupNet: func(net *Network, mockExec *executor.MockExecutor) {
				// Make readiness check fail
				mockExec.ExecuteCombinedFunc = func(ctx context.Context, name string, args ...string) ([]byte, error) {
					// Fail exec commands to simulate containers not ready
					if len(args) > 0 && args[0] == "exec" {
						return nil, fmt.Errorf("container not ready")
					}
					return []byte("success"), nil
				}
			},
			timeout:     100 * time.Millisecond,
			wantErr:     true,
			wantErrType: errors.ErrTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := executor.NewMockExecutor()

			net := &Network{
				ID:   "test-net-123",
				Name: "test-network",
				Orgs: []*Organization{
					{
						Name:   "Org1",
						MSPID:  "Org1MSP",
						Domain: "org1.example.com",
						Peers: []*Peer{
							{Name: "peer0.org1.example.com", Port: 7051},
						},
					},
				},
				Orderers: []*Orderer{
					{Name: "orderer.example.com", Port: 7050},
				},
				Channel: &Channel{
					Name:        "mychannel",
					ProfileName: "TestChannel",
				},
				BasePath:   "/tmp/test",
				ConfigPath: "/tmp/test/config",
				CryptoPath: "/tmp/test/crypto",
				exec:       mockExec,
			}

			tt.setupNet(net, mockExec)

			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			err := net.WaitForReady(ctx)

			if (err != nil) != tt.wantErr {
				t.Errorf("WaitForReady() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && tt.wantErrType != nil {
				if !stdErr.Is(err, tt.wantErrType) {
					t.Errorf("Expected error type %v, got %v", tt.wantErrType, err)
				}
			}
		})
	}
}

func TestGetEndpoints(t *testing.T) {
	net := &Network{
		Orgs: []*Organization{
			{
				Name: "Org1",
				Peers: []*Peer{
					{Name: "peer0.org1", Port: 7051},
				},
			},
			{
				Name: "Org2",
				Peers: []*Peer{
					{Name: "peer0.org2", Port: 8051},
				},
			},
		},
	}

	endpoints := net.GetEndpoints()

	expectedEndpoints := []string{
		"localhost:7051",
		"localhost:8051",
	}

	if len(endpoints) != len(expectedEndpoints) {
		t.Errorf("Expected %d endpoints, got %d", len(expectedEndpoints), len(endpoints))
	}

	for i, expected := range expectedEndpoints {
		if endpoints[i] != expected {
			t.Errorf("Endpoint %d: expected %s, got %s", i, expected, endpoints[i])
		}
	}
}

func TestGetConnectionProfile(t *testing.T) {
	net := &Network{
		Name: "test-network",
		Channel: &Channel{
			Name: "mychannel",
		},
		Orgs: []*Organization{
			{
				Name:   "Org1",
				MSPID:  "Org1MSP",
				Domain: "org1.example.com",
				Peers: []*Peer{
					{Name: "peer0.org1.example.com", Port: 7051},
				},
			},
		},
		Orderers: []*Orderer{
			{Name: "orderer.example.com", Port: 7050},
		},
	}

	profile, err := net.GetConnectionProfile("Org1")

	if err != nil {
		t.Fatalf("GetConnectionProfile() error = %v", err)
	}

	// Verify basic structure
	if profile["name"] != "test-network-network" {
		t.Errorf("Expected name 'test-network-network', got %v", profile["name"])
	}

	// Verify organizations
	orgs := profile["organizations"].(map[string]interface{})
	if _, exists := orgs["Org1"]; !exists {
		t.Error("Expected Org1 in organizations")
	}

	// Verify peers
	peers := profile["peers"].(map[string]interface{})
	if _, exists := peers["peer0.org1.example.com"]; !exists {
		t.Error("Expected peer0.org1.example.com in peers")
	}

	// Verify orderers
	orderers := profile["orderers"].(map[string]interface{})
	if _, exists := orderers["orderer.example.com"]; !exists {
		t.Error("Expected orderer.example.com in orderers")
	}
}

func TestCleanup(t *testing.T) {
	// Create a temporary network
	mockExec := executor.NewMockExecutor()
	mockExec.ExecuteCombinedFunc = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return []byte("success"), nil
	}

	config := &Config{
		NetworkName: "test-cleanup",
		NumOrgs:     1,
		ChannelName: "testchannel",
	}

	ctx := context.Background()
	net, err := Bootstrap(ctx, config, mockExec)
	if err != nil {
		t.Fatalf("Failed to bootstrap network: %v", err)
	}

	// Verify base path exists
	if _, err := os.Stat(net.BasePath); os.IsNotExist(err) {
		t.Fatal("Expected base path to exist")
	}

	// Cleanup
	if err := net.Cleanup(); err != nil {
		t.Errorf("Cleanup() error = %v", err)
	}

	// Verify base path is removed
	if _, err := os.Stat(net.BasePath); !os.IsNotExist(err) {
		t.Error("Expected base path to be removed after cleanup")
	}
}

func TestGenerateCryptoConfig(t *testing.T) {
	net := &Network{
		Orgs: []*Organization{
			{
				Name:   "Org1",
				Domain: "org1.example.com",
				Peers:  []*Peer{{Name: "peer0"}},
			},
			{
				Name:   "Org2",
				Domain: "org2.example.com",
				Peers:  []*Peer{{Name: "peer0"}},
			},
		},
	}

	config := generateCryptoConfig(net)

	// Verify OrdererOrgs
	ordererOrgs := config["OrdererOrgs"].([]map[string]interface{})
	if len(ordererOrgs) != 1 {
		t.Errorf("Expected 1 orderer org, got %d", len(ordererOrgs))
	}

	// Verify PeerOrgs
	peerOrgs := config["PeerOrgs"].([]map[string]interface{})
	if len(peerOrgs) != 2 {
		t.Errorf("Expected 2 peer orgs, got %d", len(peerOrgs))
	}

	// Verify first org
	if peerOrgs[0]["Name"] != "Org1" {
		t.Errorf("Expected first org name 'Org1', got %v", peerOrgs[0]["Name"])
	}

	if peerOrgs[0]["Domain"] != "org1.example.com" {
		t.Errorf("Expected first org domain 'org1.example.com', got %v", peerOrgs[0]["Domain"])
	}
}

// Helper function
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Benchmark tests
func BenchmarkBootstrap(b *testing.B) {
	mockExec := executor.NewMockExecutor()
	mockExec.ExecuteCombinedFunc = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return []byte("success"), nil
	}

	config := &Config{
		NetworkName: "bench-network",
		NumOrgs:     2,
		ChannelName: "benchchannel",
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		net, err := Bootstrap(ctx, config, mockExec)
		if err != nil {
			b.Fatal(err)
		}
		if err := net.Cleanup(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGenerateOrganizations(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		generateOrganizations(3)
	}
}
