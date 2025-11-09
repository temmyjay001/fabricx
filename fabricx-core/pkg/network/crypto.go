// fabricx-core/pkg/network/crypto.go
package network

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/temmyjay001/fabricx-core/pkg/errors"
	"github.com/temmyjay001/fabricx-core/pkg/utils"
)

const (
	fabricToolsImage = "hyperledger/fabric-tools:2.5"
)

// generateCrypto uses Docker to run cryptogen (no local binaries needed)
func generateCrypto(ctx context.Context, net *Network) error {
	// Generate crypto-config.yaml
	cryptoConfigPath := filepath.Join(net.ConfigPath, "crypto-config.yaml")
	if err := utils.EnsureDir(filepath.Dir(cryptoConfigPath)); err != nil {
		return errors.Wrap("generateCrypto.EnsureDir", err)
	}

	cryptoConfig := generateCryptoConfig(net)
	if err := utils.WriteYAML(cryptoConfigPath, cryptoConfig); err != nil {
		return errors.Wrap("generateCrypto.WriteYAML", err)
	}

	// Check context before long operation
	if err := ctx.Err(); err != nil {
		return errors.Wrap("generateCrypto", err)
	}

	// Run cryptogen inside Docker container
	exec := net.exec
	output, err := exec.ExecuteCombined(ctx, "docker", "run", "--rm",
		"-v", fmt.Sprintf("%s:/config", net.ConfigPath),
		"-v", fmt.Sprintf("%s:/crypto-config", net.CryptoPath),
		fabricToolsImage,
		"cryptogen", "generate",
		"--config=/config/crypto-config.yaml",
		"--output=/crypto-config",
	)

	if err != nil {
		return errors.WrapWithContext("generateCrypto", errors.ErrCryptoGenFailed, map[string]interface{}{
			"error":  err.Error(),
			"output": string(output),
		})
	}

	return nil
}

func generateCryptoConfig(net *Network) map[string]interface{} {
	config := map[string]interface{}{
		"OrdererOrgs": []map[string]interface{}{
			{
				"Name":   "Orderer",
				"Domain": "example.com",
				"Specs": []map[string]interface{}{
					{
						"Hostname": "orderer",
					},
				},
			},
		},
		"PeerOrgs": []map[string]interface{}{},
	}

	// Add peer organizations
	peerOrgs := []map[string]interface{}{}
	for _, org := range net.Orgs {
		peerOrg := map[string]interface{}{
			"Name":          org.Name,
			"Domain":        org.Domain,
			"EnableNodeOUs": true,
			"Template": map[string]interface{}{
				"Count": len(org.Peers),
			},
			"Users": map[string]interface{}{
				"Count": 1,
			},
		}
		peerOrgs = append(peerOrgs, peerOrg)
	}
	config["PeerOrgs"] = peerOrgs

	return config
}

func generateConfigTx(net *Network) error {
	configTxPath := filepath.Join(net.ConfigPath, "configtx.yaml")

	configTx := generateConfigTxYAML(net)
	if err := utils.WriteYAML(configTxPath, configTx); err != nil {
		return errors.Wrap("generateConfigTx", err)
	}

	return nil
}

