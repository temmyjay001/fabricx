// core/pkg/network/network.go
package network

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/temmyjay001/core/pkg/errors"
	"github.com/temmyjay001/core/pkg/executor"
)

type Config struct {
	NetworkName  string
	NumOrgs      int
	ChannelName  string
	CustomConfig map[string]string
}

type Network struct {
	ID         string
	Name       string
	Config     *Config
	BasePath   string
	Orgs       []*Organization
	Orderers   []*Orderer
	Channel    *Channel
	CryptoPath string
	ConfigPath string
	exec       executor.Executor // For testing
}

type Organization struct {
	Name       string
	MSPID      string
	Domain     string
	Peers      []*Peer
	CAPort     int
	AnchorPort int
}

type Peer struct {
	Name    string
	Port    int
	CouchDB bool
	DBPort  int
}

type Orderer struct {
	Name   string
	Port   int
	Domain string
}

type Channel struct {
	Name        string
	ProfileName string
}

// Bootstrap creates a new network with real executor
func Bootstrap(ctx context.Context, config *Config, exec executor.Executor) (*Network, error) {
	// Check context
	if err := ctx.Err(); err != nil {
		return nil, errors.Wrap("Bootstrap", err)
	}

	// Generate network ID
	netID := uuid.New().String()[:8]

	// Set defaults
	if config.NetworkName == "" {
		config.NetworkName = "fabricx-network"
	}
	if config.NumOrgs == 0 {
		config.NumOrgs = 2
	}
	if config.ChannelName == "" {
		config.ChannelName = "mychannel"
	}

	// Create base directory
	basePath := filepath.Join(os.TempDir(), "fabricx", netID)
	fmt.Printf("Creating network at %s\n", basePath)
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, errors.WrapWithContext("Bootstrap.CreateDir", err, map[string]interface{}{
			"base_path": basePath,
		})
	}

	// Initialize network structure
	net := &Network{
		ID:         netID,
		Name:       config.NetworkName,
		Config:     config,
		BasePath:   basePath,
		CryptoPath: filepath.Join(basePath, "crypto-config"),
		ConfigPath: filepath.Join(basePath, "config"),
		Channel: &Channel{
			Name:        config.ChannelName,
			ProfileName: "FabricXChannel",
		},
		exec: exec,
	}

	// Generate organizations
	net.Orgs = generateOrganizations(config.NumOrgs)

	// Generate orderers
	net.Orderers = generateOrderers()

	// Check context before long operations
	if err := ctx.Err(); err != nil {
		if cleanupErr := net.Cleanup(); cleanupErr != nil {
			fmt.Printf("Warning: failed to cleanup network: %v\n", cleanupErr)
		}
		return nil, errors.Wrap("Bootstrap", err)
	}

	// Generate crypto material
	if err := generateCrypto(ctx, net); err != nil {
		if cleanupErr := net.Cleanup(); cleanupErr != nil {
			fmt.Printf("Warning: failed to cleanup network: %v\n", cleanupErr)
		}
		return nil, errors.Wrap("Bootstrap.GenerateCrypto", err)
	}

	// Check context
	if err := ctx.Err(); err != nil {
		if cleanupErr := net.Cleanup(); cleanupErr != nil {
			fmt.Printf("Warning: failed to cleanup network: %v\n", cleanupErr)
		}
		return nil, errors.Wrap("Bootstrap", err)
	}

	// Generate configtx.yaml
	if err := generateConfigTx(net); err != nil {
		if cleanupErr := net.Cleanup(); cleanupErr != nil {
			fmt.Printf("Warning: failed to cleanup network: %v\n", cleanupErr)
		}
		return nil, errors.Wrap("Bootstrap.GenerateConfigTx", err)
	}

	if err := generateCoreYAML(net); err != nil {
		if cleanupErr := net.Cleanup(); cleanupErr != nil {
			fmt.Printf("Warning: failed to cleanup network: %v\n", cleanupErr)
		}
		return nil, errors.Wrap("Bootstrap.GenerateCoreYAML", err)
	}

	// Generate genesis block
	if err := generateGenesisBlock(ctx, net); err != nil {
		if cleanupErr := net.Cleanup(); cleanupErr != nil {
			fmt.Printf("Warning: failed to cleanup network: %v\n", cleanupErr)
		}
		return nil, errors.Wrap("Bootstrap.GenerateGenesisBlock", err)
	}

	// Generate channel configuration
	if err := generateChannelTx(ctx, net); err != nil {
		if cleanupErr := net.Cleanup(); cleanupErr != nil {
			fmt.Printf("Warning: failed to cleanup network: %v\n", cleanupErr)
		}
		return nil, errors.Wrap("Bootstrap.GenerateChannelTx", err)
	}

	// Generate docker-compose
	if err := generateDockerCompose(net); err != nil {
		if cleanupErr := net.Cleanup(); cleanupErr != nil {
			fmt.Printf("Warning: failed to cleanup network: %v\n", cleanupErr)
		}
		return nil, errors.Wrap("Bootstrap.GenerateDockerCompose", err)
	}

	return net, nil
}

