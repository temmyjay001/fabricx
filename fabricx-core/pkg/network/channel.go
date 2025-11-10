// pkg/network/channel.go
package network

import (
	"context"
	"fmt"
	"time"

	"github.com/temmyjay001/fabricx-core/pkg/errors"
)

// CreateChannel creates the channel using the CLI container
func (n *Network) CreateChannel(ctx context.Context) error {
	fmt.Println("📢 Creating channel...")

	// Check context
	if err := ctx.Err(); err != nil {
		return errors.Wrap("CreateChannel", err)
	}

	// Use first org for channel creation
	org := n.Orgs[0]
	ordererEndpoint := fmt.Sprintf("%s:%d", n.Orderers[0].Name, n.Orderers[0].Port)
	channelTxFile := fmt.Sprintf("/etc/hyperledger/fabric/config/%s.tx", n.Channel.Name)

	// Create channel using peer channel create in CLI container
	env := []string{
		"-e", fmt.Sprintf("CORE_PEER_LOCALMSPID=%s", org.MSPID),
		"-e", fmt.Sprintf("CORE_PEER_ADDRESS=%s:%d", org.Peers[0].Name, org.Peers[0].Port),
		"-e", fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=/etc/hyperledger/fabric/crypto/peerOrganizations/%s/users/Admin@%s/msp", org.Domain, org.Domain),
		"-e", "CORE_PEER_TLS_ENABLED=false",
	}

	args := []string{"exec"}
	args = append(args, env...)
	args = append(args, "cli",
		"peer", "channel", "create",
		"-o", ordererEndpoint,
		"-c", n.Channel.Name,
		"-f", channelTxFile,
		"--outputBlock", fmt.Sprintf("/etc/hyperledger/fabric/config/%s.block", n.Channel.Name),
	)

	output, err := n.exec.ExecuteCombined(ctx, "docker", args...)
	if err != nil {
		return errors.WrapWithContext("CreateChannel", err, map[string]interface{}{
			"channel": n.Channel.Name,
			"output":  string(output),
		})
	}

	fmt.Printf("✓ Channel '%s' created successfully\n", n.Channel.Name)

	// Wait a bit for channel to propagate
	time.Sleep(2 * time.Second)

	return nil
}

// JoinPeersToChannel joins all peers to the channel
func (n *Network) JoinPeersToChannel(ctx context.Context) error {
	fmt.Println("🔗 Joining peers to channel...")

	for _, org := range n.Orgs {
		for _, peer := range org.Peers {
			// Check context
			if err := ctx.Err(); err != nil {
				return errors.Wrap("JoinPeersToChannel", err)
			}

			fmt.Printf("   Joining %s to channel %s...\n", peer.Name, n.Channel.Name)

			// Execute join directly in the peer container using its own identity
			// The peer container now has the config directory mounted with the channel block
			args := []string{"exec", peer.Name,
				"peer", "channel", "join",
				"-b", fmt.Sprintf("/etc/hyperledger/fabric/config/%s.block", n.Channel.Name),
			}

			output, err := n.exec.ExecuteCombined(ctx, "docker", args...)
			if err != nil {
				// Log the full error for debugging
				fmt.Printf("   ⚠ Error: %s\n", string(output))
				
				return errors.WrapWithContext("JoinPeersToChannel", err, map[string]interface{}{
					"peer":    peer.Name,
					"org":     org.Name,
					"channel": n.Channel.Name,
					"output":  string(output),
				})
			}

			fmt.Printf("   ✓ %s joined channel\n", peer.Name)
			
			// Give the peer time to process the join
			time.Sleep(2 * time.Second)
		}
	}

	fmt.Println("✓ All peers joined to channel")
	return nil
}

// UpdateAnchorPeers updates anchor peers for each organization
func (n *Network) UpdateAnchorPeers(ctx context.Context) error {
	fmt.Println("⚓ Updating anchor peers...")

	for _, org := range n.Orgs {
		// Check context
		if err := ctx.Err(); err != nil {
			return errors.Wrap("UpdateAnchorPeers", err)
		}

		fmt.Printf("   Updating anchor peer for %s...\n", org.Name)

		// Generate anchor peer update transaction
		anchorTxFile := fmt.Sprintf("/etc/hyperledger/fabric/config/%sanchors.tx", org.Name)
		
		// First generate the anchor peer update tx using configtxgen
		configtxArgs := []string{"run", "--rm",
			"-v", fmt.Sprintf("%s:/config", n.ConfigPath),
			"-v", fmt.Sprintf("%s:/crypto-config", n.CryptoPath),
			"-e", "FABRIC_CFG_PATH=/config",
			fabricToolsImage,
			"configtxgen",
			"-profile", n.Channel.ProfileName,
			"-outputAnchorPeersUpdate", anchorTxFile,
			"-channelID", n.Channel.Name,
			"-asOrg", org.Name,
		}

		output, err := n.exec.ExecuteCombined(ctx, "docker", configtxArgs...)
		if err != nil {
			// Non-critical error, continue
			fmt.Printf("   Warning: Could not generate anchor peer update for %s: %v\n", org.Name, err)
			fmt.Printf("   Output: %s\n", string(output))
			continue
		}

		// Update the channel with anchor peer
		env := []string{
			"-e", fmt.Sprintf("CORE_PEER_LOCALMSPID=%s", org.MSPID),
			"-e", fmt.Sprintf("CORE_PEER_ADDRESS=%s:%d", org.Peers[0].Name, org.Peers[0].Port),
			"-e", fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=/etc/hyperledger/fabric/crypto/peerOrganizations/%s/users/Admin@%s/msp", org.Domain, org.Domain),
			"-e", "CORE_PEER_TLS_ENABLED=false",
		}

		args := []string{"exec"}
		args = append(args, env...)
		args = append(args, "cli",
			"peer", "channel", "update",
			"-o", fmt.Sprintf("%s:%d", n.Orderers[0].Name, n.Orderers[0].Port),
			"-c", n.Channel.Name,
			"-f", anchorTxFile,
		)

		output, err = n.exec.ExecuteCombined(ctx, "docker", args...)
		if err != nil {
			// Non-critical error, continue
			fmt.Printf("   Warning: Could not update anchor peer for %s: %v\n", org.Name, err)
			fmt.Printf("   Output: %s\n", string(output))
			continue
		}

		fmt.Printf("   ✓ Anchor peer updated for %s\n", org.Name)
	}

	fmt.Println("✓ Anchor peers updated")
	return nil
}