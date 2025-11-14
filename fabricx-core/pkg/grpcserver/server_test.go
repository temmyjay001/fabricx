// core/tests/integration/full_lifecycle_test.go
//go:build integration
// +build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/temmyjay001/core/pkg/chaincode"
	"github.com/temmyjay001/core/pkg/docker"
	"github.com/temmyjay001/core/pkg/executor"
	"github.com/temmyjay001/core/pkg/network"
)

// TestFullLifecycle tests the complete workflow from network creation to transaction execution
func TestFullLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	// Step 1: Bootstrap network
	t.Log("Step 1: Bootstrapping network...")
	config := &network.Config{
		NetworkName: "integration-test-network",
		NumOrgs:     2,
		ChannelName: "testchannel",
	}

	net, err := network.Bootstrap(ctx, config, executor.NewRealExecutor())
	if err != nil {
		t.Fatalf("Failed to bootstrap network: %v", err)
	}
	defer net.Cleanup()

	t.Logf("✓ Network created: %s", net.ID)

	// Step 2: Start Docker containers
	t.Log("Step 2: Starting Docker containers...")
	dockerMgr := docker.NewManager(executor.NewRealExecutor())

	if err := dockerMgr.StartNetwork(ctx, net); err != nil {
		t.Fatalf("Failed to start network: %v", err)
	}
	defer dockerMgr.StopNetwork(ctx, net, true)

	t.Log("✓ Containers started")

	// Step 3: Wait for network ready
	t.Log("Step 3: Waiting for network ready...")
	if err := net.WaitForReady(ctx); err != nil {
		t.Fatalf("Network failed to become ready: %v", err)
	}

	t.Log("✓ Network ready")

	// Step 4: Check network status
	t.Log("Step 4: Checking network status...")
	running, status, err := dockerMgr.GetNetworkStatus(ctx, net)
	if err != nil {
		t.Fatalf("Failed to get network status: %v", err)
	}

	if !running {
		t.Fatalf("Expected network to be running, got status: %s", status)
	}

	t.Logf("✓ Network status: %s", status)

	// Step 5: Deploy test chaincode
	t.Log("Step 5: Deploying test chaincode...")

	// Create a simple test chaincode
	chaincodeDir := filepath.Join(net.BasePath, "test-chaincode")
	if err := os.MkdirAll(chaincodeDir, 0755); err != nil {
		t.Fatalf("Failed to create chaincode dir: %v", err)
	}

	// Copy example chaincode
	exampleCC := "../../../examples/chaincode/asset-transfer"
	if _, err := os.Stat(exampleCC); err == nil {
		// Copy chaincode files
		copyDir(exampleCC, chaincodeDir)
	}

	deployer := chaincode.NewDeployer(net, dockerMgr, executor.NewRealExecutor())
	ccID, err := deployer.Deploy(ctx, &chaincode.DeployRequest{
		Name:     "testcc",
		Path:     chaincodeDir,
		Version:  "1.0",
		Language: "golang",
	})
	if err != nil {
		t.Fatalf("Failed to deploy chaincode: %v", err)
	}

	t.Logf("✓ Chaincode deployed: %s", ccID)

	// Step 6: Invoke transaction
	t.Log("Step 6: Invoking transaction...")
	invoker := chaincode.NewInvoker(net, executor.NewRealExecutor())

	txID, _, err := invoker.Invoke(ctx, "testcc", "InitLedger", []string{})
	if err != nil {
		t.Fatalf("Failed to invoke transaction: %v", err)
	}

	t.Logf("✓ Transaction invoked: %s", txID)

	// Step 7: Query ledger
	t.Log("Step 7: Querying ledger...")

	result, err := invoker.Query(ctx, "testcc", "GetAllAssets", []string{})
	if err != nil {
		t.Fatalf("Failed to query ledger: %v", err)
	}

	if len(result) == 0 {
		t.Fatal("Expected non-empty query result")
	}

	t.Logf("✓ Query result: %s", string(result))

	t.Log("✓ Full lifecycle test completed successfully")
}

