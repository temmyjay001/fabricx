// core/pkg/grpcserver/server.go
package grpcserver

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/temmyjay001/core/pkg/chaincode"
	"github.com/temmyjay001/core/pkg/docker"
	"github.com/temmyjay001/core/pkg/errors"
	"github.com/temmyjay001/core/pkg/executor"
	"github.com/temmyjay001/core/pkg/network"
)

type FabricXServer struct {
	UnimplementedFabricXServiceServer
	networks   map[string]*network.Network
	networksMu sync.RWMutex
	dockerMgr  *docker.Manager
}

func NewFabricXServer(mgr *docker.Manager) *FabricXServer {
	return &FabricXServer{
		networks:  make(map[string]*network.Network),
		dockerMgr: mgr,
	}
}

func (s *FabricXServer) InitNetwork(ctx context.Context, req *InitNetworkRequest) (*InitNetworkResponse, error) {
	log.Printf("InitNetwork called: %s with %d orgs", req.NetworkName, req.NumOrgs)

	// Check context
	if err := ctx.Err(); err != nil {
		return &InitNetworkResponse{
			Success: false,
			Message: fmt.Sprintf("Context error: %v", err),
		}, nil
	}

	// Create network configuration
	config := &network.Config{
		NetworkName:  req.NetworkName,
		NumOrgs:      int(req.NumOrgs),
		ChannelName:  req.ChannelName,
		CustomConfig: req.Config,
	}

	// Bootstrap the network with context
	net, err := network.Bootstrap(ctx, config, executor.NewRealExecutor())
	if err != nil {
		if errors.IsTimeout(err) {
			return &InitNetworkResponse{
				Success: false,
				Message: "Network bootstrap timed out",
			}, nil
		}
		return &InitNetworkResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to bootstrap network: %v", err),
		}, nil
	}

	// Store network reference
	s.networksMu.Lock()
	s.networks[net.ID] = net
	s.networksMu.Unlock()

	// Start Docker containers with context
	if err := s.dockerMgr.StartNetwork(ctx, net); err != nil {
		// Clean up network on failure
		s.networksMu.Lock()
		delete(s.networks, net.ID)
		s.networksMu.Unlock()

		return &InitNetworkResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to start containers: %v", err),
		}, nil
	}

	// Wait for network readiness with context
	if err := net.WaitForReady(ctx); err != nil {
		// Clean up on failure
		s.dockerMgr.StopNetwork(ctx, net, true)
		s.networksMu.Lock()
		delete(s.networks, net.ID)
		s.networksMu.Unlock()

		return &InitNetworkResponse{
			Success: false,
			Message: fmt.Sprintf("Network failed to become ready: %v", err),
		}, nil
	}

	log.Printf("Network %s initialized successfully (ID: %s)", req.NetworkName, net.ID)

	return &InitNetworkResponse{
		Success:   true,
		Message:   "Network initialized successfully",
		NetworkId: net.ID,
		Endpoints: net.GetEndpoints(),
	}, nil
}

func (s *FabricXServer) DeployChaincode(ctx context.Context, req *DeployChaincodeRequest) (*DeployChaincodeResponse, error) {
	log.Printf("DeployChaincode called: %s on network %s", req.ChaincodeName, req.NetworkId)

	// Check context
	if err := ctx.Err(); err != nil {
		return &DeployChaincodeResponse{
			Success: false,
			Message: fmt.Sprintf("Context error: %v", err),
		}, nil
	}

	// Get network
	s.networksMu.RLock()
	net, exists := s.networks[req.NetworkId]
	s.networksMu.RUnlock()

	if !exists {
		return &DeployChaincodeResponse{
			Success: false,
			Message: fmt.Sprintf("Network %s not found", req.NetworkId),
		}, nil
	}

	// Create chaincode deployer
	deployer := chaincode.NewDeployer(net, s.dockerMgr, executor.NewRealExecutor())

	// Deploy chaincode with context
	ccID, err := deployer.Deploy(ctx, &chaincode.DeployRequest{
		Name:                  req.ChaincodeName,
		Path:                  req.ChaincodePath,
		Version:               req.Version,
		Language:              req.Language,
		EndorsementPolicyOrgs: req.EndorsementPolicyOrgs,
	})

	if err != nil {
		if errors.IsTimeout(err) {
			return &DeployChaincodeResponse{
				Success: false,
				Message: "Chaincode deployment timed out",
			}, nil
		}
		return &DeployChaincodeResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to deploy chaincode: %v", err),
		}, nil
	}

	log.Printf("Chaincode %s deployed successfully (ID: %s)", req.ChaincodeName, ccID)

	return &DeployChaincodeResponse{
		Success:     true,
		Message:     "Chaincode deployed successfully",
		ChaincodeId: ccID,
	}, nil
}