func generateConfigTxYAML(net *Network) map[string]interface{} {
	// Organizations - these are the full definitions
	organizations := []map[string]interface{}{
		{
			"Name":   "OrdererOrg",
			"ID":     "OrdererMSP",
			"MSPDir": "/crypto-config/ordererOrganizations/example.com/msp",
			"Policies": map[string]interface{}{
				"Readers": map[string]interface{}{
					"Type": "Signature",
					"Rule": "OR('OrdererMSP.member')",
				},
				"Writers": map[string]interface{}{
					"Type": "Signature",
					"Rule": "OR('OrdererMSP.member')",
				},
				"Admins": map[string]interface{}{
					"Type": "Signature",
					"Rule": "OR('OrdererMSP.admin')",
				},
			},
		},
	}

	// Add peer organizations with full definitions
	for _, org := range net.Orgs {
		peerOrg := map[string]interface{}{
			"Name":   org.Name,
			"ID":     org.MSPID,
			"MSPDir": fmt.Sprintf("/crypto-config/peerOrganizations/%s/msp", org.Domain),
			"Policies": map[string]interface{}{
				"Readers": map[string]interface{}{
					"Type": "Signature",
					"Rule": fmt.Sprintf("OR('%s.admin', '%s.peer', '%s.client')", org.MSPID, org.MSPID, org.MSPID),
				},
				"Writers": map[string]interface{}{
					"Type": "Signature",
					"Rule": fmt.Sprintf("OR('%s.admin', '%s.client')", org.MSPID, org.MSPID),
				},
				"Admins": map[string]interface{}{
					"Type": "Signature",
					"Rule": fmt.Sprintf("OR('%s.admin')", org.MSPID),
				},
				"Endorsement": map[string]interface{}{
					"Type": "Signature",
					"Rule": fmt.Sprintf("OR('%s.peer')", org.MSPID),
				},
			},
			"AnchorPeers": []map[string]interface{}{
				{
					"Host": org.Peers[0].Name,
					"Port": org.Peers[0].Port,
				},
			},
		}
		organizations = append(organizations, peerOrg)
	}

	// Capabilities
	capabilities := map[string]interface{}{
		"Channel": map[string]interface{}{
			"V2_0": true,
		},
		"Orderer": map[string]interface{}{
			"V2_0": true,
		},
		"Application": map[string]interface{}{
			"V2_0": true,
		},
	}

	// Application defaults
	application := map[string]interface{}{
		"Organizations": nil,
		"Policies": map[string]interface{}{
			"Readers": map[string]interface{}{
				"Type": "ImplicitMeta",
				"Rule": "ANY Readers",
			},
			"Writers": map[string]interface{}{
				"Type": "ImplicitMeta",
				"Rule": "ANY Writers",
			},
			"Admins": map[string]interface{}{
				"Type": "ImplicitMeta",
				"Rule": "MAJORITY Admins",
			},
			"LifecycleEndorsement": map[string]interface{}{
				"Type": "ImplicitMeta",
				"Rule": "MAJORITY Endorsement",
			},
			"Endorsement": map[string]interface{}{
				"Type": "ImplicitMeta",
				"Rule": "MAJORITY Endorsement",
			},
		},
		"Capabilities": map[string]interface{}{
			"V2_0": true,
		},
	}

	// Orderer defaults
	orderer := map[string]interface{}{
		"OrdererType": "solo",
		"Addresses": []string{
			fmt.Sprintf("%s:%d", net.Orderers[0].Name, net.Orderers[0].Port),
		},
		"BatchTimeout": "2s",
		"BatchSize": map[string]interface{}{
			"MaxMessageCount":   10,
			"AbsoluteMaxBytes":  "99 MB",
			"PreferredMaxBytes": "512 KB",
		},
		"Organizations": nil,
		"Policies": map[string]interface{}{
			"Readers": map[string]interface{}{
				"Type": "ImplicitMeta",
				"Rule": "ANY Readers",
			},
			"Writers": map[string]interface{}{
				"Type": "ImplicitMeta",
				"Rule": "ANY Writers",
			},
			"Admins": map[string]interface{}{
				"Type": "ImplicitMeta",
				"Rule": "MAJORITY Admins",
			},
			"BlockValidation": map[string]interface{}{
				"Type": "ImplicitMeta",
				"Rule": "ANY Writers",
			},
		},
	}

	// Channel defaults
	channel := map[string]interface{}{
		"Policies": map[string]interface{}{
			"Readers": map[string]interface{}{
				"Type": "ImplicitMeta",
				"Rule": "ANY Readers",
			},
			"Writers": map[string]interface{}{
				"Type": "ImplicitMeta",
				"Rule": "ANY Writers",
			},
			"Admins": map[string]interface{}{
				"Type": "ImplicitMeta",
				"Rule": "MAJORITY Admins",
			},
		},
		"Capabilities": map[string]interface{}{
			"V2_0": true,
		},
	}

	ordererOrg := organizations[0]
	peerOrgs := organizations[1:]

	profiles := map[string]interface{}{
		"FabricXOrdererGenesis": map[string]interface{}{
			"Orderer": map[string]interface{}{
				"OrdererType": "solo",
				"Addresses": []string{
					fmt.Sprintf("%s:%d", net.Orderers[0].Name, net.Orderers[0].Port),
				},
				"BatchTimeout": "2s",
				"BatchSize": map[string]interface{}{
					"MaxMessageCount":   10,
					"AbsoluteMaxBytes":  "99 MB",
					"PreferredMaxBytes": "512 KB",
				},
				"Organizations": []interface{}{ordererOrg},
				"Policies": map[string]interface{}{
					"Readers": map[string]interface{}{
						"Type": "ImplicitMeta",
						"Rule": "ANY Readers",
					},
					"Writers": map[string]interface{}{
						"Type": "ImplicitMeta",
						"Rule": "ANY Writers",
					},
					"Admins": map[string]interface{}{
						"Type": "ImplicitMeta",
						"Rule": "MAJORITY Admins",
					},
					"BlockValidation": map[string]interface{}{
						"Type": "ImplicitMeta",
						"Rule": "ANY Writers",
					},
				},
				"Capabilities": map[string]interface{}{
					"V2_0": true,
				},
			},
			"Consortiums": map[string]interface{}{
				"FabricXConsortium": map[string]interface{}{
					"Organizations": peerOrgs,
				},
			},
			"Capabilities": map[string]interface{}{
				"V2_0": true,
			},
			"Policies": map[string]interface{}{
				"Readers": map[string]interface{}{
					"Type": "ImplicitMeta",
					"Rule": "ANY Readers",
				},
				"Writers": map[string]interface{}{
					"Type": "ImplicitMeta",
					"Rule": "ANY Writers",
				},
				"Admins": map[string]interface{}{
					"Type": "ImplicitMeta",
					"Rule": "MAJORITY Admins",
				},
			},
		},
		net.Channel.ProfileName: map[string]interface{}{
			"Consortium": "FabricXConsortium",
			"Policies": map[string]interface{}{
				"Readers": map[string]interface{}{
					"Type": "ImplicitMeta",
					"Rule": "ANY Readers",
				},
				"Writers": map[string]interface{}{
					"Type": "ImplicitMeta",
					"Rule": "ANY Writers",
				},
				"Admins": map[string]interface{}{
					"Type": "ImplicitMeta",
					"Rule": "MAJORITY Admins",
				},
			},
			"Capabilities": map[string]interface{}{
				"V2_0": true,
			},
			"Application": map[string]interface{}{
				"Organizations": peerOrgs,
				"Capabilities": map[string]interface{}{
					"V2_0": true,
				},
				"Policies": map[string]interface{}{
					"Readers": map[string]interface{}{
						"Type": "ImplicitMeta",
						"Rule": "ANY Readers",
					},
					"Writers": map[string]interface{}{
						"Type": "ImplicitMeta",
						"Rule": "ANY Writers",
					},
					"Admins": map[string]interface{}{
						"Type": "ImplicitMeta",
						"Rule": "MAJORITY Admins",
					},
					"LifecycleEndorsement": map[string]interface{}{
						"Type": "ImplicitMeta",
						"Rule": "MAJORITY Endorsement",
					},
					"Endorsement": map[string]interface{}{
						"Type": "ImplicitMeta",
						"Rule": "MAJORITY Endorsement",
					},
				},
			},
		},
	}

	return map[string]interface{}{
		"Organizations": organizations,
		"Capabilities":  capabilities,
		"Application":   application,
		"Orderer":       orderer,
		"Channel":       channel,
		"Profiles":      profiles,
	}
}

