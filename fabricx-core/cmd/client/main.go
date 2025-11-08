// cmd/client/main.go
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	pb "github.com/temmyjay001/fabricx-core/pkg/grpcserver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	serverAddr = flag.String("server", "localhost:50051", "FabricX runtime server address")
	timeout    = flag.Duration("timeout", 120*time.Second, "Operation timeout")
)

func main() {
	flag.Parse()

	if len(flag.Args()) == 0 {
		printUsage()
		os.Exit(1)
	}

	command := flag.Args()[0]

	// Connect to server
	conn, err := grpc.Dial(*serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	client := pb.NewFabricXServiceClient(conn)

	// Execute command
	switch command {
	case "init":
		initNetwork(client)
	case "status":
		getStatus(client)
	case "deploy":
		deployChaincode(client)
	case "invoke":
		invokeTransaction(client)
	case "query":
		queryLedger(client)
	case "logs":
		streamLogs(client)
	case "stop":
		stopNetwork(client)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("FabricX CLI Client")
	fmt.Println("\nUsage:")
	fmt.Println("  fabricx-client [flags] <command> [args]")
	fmt.Println("\nFlags:")
	fmt.Println("  -server string    Server address (default: localhost:50051)")
	fmt.Println("  -timeout duration Operation timeout (default: 120s)")
	fmt.Println("\nCommands:")
	fmt.Println("  init              Initialize a new Fabric network")
	fmt.Println("  status <net-id>   Get network status")
	fmt.Println("  deploy <net-id> <chaincode-name> <path> Deploy chaincode")
	fmt.Println("  invoke <net-id> <chaincode> <function> <args...> Invoke transaction")
	fmt.Println("  query <net-id> <chaincode> <function> <args...>  Query ledger")
	fmt.Println("  logs <net-id> [container]  Stream container logs")
	fmt.Println("  stop <net-id>     Stop and cleanup network")
	fmt.Println("\nExamples:")
	fmt.Println("  # Initialize network")
	fmt.Println("  fabricx-client init")
	fmt.Println("")
	fmt.Println("  # Get status")
	fmt.Println("  fabricx-client status abc123")
	fmt.Println("")
	fmt.Println("  # Deploy chaincode")
	fmt.Println("  fabricx-client deploy abc123 mycc ./chaincode")
	fmt.Println("")
	fmt.Println("  # Invoke transaction")
	fmt.Println("  fabricx-client invoke abc123 mycc createAsset asset1 owner1 100")
	fmt.Println("")
	fmt.Println("  # Query")
	fmt.Println("  fabricx-client query abc123 mycc getAsset asset1")
	fmt.Println("")
	fmt.Println("  # Stop network")
	fmt.Println("  fabricx-client stop abc123")
}

func initNetwork(client pb.FabricXServiceClient) {
	args := flag.Args()[1:]

	networkName := "fabricx-network"
	numOrgs := int32(2)
	channelName := "mychannel"

	// Parse optional arguments
	for i := 0; i < len(args); i++ {
		if args[i] == "--name" && i+1 < len(args) {
			networkName = args[i+1]
			i++
		} else if args[i] == "--orgs" && i+1 < len(args) {
			fmt.Sscanf(args[i+1], "%d", &numOrgs)
			i++
		} else if args[i] == "--channel" && i+1 < len(args) {
			channelName = args[i+1]
			i++
		}
	}

	fmt.Printf("🚀 Initializing Fabric network...\n")
	fmt.Printf("   Name: %s\n", networkName)
	fmt.Printf("   Organizations: %d\n", numOrgs)
	fmt.Printf("   Channel: %s\n", channelName)

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	resp, err := client.InitNetwork(ctx, &pb.InitNetworkRequest{
		NetworkName: networkName,
		NumOrgs:     numOrgs,
		ChannelName: channelName,
	})

	if err != nil {
		log.Fatalf("❌ Failed to initialize network: %v", err)
	}

	if !resp.Success {
		log.Fatalf("❌ Network initialization failed: %s", resp.Message)
	}

	fmt.Printf("\n✅ Network initialized successfully!\n")
	fmt.Printf("   Network ID: %s\n", resp.NetworkId)
	fmt.Printf("   Endpoints: %s\n", strings.Join(resp.Endpoints, ", "))
	fmt.Printf("\n💡 Save this network ID for future commands\n")
}

func getStatus(client pb.FabricXServiceClient) {
	args := flag.Args()[1:]
	if len(args) < 1 {
		log.Fatal("Usage: fabricx-client status <network-id>")
	}

	networkID := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.GetNetworkStatus(ctx, &pb.NetworkStatusRequest{
		NetworkId: networkID,
	})

	if err != nil {
		log.Fatalf("❌ Failed to get status: %v", err)
	}

	fmt.Printf("Network Status: %s\n", networkID)
	fmt.Printf("  Running: %v\n", resp.Running)
	fmt.Printf("  Status: %s\n", resp.Status)

	if len(resp.Peers) > 0 {
		fmt.Println("\nPeers:")
		for _, peer := range resp.Peers {
			fmt.Printf("  - %s (%s)\n", peer.Name, peer.Endpoint)
			fmt.Printf("    Organization: %s\n", peer.Org)
			fmt.Printf("    Status: %s\n", peer.Status)
		}
	}

	if len(resp.Orderers) > 0 {
		fmt.Println("\nOrderers:")
		for _, orderer := range resp.Orderers {
			fmt.Printf("  - %s (%s)\n", orderer.Name, orderer.Endpoint)
			fmt.Printf("    Status: %s\n", orderer.Status)
		}
	}
}

func deployChaincode(client pb.FabricXServiceClient) {
	args := flag.Args()[1:]
	if len(args) < 3 {
		log.Fatal("Usage: fabricx-client deploy <network-id> <chaincode-name> <chaincode-path> [--version v] [--lang go|node]")
	}

	networkID := args[0]
	chaincodeName := args[1]
	chaincodePath := args[2]

	version := "1.0"
	language := "golang"

	// Parse optional flags
	for i := 3; i < len(args); i++ {
		if args[i] == "--version" && i+1 < len(args) {
			version = args[i+1]
			i++
		} else if args[i] == "--lang" && i+1 < len(args) {
			language = args[i+1]
			i++
		}
	}

	fmt.Printf("📦 Deploying chaincode...\n")
	fmt.Printf("   Network: %s\n", networkID)
	fmt.Printf("   Chaincode: %s\n", chaincodeName)
	fmt.Printf("   Path: %s\n", chaincodePath)
	fmt.Printf("   Version: %s\n", version)
	fmt.Printf("   Language: %s\n", language)

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	resp, err := client.DeployChaincode(ctx, &pb.DeployChaincodeRequest{
		NetworkId:     networkID,
		ChaincodeName: chaincodeName,
		ChaincodePath: chaincodePath,
		Version:       version,
		Language:      language,
	})

	if err != nil {
		log.Fatalf("❌ Failed to deploy chaincode: %v", err)
	}

	if !resp.Success {
		log.Fatalf("❌ Deployment failed: %s", resp.Message)
	}

	fmt.Printf("\n✅ Chaincode deployed successfully!\n")
	fmt.Printf("   Chaincode ID: %s\n", resp.ChaincodeId)
}

func invokeTransaction(client pb.FabricXServiceClient) {
	args := flag.Args()[1:]
	if len(args) < 3 {
		log.Fatal("Usage: fabricx-client invoke <network-id> <chaincode> <function> [args...]")
	}

	networkID := args[0]
	chaincodeName := args[1]
	functionName := args[2]
	txArgs := args[3:]

	fmt.Printf("📝 Invoking transaction...\n")
	fmt.Printf("   Network: %s\n", networkID)
	fmt.Printf("   Chaincode: %s\n", chaincodeName)
	fmt.Printf("   Function: %s\n", functionName)
	fmt.Printf("   Args: %v\n", txArgs)

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	resp, err := client.InvokeTransaction(ctx, &pb.InvokeTransactionRequest{
		NetworkId:     networkID,
		ChaincodeName: chaincodeName,
		FunctionName:  functionName,
		Args:          txArgs,
	})

	if err != nil {
		log.Fatalf("❌ Failed to invoke transaction: %v", err)
	}

	if !resp.Success {
		log.Fatalf("❌ Transaction failed: %s", resp.Message)
	}

	fmt.Printf("\n✅ Transaction invoked successfully!\n")
	fmt.Printf("   Transaction ID: %s\n", resp.TransactionId)

	if len(resp.Payload) > 0 {
		fmt.Printf("   Payload: %s\n", string(resp.Payload))
	}
}

func queryLedger(client pb.FabricXServiceClient) {
	args := flag.Args()[1:]
	if len(args) < 3 {
		log.Fatal("Usage: fabricx-client query <network-id> <chaincode> <function> [args...]")
	}

	networkID := args[0]
	chaincodeName := args[1]
	functionName := args[2]
	queryArgs := args[3:]

	fmt.Printf("🔍 Querying ledger...\n")
	fmt.Printf("   Network: %s\n", networkID)
	fmt.Printf("   Chaincode: %s\n", chaincodeName)
	fmt.Printf("   Function: %s\n", functionName)
	fmt.Printf("   Args: %v\n", queryArgs)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := client.QueryLedger(ctx, &pb.QueryLedgerRequest{
		NetworkId:     networkID,
		ChaincodeName: chaincodeName,
		FunctionName:  functionName,
		Args:          queryArgs,
	})

	if err != nil {
		log.Fatalf("❌ Failed to query: %v", err)
	}

	if !resp.Success {
		log.Fatalf("❌ Query failed: %s", resp.Message)
	}

	fmt.Printf("\n✅ Query successful!\n")

	// Try to pretty-print JSON
	var prettyJSON interface{}
	if err := json.Unmarshal(resp.Payload, &prettyJSON); err == nil {
		formatted, _ := json.MarshalIndent(prettyJSON, "   ", "  ")
		fmt.Printf("   Result:\n   %s\n", string(formatted))
	} else {
		fmt.Printf("   Result: %s\n", string(resp.Payload))
	}
}

func streamLogs(client pb.FabricXServiceClient) {
	args := flag.Args()[1:]
	if len(args) < 1 {
		log.Fatal("Usage: fabricx-client logs <network-id> [container-name]")
	}

	networkID := args[0]
	containerName := ""
	if len(args) > 1 {
		containerName = args[1]
	}

	fmt.Printf("📜 Streaming logs from network %s", networkID)
	if containerName != "" {
		fmt.Printf(" (container: %s)", containerName)
	}
	fmt.Println("\n   Press Ctrl+C to stop\n")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := client.StreamLogs(ctx, &pb.StreamLogsRequest{
		NetworkId:     networkID,
		ContainerName: containerName,
	})

	if err != nil {
		log.Fatalf("❌ Failed to start log stream: %v", err)
	}

	for {
		msg, err := stream.Recv()
		if err != nil {
			log.Printf("Stream ended: %v", err)
			break
		}

		fmt.Printf("[%s] %s\n", msg.Container, msg.Message)
	}
}

func stopNetwork(client pb.FabricXServiceClient) {
	args := flag.Args()[1:]
	if len(args) < 1 {
		log.Fatal("Usage: fabricx-client stop <network-id> [--cleanup]")
	}

	networkID := args[0]
	cleanup := false

	for i := 1; i < len(args); i++ {
		if args[i] == "--cleanup" {
			cleanup = true
		}
	}

	fmt.Printf("🛑 Stopping network %s", networkID)
	if cleanup {
		fmt.Print(" (with cleanup)")
	}
	fmt.Println()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := client.StopNetwork(ctx, &pb.StopNetworkRequest{
		NetworkId: networkID,
		Cleanup:   cleanup,
	})

	if err != nil {
		log.Fatalf("❌ Failed to stop network: %v", err)
	}

	if !resp.Success {
		log.Fatalf("❌ Stop failed: %s", resp.Message)
	}

	fmt.Printf("\n✅ Network stopped successfully!\n")
	if cleanup {
		fmt.Printf("   All containers and volumes removed\n")
	}
}
