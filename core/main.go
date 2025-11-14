// main.go
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/temmyjay001/core/pkg/docker"
	"github.com/temmyjay001/core/pkg/executor"
	"github.com/temmyjay001/core/pkg/grpcserver"
	"google.golang.org/grpc"
)

const (
	defaultPort = "50051"
	version     = "0.1.0"
)

func main() {
	// CLI flags
	port := flag.String("port", defaultPort, "gRPC server port")
	showVersion := flag.Bool("version", false, "Show version")
	flag.Parse()

	if *showVersion {
		fmt.Printf("FabricX Runtime v%s\n", version)
		os.Exit(0)
	}

	// Ensure Docker is available
	dockerManager := docker.NewManager(executor.NewRealExecutor())
	if err := checkDockerAvailable(dockerManager); err != nil {
		log.Fatalf("Docker is not available: %v\nPlease ensure Docker is installed and running.", err)
	}

	// Start gRPC server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", *port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	fabricxServer := grpcserver.NewFabricXServer(dockerManager)
	grpcserver.RegisterFabricXServiceServer(grpcServer, fabricxServer)

	log.Printf("🚀 FabricX Runtime v%s starting on port %s", version, *port)
	log.Printf("📦 All Fabric operations will run in Docker containers")
	log.Printf("✅ No local Fabric binaries required!")

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Println("🛑 Shutting down gracefully...")
		grpcServer.GracefulStop()
	}()

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

func checkDockerAvailable(dockerManager *docker.Manager) error {
	return dockerManager.CheckDockerAvailable(context.Background())
}
