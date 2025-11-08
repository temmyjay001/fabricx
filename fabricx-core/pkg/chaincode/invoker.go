// pkg/chaincode/invoker.go
package chaincode

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/temmyjay001/fabricx-core/pkg/network"
)

type Invoker struct {
	network *network.Network
}

func NewInvoker(net *network.Network) *Invoker {
	return &Invoker{
		network: net,
	}
}

// Invoke executes a transaction inside a peer container (no local binaries)
func (inv *Invoker) Invoke(ctx context.Context, chaincodeName, functionName string, args []string) (string, []byte, error) {
	// Use first org for invocation
	org := inv.network.Orgs[0]
	peer := org.Peers[0]
	containerName := peer.Name

	// Build arguments JSON
	argsJSON := inv.buildArgsJSON(functionName, args)

	// Build peer addresses for endorsement
	peerAddresses := []string{}
	for _, org := range inv.network.Orgs {
		for _, peer := range org.Peers {
			peerAddresses = append(peerAddresses, "--peerAddresses", fmt.Sprintf("%s:%d", peer.Name, peer.Port))
		}
	}

	// Execute invoke inside peer container
	env := inv.getPeerEnvArgs(org, peer)
	cmdArgs := []string{"exec"}
	cmdArgs = append(cmdArgs, env...)
	cmdArgs = append(cmdArgs, containerName,
		"peer", "chaincode", "invoke",
		"-o", fmt.Sprintf("%s:%d", inv.network.Orderers[0].Name, inv.network.Orderers[0].Port),
		"-C", inv.network.Channel.Name,
		"-n", chaincodeName,
		"-c", argsJSON,
		"--waitForEvent",
	)
	cmdArgs = append(cmdArgs, peerAddresses...)

	cmd := exec.CommandContext(ctx, "docker", cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", nil, fmt.Errorf("invoke failed: %w\nOutput: %s", err, string(output))
	}

	// Extract transaction ID from output
	txID := inv.extractTxID(string(output))

	return txID, output, nil
}

// Query executes a read-only query inside a peer container
func (inv *Invoker) Query(ctx context.Context, chaincodeName, functionName string, args []string) ([]byte, error) {
	// Use first org for query
	org := inv.network.Orgs[0]
	peer := org.Peers[0]
	containerName := peer.Name

	// Build arguments JSON
	argsJSON := inv.buildArgsJSON(functionName, args)

	// Execute query inside peer container
	env := inv.getPeerEnvArgs(org, peer)
	cmdArgs := []string{"exec"}
	cmdArgs = append(cmdArgs, env...)
	cmdArgs = append(cmdArgs, containerName,
		"peer", "chaincode", "query",
		"-C", inv.network.Channel.Name,
		"-n", chaincodeName,
		"-c", argsJSON,
	)

	cmd := exec.CommandContext(ctx, "docker", cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("query failed: %w\nOutput: %s", err, string(output))
	}

	return output, nil
}

// InvokeWithTransient executes a transaction with transient data
func (inv *Invoker) InvokeWithTransient(ctx context.Context, chaincodeName, functionName string, args []string, transient map[string][]byte) (string, []byte, error) {
	org := inv.network.Orgs[0]
	peer := org.Peers[0]
	containerName := peer.Name

	argsJSON := inv.buildArgsJSON(functionName, args)

	// For transient data, we'd need to write it to a file in the container
	// This is simplified - production would handle this more robustly
	transientJSON, _ := json.Marshal(transient)

	peerAddresses := []string{}
	for _, org := range inv.network.Orgs {
		for _, peer := range org.Peers {
			peerAddresses = append(peerAddresses, "--peerAddresses", fmt.Sprintf("%s:%d", peer.Name, peer.Port))
		}
	}

	env := inv.getPeerEnvArgs(org, peer)
	cmdArgs := []string{"exec"}
	cmdArgs = append(cmdArgs, env...)
	cmdArgs = append(cmdArgs, containerName,
		"peer", "chaincode", "invoke",
		"-o", fmt.Sprintf("%s:%d", inv.network.Orderers[0].Name, inv.network.Orderers[0].Port),
		"-C", inv.network.Channel.Name,
		"-n", chaincodeName,
		"-c", argsJSON,
		"--transient", string(transientJSON),
		"--waitForEvent",
	)
	cmdArgs = append(cmdArgs, peerAddresses...)

	cmd := exec.CommandContext(ctx, "docker", cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", nil, fmt.Errorf("invoke failed: %w\nOutput: %s", err, string(output))
	}

	txID := inv.extractTxID(string(output))

	return txID, output, nil
}

func (inv *Invoker) getPeerEnvArgs(org *network.Organization, peer *network.Peer) []string {
	return []string{
		"-e", fmt.Sprintf("CORE_PEER_LOCALMSPID=%s", org.MSPID),
		"-e", fmt.Sprintf("CORE_PEER_ADDRESS=%s:%d", peer.Name, peer.Port),
		"-e", fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=/etc/hyperledger/fabric/users/Admin@%s/msp", org.Domain),
		"-e", "CORE_PEER_TLS_ENABLED=false",
	}
}

func (inv *Invoker) buildArgsJSON(functionName string, args []string) string {
	allArgs := append([]string{functionName}, args...)

	argsMap := map[string]interface{}{
		"Args": allArgs,
	}

	jsonBytes, _ := json.Marshal(argsMap)
	return string(jsonBytes)
}

func (inv *Invoker) extractTxID(output string) string {
	// Parse transaction ID from output
	// Format: "... Chaincode invoke successful. result: status:200 payload:... txid:TRANSACTION_ID ..."
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "txid:") {
			parts := strings.Split(line, "txid:")
			if len(parts) > 1 {
				txID := strings.TrimSpace(strings.Split(parts[1], " ")[0])
				return txID
			}
		}
	}
	return "unknown"
}

// Helper to get block info
func (inv *Invoker) GetBlockByNumber(ctx context.Context, blockNum uint64) ([]byte, error) {
	org := inv.network.Orgs[0]
	peer := org.Peers[0]
	containerName := peer.Name

	env := inv.getPeerEnvArgs(org, peer)
	cmdArgs := []string{"exec"}
	cmdArgs = append(cmdArgs, env...)
	cmdArgs = append(cmdArgs, containerName,
		"peer", "channel", "getinfo",
		"-c", inv.network.Channel.Name,
	)

	cmd := exec.CommandContext(ctx, "docker", cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("getinfo failed: %w\nOutput: %s", err, string(output))
	}

	return output, nil
}

// Helper to get transaction by ID
func (inv *Invoker) GetTransactionByID(ctx context.Context, txID string) ([]byte, error) {
	org := inv.network.Orgs[0]
	peer := org.Peers[0]
	containerName := peer.Name

	env := inv.getPeerEnvArgs(org, peer)
	cmdArgs := []string{"exec"}
	cmdArgs = append(cmdArgs, env...)
	cmdArgs = append(cmdArgs, containerName,
		"peer", "chaincode", "query",
		"-C", inv.network.Channel.Name,
		"-n", "qscc",
		"-c", fmt.Sprintf(`{"Args":["GetTransactionByID","%s","%s"]}`, inv.network.Channel.Name, txID),
	)

	cmd := exec.CommandContext(ctx, "docker", cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("GetTransactionByID failed: %w\nOutput: %s", err, string(output))
	}

	return output, nil
}
