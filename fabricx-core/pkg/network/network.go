// pkg/network/network.go
package network

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
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

func Bootstrap(config *Config) (*Network, error) {
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
		return nil, fmt.Errorf("failed to create base directory: %w", err)
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
	}

	// Generate organizations
	net.Orgs = generateOrganizations(config.NumOrgs)

	// Generate orderers
	net.Orderers = generateOrderers()

	// Generate crypto material
	if err := generateCrypto(net); err != nil {
		return nil, fmt.Errorf("failed to generate crypto: %w", err)
	}

	// Generate configtx.yaml
	if err := generateConfigTx(net); err != nil {
		return nil, fmt.Errorf("failed to generate configtx: %w", err)
	}

	// Generate genesis block
	if err := generateGenesisBlock(net); err != nil {
		return nil, fmt.Errorf("failed to generate genesis block: %w", err)
	}

	// Generate channel configuration
	if err := generateChannelTx(net); err != nil {
		return nil, fmt.Errorf("failed to generate channel tx: %w", err)
	}

	// Generate docker-compose
	if err := generateDockerCompose(net); err != nil {
		return nil, fmt.Errorf("failed to generate docker-compose: %w", err)
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
	timeout := time.After(120 * time.Second)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for network to be ready")
		case <-ticker.C:
			if n.checkReadiness() {
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (n *Network) checkReadiness() bool {
	// Check if all containers are running (simplified)
	// In production, this would check actual container health
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
	return os.RemoveAll(n.BasePath)
}