// generateGenesisBlock uses Docker to run configtxgen
func generateGenesisBlock(ctx context.Context, net *Network) error {
	// Check context
	if err := ctx.Err(); err != nil {
		return errors.Wrap("generateGenesisBlock", err)
	}

	exec := net.exec
	output, err := exec.ExecuteCombined(ctx, "docker", "run", "--rm",
		"-v", fmt.Sprintf("%s:/config", net.ConfigPath),
		"-v", fmt.Sprintf("%s:/crypto-config", net.CryptoPath),
		"-e", "FABRIC_CFG_PATH=/config",
		fabricToolsImage,
		"configtxgen",
		"-profile", "FabricXOrdererGenesis",
		"-channelID", "system-channel",
		"-outputBlock", "/config/genesis.block",
	)

	if err != nil {
		return errors.WrapWithContext("generateGenesisBlock", err, map[string]interface{}{
			"output": string(output),
		})
	}

	return nil
}

// generateChannelTx uses Docker to run configtxgen
func generateChannelTx(ctx context.Context, net *Network) error {
	// Check context
	if err := ctx.Err(); err != nil {
		return errors.Wrap("generateChannelTx", err)
	}

	channelTxPath := fmt.Sprintf("%s.tx", net.Channel.Name)

	exec := net.exec
	output, err := exec.ExecuteCombined(ctx, "docker", "run", "--rm",
		"-v", fmt.Sprintf("%s:/config", net.ConfigPath),
		"-v", fmt.Sprintf("%s:/crypto-config", net.CryptoPath),
		"-e", "FABRIC_CFG_PATH=/config",
		fabricToolsImage,
		"configtxgen",
		"-profile", net.Channel.ProfileName,
		"-outputCreateChannelTx", fmt.Sprintf("/config/%s", channelTxPath),
		"-channelID", net.Channel.Name,
	)

	if err != nil {
		return errors.WrapWithContext("generateChannelTx", err, map[string]interface{}{
			"output": string(output),
		})
	}

	return nil
}