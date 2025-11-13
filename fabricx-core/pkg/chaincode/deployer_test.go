// fabricx-core/pkg/chaincode/deployer_test.go
package chaincode

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/temmyjay001/fabricx-core/pkg/docker"
	"github.com/temmyjay001/fabricx-core/pkg/executor"
	"github.com/temmyjay001/fabricx-core/pkg/network"
)

func createMockNetwork() *network.Network {
	tempDir, _ := os.MkdirTemp("", "fabricx-test-*")

	return &network.Network{
		ID:         "test-net-123",
		Name:       "test-network",
		BasePath:   tempDir,
		ConfigPath: filepath.Join(tempDir, "config"),
		CryptoPath: filepath.Join(tempDir, "crypto"),
		Orgs: []*network.Organization{
			{
				Name:   "Org1",
				MSPID:  "Org1MSP",
				Domain: "org1.example.com",
				Peers: []*network.Peer{
					{Name: "peer0.org1.example.com", Port: 7051},
				},
			},
			{
				Name:   "Org2",
				MSPID:  "Org2MSP",
				Domain: "org2.example.com",
				Peers: []*network.Peer{
					{Name: "peer0.org2.example.com", Port: 8051},
				},
			},
		},
		Orderers: []*network.Orderer{
			{Name: "orderer.example.com", Port: 7050},
		},
		Channel: &network.Channel{
			Name:        "mychannel",
			ProfileName: "TestChannel",
		},
	}
}

func TestDeploy(t *testing.T) {
	tests := []struct {
		name    string
		req     *DeployRequest
		setup   func(*executor.MockExecutor, string)
		wantErr bool
		errType error
	}{
		{
			name: "successful deployment",
			req: &DeployRequest{
				Name:     "mycc",
				Path:     "/chaincode/mycc",
				Version:  "1.0",
				Language: "golang",
			},
			setup: func(m *executor.MockExecutor, tempDir string) {
				// Create temp chaincode directory
				os.MkdirAll(filepath.Join(tempDir, "chaincode"), 0755)

				m.ExecuteCombinedFunc = func(ctx context.Context, name string, args ...string) ([]byte, error) {
					// Simulate successful docker operations
					if len(args) > 0 {
						// Handle different commands
						if contains(args, "package") {
							// Create a dummy package file
							pkgPath := filepath.Join(tempDir, "chaincode", "mycc.tar.gz")
							os.WriteFile(pkgPath, []byte("dummy"), 0644)
							return []byte("Packaged"), nil
						}
						if contains(args, "install") {
							return []byte("Installed"), nil
						}
						if contains(args, "queryinstalled") {
							return []byte("Package ID: mycc_1.0:hash123, Label: mycc_1.0"), nil
						}
						if contains(args, "approveformyorg") {
							return []byte("Approved"), nil
						}
						if contains(args, "commit") {
							return []byte("Committed"), nil
						}
						if contains(args, "invoke") && contains(args, "Init") {
							return []byte("Initialized"), nil
						}
					}
					return []byte("success"), nil
				}
			},
			wantErr: false,
		},
		{
			name: "packaging fails",
			req: &DeployRequest{
				Name:    "mycc",
				Path:    "/chaincode/mycc",
				Version: "1.0",
			},
			setup: func(m *executor.MockExecutor, tempDir string) {
				m.ExecuteCombinedFunc = func(ctx context.Context, name string, args ...string) ([]byte, error) {
					// Fail on package command
					if len(args) > 0 && contains(args, "package") {
						return nil, fmt.Errorf("package failed")
					}
					return []byte("success"), nil
				}
			},
			wantErr: true,
		},
		{
			name: "with default values",
			req: &DeployRequest{
				Name: "mycc",
				Path: "/chaincode/mycc",
				// Version and Language not provided - should use defaults
			},
			setup: func(m *executor.MockExecutor, tempDir string) {
				os.MkdirAll(filepath.Join(tempDir, "chaincode"), 0755)

				m.ExecuteCombinedFunc = func(ctx context.Context, name string, args ...string) ([]byte, error) {
					if contains(args, "package") {
						pkgPath := filepath.Join(tempDir, "chaincode", "mycc.tar.gz")
						os.WriteFile(pkgPath, []byte("dummy"), 0644)
						return []byte("Packaged"), nil
					}
					if contains(args, "queryinstalled") {
						return []byte("Package ID: mycc_1.0:hash123, Label: mycc_1.0"), nil
					}
					return []byte("success"), nil
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := executor.NewMockExecutor()
			net := createMockNetwork()
			defer os.RemoveAll(net.BasePath)

			tt.setup(mockExec, net.BasePath)

			dockerMgr := docker.NewManager(mockExec)
			deployer := NewDeployer(net, dockerMgr, mockExec)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			ccID, err := deployer.Deploy(ctx, tt.req)

			if (err != nil) != tt.wantErr {
				t.Errorf("Deploy() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				if ccID == "" {
					t.Error("Expected non-empty chaincode ID")
				}

				// Verify defaults were applied
				if tt.req.Version == "" && tt.req.Version != "1.0" {
					t.Error("Expected default version to be set")
				}
				if tt.req.Language == "" && tt.req.Language != "golang" {
					t.Error("Expected default language to be set")
				}
			}
		})
	}
}

func TestDeployContextCancellation(t *testing.T) {
	mockExec := executor.NewMockExecutor()
	mockExec.ExecuteCombinedFunc = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		// Simulate long operation
		select {
		case <-time.After(5 * time.Second):
			return []byte("done"), nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	net := createMockNetwork()
	defer os.RemoveAll(net.BasePath)

	dockerMgr := docker.NewManager(mockExec)
	deployer := NewDeployer(net, dockerMgr, mockExec)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	req := &DeployRequest{
		Name:    "mycc",
		Path:    "/chaincode/mycc",
		Version: "1.0",
	}

	_, err := deployer.Deploy(ctx, req)

	if err == nil {
		t.Error("Expected error due to context cancellation")
	}

	if ctx.Err() == nil {
		t.Error("Expected context to be cancelled")
	}
}

func TestGetPackageID(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*executor.MockExecutor)
		ccName  string
		version string
		wantID  string
		wantErr bool
	}{
		{
			name: "package ID found",
			setup: func(m *executor.MockExecutor) {
				m.ExecuteCombinedFunc = func(ctx context.Context, name string, args ...string) ([]byte, error) {
					return []byte("Package ID: mycc_1.0:hash123abc, Label: mycc_1.0\n"), nil
				}
			},
			ccName:  "mycc",
			version: "1.0",
			wantID:  "mycc_1.0:hash123abc",
			wantErr: false,
		},
		{
			name: "package ID not found",
			setup: func(m *executor.MockExecutor) {
				m.ExecuteCombinedFunc = func(ctx context.Context, name string, args ...string) ([]byte, error) {
					return []byte("No packages installed"), nil
				}
			},
			ccName:  "mycc",
			version: "1.0",
			wantErr: true,
		},
		{
			name: "query fails",
			setup: func(m *executor.MockExecutor) {
				m.ExecuteCombinedFunc = func(ctx context.Context, name string, args ...string) ([]byte, error) {
					return nil, fmt.Errorf("query failed")
				}
			},
			ccName:  "mycc",
			version: "1.0",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := executor.NewMockExecutor()
			tt.setup(mockExec)

			net := createMockNetwork()
			defer os.RemoveAll(net.BasePath)

			dockerMgr := docker.NewManager(mockExec)
			deployer := NewDeployer(net, dockerMgr, mockExec)

			ctx := context.Background()
			org := net.Orgs[0]

			packageID, err := deployer.getPackageID(ctx, org, tt.ccName, tt.version)

			if (err != nil) != tt.wantErr {
				t.Errorf("getPackageID() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && packageID != tt.wantID {
				t.Errorf("getPackageID() = %v, want %v", packageID, tt.wantID)
			}
		})
	}
}

