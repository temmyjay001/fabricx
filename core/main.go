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
	"runtime"
	"syscall"
	"time"

	"github.com/temmyjay001/core/pkg/docker"
	"github.com/temmyjay001/core/pkg/executor"
	"github.com/temmyjay001/core/pkg/grpcserver"
	"google.golang.org/grpc"
)

var (
	version   = "dev"
	gitCommit = "unknown"
	buildDate = "unknown"
)

const defaultPort = "50051"

func main() {
	// CLI flags
	port := flag.String("port", defaultPort, "gRPC server port")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	if *showVersion {
		fmt.Printf("FabricX Runtime\n")
		fmt.Printf("  Version:    %s\n", version)
		fmt.Printf("  Git Commit: %s\n", gitCommit)
		fmt.Printf("  Built:      %s\n", buildDate)
		fmt.Printf("  Go Version: %s\n", runtime.Version())
		fmt.Printf("  OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
		os.Exit(0)
	}

	// Ensure Docker is available
	dockerManager := docker.NewManager(executor.NewRealExecutor())
	if err := checkDockerAvailable(dockerManager); err != nil {
		log.Fatalf("❌ Docker is not available: %v\n\n"+
			"Please ensure Docker is installed and running:\n"+
			"  • macOS: Start Docker Desktop\n"+
			"  • Linux: sudo systemctl start docker\n"+
			"  • Windows: Start Docker Desktop\n", err)
	}

	// Start gRPC server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", *port))
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", *port, err)
	}

	grpcServer := grpc.NewServer()
	fabricxServer := grpcserver.NewFabricXServer(dockerManager)
	grpcserver.RegisterFabricXServiceServer(grpcServer, fabricxServer)

	log.Printf("🚀 FabricX Runtime v%s starting on port %s", version, *port)
	log.Printf("📦 All Fabric operations will run in Docker containers")
	log.Printf("✅ No local Fabric binaries required!")
	log.Printf("💡 Only Docker needs to be installed and running")

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Println("🛑 Shutting down gracefully...")
		

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		fabricxServer.Shutdown(ctx)
		
		grpcServer.GracefulStop()
	}()

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

func checkDockerAvailable(dockerManager *docker.Manager) error {
	return dockerManager.CheckDockerAvailable(context.Background())
}