// TestMockFullLifecycle demonstrates the full lifecycle with mocked executor
func TestMockFullLifecycle(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Setup mock executor
	mockExec := executor.NewMockExecutor()
	mockExec.ExecuteCombinedFunc = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		// Simulate successful docker operations
		if len(args) > 0 {
			switch {
			case contains(args, "queryinstalled"):
				return []byte("Package ID: testcc_1.0:hash123, Label: testcc_1.0"), nil
			case contains(args, "invoke"):
				return []byte("txid [tx123abc456] committed with status (VALID)"), nil
			case contains(args, "query"):
				return []byte(`[{"ID":"asset1","value":"test"}]`), nil
			default:
				return []byte("success"), nil
			}
		}
		return []byte("success"), nil
	}

	// Step 1: Bootstrap network with mock executor
	t.Log("Step 1: Bootstrapping network (mocked)...")
	config := &network.Config{
		NetworkName: "mock-test-network",
		NumOrgs:     2,
		ChannelName: "testchannel",
	}

	net, err := network.Bootstrap(ctx, config, mockExec)
	if err != nil {
		t.Fatalf("Failed to bootstrap network: %v", err)
	}
	defer net.Cleanup()

	if net.ID == "" {
		t.Fatal("Expected network ID to be set")
	}

	t.Logf("✓ Network created: %s", net.ID)

	// Step 2: Start containers (mocked)
	t.Log("Step 2: Starting containers (mocked)...")
	dockerMgr := docker.NewManager(mockExec)

	if err := dockerMgr.StartNetwork(ctx, net); err != nil {
		t.Fatalf("Failed to start network: %v", err)
	}

	t.Log("✓ Containers started")

	// Step 3: Deploy chaincode (mocked)
	t.Log("Step 3: Deploying chaincode (mocked)...")
	deployer := chaincode.NewDeployer(net, dockerMgr, mockExec)

	req := &chaincode.DeployRequest{
		Name:     "testcc",
		Path:     "/chaincode/testcc",
		Version:  "1.0",
		Language: "golang",
	}

	ccID, err := deployer.Deploy(ctx, req)
	if err != nil {
		t.Fatalf("Failed to deploy chaincode: %v", err)
	}

	if ccID == "" {
		t.Fatal("Expected chaincode ID to be set")
	}

	t.Logf("✓ Chaincode deployed: %s", ccID)

	// Step 4: Invoke transaction (mocked)
	t.Log("Step 4: Invoking transaction (mocked)...")
	invoker := chaincode.NewInvoker(net, mockExec)

	txID, _, err := invoker.Invoke(ctx, "testcc", "createAsset", []string{"asset1", "value1"})
	if err != nil {
		t.Fatalf("Failed to invoke transaction: %v", err)
	}

	if txID == "" || txID == "unknown" {
		t.Fatalf("Expected valid transaction ID, got: %s", txID)
	}

	t.Logf("✓ Transaction invoked: %s", txID)

	// Step 5: Query ledger (mocked)
	t.Log("Step 5: Querying ledger (mocked)...")

	result, err := invoker.Query(ctx, "testcc", "getAsset", []string{"asset1"})
	if err != nil {
		t.Fatalf("Failed to query ledger: %v", err)
	}

	if len(result) == 0 {
		t.Fatal("Expected non-empty query result")
	}

	t.Logf("✓ Query result: %s", string(result))

	// Step 6: Stop network (mocked)
	t.Log("Step 6: Stopping network (mocked)...")
	if err := dockerMgr.StopNetwork(ctx, net, true); err != nil {
		t.Fatalf("Failed to stop network: %v", err)
	}

	t.Log("✓ Network stopped")

	// Verify mock was called appropriately
	if len(mockExec.GetCalls()) == 0 {
		t.Error("Expected mock executor to be called")
	}

	t.Logf("✓ Mock executor was called %d times", len(mockExec.GetCalls()))
	t.Log("✓ Full lifecycle test (mocked) completed successfully")
}

// TestLifecycleWithErrors tests error handling throughout the lifecycle
func TestLifecycleWithErrors(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	t.Run("network bootstrap fails", func(t *testing.T) {
		mockExec := executor.NewMockExecutor()
		mockExec.ExecuteCombinedFunc = func(ctx context.Context, name string, args ...string) ([]byte, error) {
			// Fail on cryptogen
			if len(args) > 0 && args[0] == "run" && contains(args, "cryptogen") {
				return nil, fmt.Errorf("docker run failed")
			}
			return []byte("success"), nil
		}

		config := &network.Config{
			NetworkName: "error-test-network",
			NumOrgs:     2,
			ChannelName: "testchannel",
		}

		_, err := network.Bootstrap(ctx, config, mockExec)
		if err == nil {
			t.Error("Expected error during bootstrap")
		}
	})

	t.Run("container start fails", func(t *testing.T) {
		mockExec := executor.NewMockExecutor()

		// First calls succeed (bootstrap), then fail (container start)
		callCount := 0
		mockExec.ExecuteCombinedFunc = func(ctx context.Context, name string, args ...string) ([]byte, error) {
			callCount++
			// Allow initial setup commands to succeed
			if len(args) > 0 {
				if args[0] == "run" || args[0] == "mkdir" {
					return []byte("success"), nil
				}
				// Fail on docker-compose up
				if contains(args, "up") {
					return nil, fmt.Errorf("docker-compose up failed")
				}
			}
			return []byte("success"), nil
		}

		config := &network.Config{
			NetworkName: "error-test-network-2",
			NumOrgs:     1,
			ChannelName: "testchannel",
		}

		net, err := network.Bootstrap(ctx, config, mockExec)
		if err != nil {
			t.Fatalf("Bootstrap failed: %v", err)
		}
		defer net.Cleanup()

		dockerMgr := docker.NewManager(mockExec)
		err = dockerMgr.StartNetwork(ctx, net)
		if err == nil {
			t.Error("Expected error starting network")
		}
	})

	t.Run("chaincode deployment fails", func(t *testing.T) {
		mockExec := executor.NewMockExecutor()
		mockExec.ExecuteCombinedFunc = func(ctx context.Context, name string, args ...string) ([]byte, error) {
			// Fail on chaincode package
			if len(args) > 0 && contains(args, "package") {
				return nil, fmt.Errorf("chaincode package failed")
			}
			return []byte("success"), nil
		}

		config := &network.Config{
			NetworkName: "error-test-network-3",
			NumOrgs:     1,
			ChannelName: "testchannel",
		}

		net, err := network.Bootstrap(ctx, config, mockExec)
		if err != nil {
			t.Fatalf("Bootstrap failed: %v", err)
		}
		defer net.Cleanup()

		dockerMgr := docker.NewManager(mockExec)
		deployer := chaincode.NewDeployer(net, dockerMgr, mockExec)

		req := &chaincode.DeployRequest{
			Name:    "testcc",
			Path:    "/chaincode/testcc",
			Version: "1.0",
		}

		_, err = deployer.Deploy(ctx, req)
		if err == nil {
			t.Error("Expected error deploying chaincode")
		}
	})
}