func TestBuildEndorsementPolicy(t *testing.T) {
	tests := []struct {
		name         string
		orgs         []string
		wantContains []string
	}{
		{
			name:         "default policy (all orgs)",
			orgs:         []string{},
			wantContains: []string{"OR(", "Org1MSP.member", "Org2MSP.member"},
		},
		{
			name:         "specific org",
			orgs:         []string{"Org1"},
			wantContains: []string{"OR(", "Org1MSP.member"},
		},
		{
			name:         "multiple specific orgs",
			orgs:         []string{"Org1", "Org2"},
			wantContains: []string{"OR(", "Org1MSP.member", "Org2MSP.member"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			net := createMockNetwork()
			defer os.RemoveAll(net.BasePath)

			dockerMgr := docker.NewManager(executor.NewMockExecutor())
			deployer := NewDeployer(net, dockerMgr, executor.NewMockExecutor())

			policy := deployer.buildEndorsementPolicy(tt.orgs)

			for _, want := range tt.wantContains {
				if !containsStr(policy, want) {
					t.Errorf("buildEndorsementPolicy() = %v, want to contain %v", policy, want)
				}
			}
		})
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || contains([]string{s}, substr))
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Invoker Tests

func TestInvoke(t *testing.T) {
	tests := []struct {
		name      string
		chaincode string
		function  string
		args      []string
		setup     func(*executor.MockExecutor)
		wantTxID  string
		wantErr   bool
	}{
		{
			name:      "successful invoke",
			chaincode: "mycc",
			function:  "createAsset",
			args:      []string{"asset1", "value1"},
			setup: func(m *executor.MockExecutor) {
				m.ExecuteCombinedFunc = func(ctx context.Context, name string, args ...string) ([]byte, error) {
					return []byte("Chaincode invoke successful. result: status:200 txid [abc123def456] committed with status (VALID)"), nil
				}
			},
			wantTxID: "abc123def456",
			wantErr:  false,
		},
		{
			name:      "invoke fails",
			chaincode: "mycc",
			function:  "createAsset",
			args:      []string{"asset1"},
			setup: func(m *executor.MockExecutor) {
				m.ExecuteCombinedFunc = func(ctx context.Context, name string, args ...string) ([]byte, error) {
					return nil, fmt.Errorf("invoke failed")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := executor.NewMockExecutor()
			tt.setup(mockExec)

			net := createMockNetwork()
			defer os.RemoveAll(net.BasePath)

			invoker := NewInvoker(net, mockExec)

			ctx := context.Background()
			txID, _, err := invoker.Invoke(ctx, tt.chaincode, tt.function, tt.args)

			if (err != nil) != tt.wantErr {
				t.Errorf("Invoke() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && txID != tt.wantTxID {
				t.Errorf("Invoke() txID = %v, want %v", txID, tt.wantTxID)
			}
		})
	}
}

func TestQuery(t *testing.T) {
	tests := []struct {
		name      string
		chaincode string
		function  string
		args      []string
		setup     func(*executor.MockExecutor)
		wantData  string
		wantErr   bool
	}{
		{
			name:      "successful query",
			chaincode: "mycc",
			function:  "getAsset",
			args:      []string{"asset1"},
			setup: func(m *executor.MockExecutor) {
				m.ExecuteCombinedFunc = func(ctx context.Context, name string, args ...string) ([]byte, error) {
					return []byte(`{"id":"asset1","value":"value1"}`), nil
				}
			},
			wantData: `{"id":"asset1","value":"value1"}`,
			wantErr:  false,
		},
		{
			name:      "query fails",
			chaincode: "mycc",
			function:  "getAsset",
			args:      []string{"nonexistent"},
			setup: func(m *executor.MockExecutor) {
				m.ExecuteCombinedFunc = func(ctx context.Context, name string, args ...string) ([]byte, error) {
					return nil, fmt.Errorf("asset not found")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := executor.NewMockExecutor()
			tt.setup(mockExec)

			net := createMockNetwork()
			defer os.RemoveAll(net.BasePath)

			invoker := NewInvoker(net, mockExec)

			ctx := context.Background()
			data, err := invoker.Query(ctx, tt.chaincode, tt.function, tt.args)

			if (err != nil) != tt.wantErr {
				t.Errorf("Query() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && string(data) != tt.wantData {
				t.Errorf("Query() data = %v, want %v", string(data), tt.wantData)
			}
		})
	}
}

func TestBuildArgsJSON(t *testing.T) {
	net := createMockNetwork()
	defer os.RemoveAll(net.BasePath)

	invoker := NewInvoker(net, executor.NewMockExecutor())

	tests := []struct {
		name     string
		function string
		args     []string
		want     string
	}{
		{
			name:     "no args",
			function: "init",
			args:     []string{},
			want:     `{"Args":["init"]}`,
		},
		{
			name:     "with args",
			function: "createAsset",
			args:     []string{"asset1", "value1"},
			want:     `{"Args":["createAsset","asset1","value1"]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := invoker.buildArgsJSON(tt.function, tt.args)
			if result != tt.want {
				t.Errorf("buildArgsJSON() = %v, want %v", result, tt.want)
			}
		})
	}
}

func TestExtractTxID(t *testing.T) {
	net := createMockNetwork()
	defer os.RemoveAll(net.BasePath)

	invoker := NewInvoker(net, executor.NewMockExecutor())

	tests := []struct {
		name   string
		output string
		want   string
	}{
		{
			name:   "valid tx ID",
			output: "Chaincode invoke successful. result: status:200 txid [abc123def456] committed with status (VALID)",
			want:   "abc123def456",
		},
		{
			name:   "tx ID with newlines",
			output: "Some output\nChaincode invoke successful.\ntxid [xyz789abc] committed\nMore output",
			want:   "xyz789abc",
		},
		{
			name:   "no tx ID",
			output: "Some output without transaction ID",
			want:   "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := invoker.extractTxID(tt.output)
			if result != tt.want {
				t.Errorf("extractTxID() = %v, want %v", result, tt.want)
			}
		})
	}
}

// Benchmark tests
func BenchmarkDeploy(b *testing.B) {
	mockExec := executor.NewMockExecutor()
	mockExec.ExecuteCombinedFunc = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		if len(args) > 0 && contains(args, "queryinstalled") {
			return []byte("Package ID: mycc_1.0:hash123, Label: mycc_1.0"), nil
		}
		return []byte("success"), nil
	}

	net := createMockNetwork()
	defer os.RemoveAll(net.BasePath)

	dockerMgr := docker.NewManager(mockExec)
	deployer := NewDeployer(net, dockerMgr, mockExec)

	req := &DeployRequest{
		Name:    "mycc",
		Path:    "/chaincode/mycc",
		Version: "1.0",
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		deployer.Deploy(ctx, req)
	}
}

func BenchmarkInvoke(b *testing.B) {
	mockExec := executor.NewMockExecutor()
	mockExec.ExecuteCombinedFunc = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return []byte("txid [abc123] committed"), nil
	}

	net := createMockNetwork()
	defer os.RemoveAll(net.BasePath)

	invoker := NewInvoker(net, mockExec)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		invoker.Invoke(ctx, "mycc", "invoke", []string{"arg1"})
	}
}