func generateOrganizations(numOrgs int) []*Organization {
	orgs := make([]*Organization, numOrgs)
	basePort := 7051

	for i := 0; i < numOrgs; i++ {
		orgName := fmt.Sprintf("Org%d", i+1)
		orgs[i] = &Organization{
			Name:       orgName,
			MSPID:      fmt.Sprintf("%sMSP", orgName),
			Domain:     fmt.Sprintf("org%d.example.com", i+1),
			CAPort:     7054 + (i * 1000),
			AnchorPort: basePort + (i * 1000),
			Peers: []*Peer{
				{
					Name:    fmt.Sprintf("peer0.org%d.example.com", i+1),
					Port:    basePort + (i * 1000),
					CouchDB: true,
					DBPort:  5984 + (i * 1000),
				},
			},
		}
	}

	return orgs
}

func generateOrderers() []*Orderer {
	return []*Orderer{
		{
			Name:   "orderer.example.com",
			Port:   7050,
			Domain: "example.com",
		},
	}
}

func (n *Network) WaitForReady(ctx context.Context) error {
	// Create a deadline context if not already set
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 120*time.Second)
		defer cancel()
	}

	// Wait for containers to be healthy
	fmt.Println("⏳ Waiting for containers to be ready...")
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	startTime := time.Now()
	timeout := 30 * time.Second

	for {
		select {
		case <-ctx.Done():
			return errors.WrapWithContext("WaitForReady", errors.ErrTimeout, map[string]interface{}{
				"network_id": n.ID,
			})
		case <-ticker.C:
			if time.Since(startTime) > timeout {
				fmt.Println("✓ Containers should be ready now")
				goto CHANNEL_SETUP
			}

			// Check container health
			if n.checkReadiness(ctx) {
				fmt.Println("✓ Containers are healthy")
				goto CHANNEL_SETUP
			}
		}
	}

CHANNEL_SETUP:
	// Create channel
	if err := n.CreateChannel(ctx); err != nil {
		return errors.Wrap("WaitForReady.CreateChannel", err)
	}

	// Join peers to channel
	if err := n.JoinPeersToChannel(ctx); err != nil {
		return errors.Wrap("WaitForReady.JoinPeers", err)
	}

	// Update anchor peers (non-critical)
	if err := n.UpdateAnchorPeers(ctx); err != nil {
		fmt.Printf("Warning: Could not update anchor peers: %v\n", err)
		// Don't fail on anchor peer update
	}

	fmt.Println("✅ Network is fully ready!")
	return nil
}

func (n *Network) checkReadiness(ctx context.Context) bool {
	// Check if context is cancelled
	if ctx.Err() != nil {
		return false
	}

	// Check if orderer is responsive
	output, err := n.exec.ExecuteCombined(ctx, "docker", "exec", n.Orderers[0].Name,
		"sh", "-c", "echo 'test' > /dev/null")
	if err != nil {
		return false
	}
	_ = output

	// Check if at least one peer is responsive
	for _, org := range n.Orgs {
		for _, peer := range org.Peers {
			output, err := n.exec.ExecuteCombined(ctx, "docker", "exec", peer.Name,
				"sh", "-c", "echo 'test' > /dev/null")
			if err == nil {
				_ = output
				return true
			}
		}
	}

	return false
}

func (n *Network) GetEndpoints() []string {
	endpoints := []string{}
	for _, org := range n.Orgs {
		for _, peer := range org.Peers {
			endpoints = append(endpoints, fmt.Sprintf("localhost:%d", peer.Port))
		}
	}
	return endpoints
}

// Interface methods for docker.Manager
func (n *Network) GetID() string {
	return n.ID
}

func (n *Network) GetConfigPath() string {
	return n.ConfigPath
}

func (n *Network) GetOrgs() interface{} {
	return n.Orgs
}

func (n *Network) GetOrderers() interface{} {
	return n.Orderers
}

func (n *Network) GetConnectionProfile(orgName string) (map[string]interface{}, error) {
	// Generate connection profile for SDK
	profile := map[string]interface{}{
		"name":    fmt.Sprintf("%s-network", n.Name),
		"version": "1.0.0",
		"client": map[string]interface{}{
			"organization": orgName,
		},
		"channels": map[string]interface{}{
			n.Channel.Name: map[string]interface{}{
				"orderers": []string{n.Orderers[0].Name},
				"peers":    map[string]interface{}{},
			},
		},
		"organizations": map[string]interface{}{},
		"orderers":      map[string]interface{}{},
		"peers":         map[string]interface{}{},
	}

	// Add organizations
	for _, org := range n.Orgs {
		profile["organizations"].(map[string]interface{})[org.Name] = map[string]interface{}{
			"mspid": org.MSPID,
			"peers": []string{org.Peers[0].Name},
			"certificateAuthorities": []string{
				fmt.Sprintf("ca.%s", org.Domain),
			},
		}
	}

	// Add peers
	for _, org := range n.Orgs {
		for _, peer := range org.Peers {
			profile["peers"].(map[string]interface{})[peer.Name] = map[string]interface{}{
				"url": fmt.Sprintf("grpc://localhost:%d", peer.Port),
			}
		}
	}

	// Add orderers
	for _, orderer := range n.Orderers {
		profile["orderers"].(map[string]interface{})[orderer.Name] = map[string]interface{}{
			"url": fmt.Sprintf("grpc://localhost:%d", orderer.Port),
		}
	}

	return profile, nil
}

func (n *Network) Cleanup() error {
	if err := os.RemoveAll(n.BasePath); err != nil {
		return errors.WrapWithContext("Cleanup", err, map[string]interface{}{
			"base_path": n.BasePath,
		})
	}
	return nil
}