func (s *FabricXServer) InvokeTransaction(ctx context.Context, req *InvokeTransactionRequest) (*InvokeTransactionResponse, error) {
	log.Printf("InvokeTransaction called: %s.%s on network %s", req.ChaincodeName, req.FunctionName, req.NetworkId)

	// Check context
	if err := ctx.Err(); err != nil {
		return &InvokeTransactionResponse{
			Success: false,
			Message: fmt.Sprintf("Context error: %v", err),
		}, nil
	}

	// Get network
	s.networksMu.RLock()
	net, exists := s.networks[req.NetworkId]
	s.networksMu.RUnlock()

	if !exists {
		return &InvokeTransactionResponse{
			Success: false,
			Message: fmt.Sprintf("Network %s not found", req.NetworkId),
		}, nil
	}

	// Create transaction invoker
	invoker := chaincode.NewInvoker(net, executor.NewRealExecutor())

	// Invoke transaction with context
	txID, payload, err := invoker.Invoke(ctx, req.ChaincodeName, req.FunctionName, req.Args)
	if err != nil {
		if errors.IsTimeout(err) {
			return &InvokeTransactionResponse{
				Success: false,
				Message: "Transaction timed out",
			}, nil
		}
		return &InvokeTransactionResponse{
			Success: false,
			Message: fmt.Sprintf("Transaction failed: %v", err),
		}, nil
	}

	log.Printf("Transaction invoked successfully: %s", txID)

	return &InvokeTransactionResponse{
		Success:       true,
		Message:       "Transaction invoked successfully",
		TransactionId: txID,
		Payload:       payload,
	}, nil
}

func (s *FabricXServer) QueryLedger(ctx context.Context, req *QueryLedgerRequest) (*QueryLedgerResponse, error) {
	log.Printf("QueryLedger called: %s.%s on network %s", req.ChaincodeName, req.FunctionName, req.NetworkId)

	// Check context
	if err := ctx.Err(); err != nil {
		return &QueryLedgerResponse{
			Success: false,
			Message: fmt.Sprintf("Context error: %v", err),
		}, nil
	}

	// Get network
	s.networksMu.RLock()
	net, exists := s.networks[req.NetworkId]
	s.networksMu.RUnlock()

	if !exists {
		return &QueryLedgerResponse{
			Success: false,
			Message: fmt.Sprintf("Network %s not found", req.NetworkId),
		}, nil
	}

	// Create query executor
	invoker := chaincode.NewInvoker(net, executor.NewRealExecutor())

	// Query ledger with context
	payload, err := invoker.Query(ctx, req.ChaincodeName, req.FunctionName, req.Args)
	if err != nil {
		if errors.IsTimeout(err) {
			return &QueryLedgerResponse{
				Success: false,
				Message: "Query timed out",
			}, nil
		}
		return &QueryLedgerResponse{
			Success: false,
			Message: fmt.Sprintf("Query failed: %v", err),
		}, nil
	}

	log.Printf("Query executed successfully")

	return &QueryLedgerResponse{
		Success: true,
		Message: "Query executed successfully",
		Payload: payload,
	}, nil
}