// TestLifecycleContextCancellation tests context cancellation at various stages
func TestLifecycleContextCancellation(t *testing.T) {
	t.Run("cancel during bootstrap", func(t *testing.T) {
		mockExec := executor.NewMockExecutor()
		mockExec.ExecuteCombinedFunc = func(ctx context.Context, name string, args ...string) ([]byte, error) {
			// Simulate slow operation
			select {
			case <-time.After(5 * time.Second):
				return []byte("success"), nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		config := &network.Config{
			NetworkName: "cancel-test-network",
			NumOrgs:     2,
			ChannelName: "testchannel",
		}

		_, err := network.Bootstrap(ctx, config, mockExec)
		if err == nil {
			t.Error("Expected error due to context cancellation")
		}
	})

	t.Run("cancel during deployment", func(t *testing.T) {
		mockExec := executor.NewMockExecutor()

		// Bootstrap succeeds quickly
		bootstrapDone := false
		mockExec.ExecuteCombinedFunc = func(ctx context.Context, name string, args ...string) ([]byte, error) {
			if !bootstrapDone && len(args) > 0 && args[0] == "run" {
				return []byte("success"), nil
			}
			bootstrapDone = true

			// Deployment is slow
			select {
			case <-time.After(5 * time.Second):
				return []byte("success"), nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		// Bootstrap with long timeout
		bootstrapCtx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		config := &network.Config{
			NetworkName: "cancel-test-network-2",
			NumOrgs:     1,
			ChannelName: "testchannel",
		}

		net, err := network.Bootstrap(bootstrapCtx, config, mockExec)
		if err != nil {
			t.Fatalf("Bootstrap failed: %v", err)
		}
		defer net.Cleanup()

		// Deploy with short timeout
		deployCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		dockerMgr := docker.NewManager(mockExec)
		deployer := chaincode.NewDeployer(net, dockerMgr, mockExec)

		req := &chaincode.DeployRequest{
			Name:    "testcc",
			Path:    "/chaincode/testcc",
			Version: "1.0",
		}

		_, err = deployer.Deploy(deployCtx, req)
		if err == nil {
			t.Error("Expected error due to context cancellation")
		}
	})
}

// BenchmarkFullLifecycle benchmarks the complete workflow
func BenchmarkFullLifecycle(b *testing.B) {
	mockExec := executor.NewMockExecutor()
	mockExec.ExecuteCombinedFunc = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		if len(args) > 0 {
			if contains(args, "queryinstalled") {
				return []byte("Package ID: testcc_1.0:hash123, Label: testcc_1.0"), nil
			}
			if contains(args, "invoke") {
				return []byte("txid [tx123] committed"), nil
			}
		}
		return []byte("success"), nil
	}

	config := &network.Config{
		NetworkName: "bench-network",
		NumOrgs:     2,
		ChannelName: "benchchannel",
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Bootstrap
		net, err := network.Bootstrap(ctx, config, mockExec)
		if err != nil {
			b.Fatal(err)
		}

		// Start network
		dockerMgr := docker.NewManager(mockExec)
		dockerMgr.StartNetwork(ctx, net)

		// Deploy chaincode
		deployer := chaincode.NewDeployer(net, dockerMgr, mockExec)
		req := &chaincode.DeployRequest{
			Name:    "testcc",
			Path:    "/chaincode/testcc",
			Version: "1.0",
		}
		deployer.Deploy(ctx, req)

		// Invoke transaction
		invoker := chaincode.NewInvoker(net, mockExec)
		invoker.Invoke(ctx, "testcc", "invoke", []string{"arg1"})

		// Cleanup
		dockerMgr.StopNetwork(ctx, net, true)
		net.Cleanup()
	}
}

// Helper functions
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate destination path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		// Copy file
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(destPath, data, info.Mode())
	})
}
