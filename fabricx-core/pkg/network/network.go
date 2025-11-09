// fabricx-core/pkg/network/network.go
package network

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/temmyjay001/fabricx-core/pkg/errors"
	"github.com/temmyjay001/fabricx-core/pkg/executor"
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
func Bootstrap(config *Config) (*Network, error) {
	return BootstrapWithContext(context.Background(), config)
}

// BootstrapWithContext creates a network with context support
func BootstrapWithContext(ctx context.Context, config *Config) (*Network, error) {
	return BootstrapWithExecutor(ctx, config, executor.NewRealExecutor())
}

// BootstrapWithExecutor creates a network with custom executor (for testing)
func BootstrapWithExecutor(ctx context.Context, config *Config, exec executor.Executor) (*Network, error) {
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
		net.Cleanup()
		return nil, errors.Wrap("Bootstrap", err)
	}

	// Generate crypto material
	if err := generateCrypto(ctx, net); err != nil {
		net.Cleanup()
		return nil, errors.Wrap("Bootstrap.GenerateCrypto", err)
	}

	// Check context
	if err := ctx.Err(); err != nil {
		net.Cleanup()
		return nil, errors.Wrap("Bootstrap", err)
	}

	// Generate configtx.yaml
	if err := generateConfigTx(net); err != nil {
		net.Cleanup()
		return nil, errors.Wrap("Bootstrap.GenerateConfigTx", err)
	}

	// Generate genesis block
	if err := generateGenesisBlock(ctx, net); err != nil {
		net.Cleanup()
		return nil, errors.Wrap("Bootstrap.GenerateGenesisBlock", err)
	}

	// Generate channel configuration
	if err := generateChannelTx(ctx, net); err != nil {
		net.Cleanup()
		return nil, errors.Wrap("Bootstrap.GenerateChannelTx", err)
	}

	// Generate docker-compose
	if err := generateDockerCompose(net); err != nil {
		net.Cleanup()
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

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return errors.WrapWithContext("WaitForReady", errors.ErrTimeout, map[string]interface{}{
				"network_id": n.ID,
			})
		case <-ticker.C:
			if n.checkReadiness(ctx) {
				return nil
			}
		}
	}
}

func (n *Network) checkReadiness(ctx context.Context) bool {
	// Check if context is cancelled
	if ctx.Err() != nil {
		return false
	}

	// In production, this would check actual container health
	// For now, we assume network is ready after creation
	return true
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

// Helper to get executor
func (n *Network) GetExecutor() executor.Executor {
	if n.exec == nil {
		n.exec = executor.NewRealExecutor()
	}
	return n.exec
}