func (s *FabricXServer) StopNetwork(ctx context.Context, req *StopNetworkRequest) (*StopNetworkResponse, error) {
	log.Printf("StopNetwork called: %s (cleanup: %v)", req.NetworkId, req.Cleanup)

	// Check context
	if err := ctx.Err(); err != nil {
		return &StopNetworkResponse{
			Success: false,
			Message: fmt.Sprintf("Context error: %v", err),
		}, nil
	}

	// Get network
	s.networksMu.Lock()
	net, exists := s.networks[req.NetworkId]
	if exists {
		delete(s.networks, req.NetworkId)
	}
	s.networksMu.Unlock()

	if !exists {
		return &StopNetworkResponse{
			Success: false,
			Message: fmt.Sprintf("Network %s not found", req.NetworkId),
		}, nil
	}

	// Stop Docker containers with context
	if err := s.dockerMgr.StopNetwork(ctx, net, req.Cleanup); err != nil {
		return &StopNetworkResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to stop network: %v", err),
		}, nil
	}

	log.Printf("Network %s stopped successfully", req.NetworkId)

	return &StopNetworkResponse{
		Success: true,
		Message: "Network stopped successfully",
	}, nil
}

func (s *FabricXServer) GetNetworkStatus(ctx context.Context, req *NetworkStatusRequest) (*NetworkStatusResponse, error) {
	// Check context
	if err := ctx.Err(); err != nil {
		return &NetworkStatusResponse{
			Running: false,
			Status:  fmt.Sprintf("context error: %v", err),
		}, nil
	}

	// Get network
	s.networksMu.RLock()
	net, exists := s.networks[req.NetworkId]
	s.networksMu.RUnlock()

	if !exists {
		return &NetworkStatusResponse{
			Running: false,
			Status:  "not found",
		}, nil
	}

	// Get container status from docker manager with context
	running, status, err := s.dockerMgr.GetNetworkStatus(ctx, net)
	if err != nil {
		return &NetworkStatusResponse{
			Running: false,
			Status:  fmt.Sprintf("error: %v", err),
		}, nil
	}

	// Build detailed status
	peers := []*PeerStatus{}
	for _, org := range net.Orgs {
		for _, peer := range org.Peers {
			peers = append(peers, &PeerStatus{
				Name:     peer.Name,
				Org:      org.Name,
				Status:   "running",
				Endpoint: fmt.Sprintf("localhost:%d", peer.Port),
			})
		}
	}

	orderers := []*OrdererStatus{}
	for _, orderer := range net.Orderers {
		orderers = append(orderers, &OrdererStatus{
			Name:     orderer.Name,
			Status:   "running",
			Endpoint: fmt.Sprintf("localhost:%d", orderer.Port),
		})
	}

	return &NetworkStatusResponse{
		Running:  running,
		Status:   status,
		Peers:    peers,
		Orderers: orderers,
	}, nil
}

func (s *FabricXServer) StreamLogs(req *StreamLogsRequest, stream FabricXService_StreamLogsServer) error {
	log.Printf("StreamLogs called for network %s, container %s", req.NetworkId, req.ContainerName)

	// Get network
	s.networksMu.RLock()
	net, exists := s.networks[req.NetworkId]
	s.networksMu.RUnlock()

	if !exists {
		return errors.WrapWithContext("StreamLogs", errors.ErrNetworkNotFound, map[string]interface{}{
			"network_id": req.NetworkId,
		})
	}

	// Get log channels from docker manager with stream context
	ctx := stream.Context()
	logChan, errChan := s.dockerMgr.StreamLogs(ctx, net, req.ContainerName)

	// Forward logs to gRPC stream
	for {
		select {
		case line, ok := <-logChan:
			if !ok {
				return nil
			}
			if err := stream.Send(&LogMessage{
				Timestamp: "",
				Container: req.ContainerName,
				Message:   line,
			}); err != nil {
				return errors.Wrap("StreamLogs.Send", err)
			}
		case err := <-errChan:
			if err != nil {
				return errors.Wrap("StreamLogs", err)
			}
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Shutdown gracefully shuts down the server
func (s *FabricXServer) Shutdown(ctx context.Context) error {
	log.Println("Shutting down FabricX server...")

	s.networksMu.Lock()
	defer s.networksMu.Unlock()

	// Stop all running networks
	for id, net := range s.networks {
		log.Printf("Stopping network %s", id)
		if err := s.dockerMgr.StopNetwork(ctx, net, false); err != nil {
			log.Printf("Error stopping network %s: %v", id, err)
		}
	}

	s.networks = make(map[string]*network.Network)
	log.Println("FabricX server shutdown complete")

	return nil
}
