// core/pkg/chaincode/invoker.go
package chaincode

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/temmyjay001/core/pkg/errors"
	"github.com/temmyjay001/core/pkg/executor"
	"github.com/temmyjay001/core/pkg/network"
)

type Invoker struct {
	network *network.Network
	exec    executor.Executor
}

func NewInvoker(net *network.Network, exec executor.Executor) *Invoker {
	return &Invoker{
		network: net,
		exec:    exec,
	}
}

// Invoke executes a transaction inside a peer container (no local binaries)
func (inv *Invoker) Invoke(ctx context.Context, chaincodeName, functionName string, args []string) (string, []byte, error) {
	// Check context
	if err := ctx.Err(); err != nil {
		return "", nil, errors.Wrap("Invoke", err)
	}

	// Use first org for invocation
	org := inv.network.Orgs[0]
	peer := org.Peers[0]
	containerName := "cli"

	// Build arguments JSON
	argsJSON := inv.buildArgsJSON(functionName, args)

	// Build peer addresses for endorsement
	peerAddresses := []string{}
	peerTLSRootCerts := []string{}
	for _, org := range inv.network.Orgs {
		for _, peer := range org.Peers {
			peerAddresses = append(peerAddresses, "--peerAddresses", fmt.Sprintf("%s:%d", peer.Name, peer.Port))
			tlsCert := fmt.Sprintf("/etc/hyperledger/fabric/crypto/peerOrganizations/%s/peers/%s/tls/ca.crt", org.Domain, peer.Name)
			peerTLSRootCerts = append(peerTLSRootCerts, "--tlsRootCertFiles", tlsCert)
		}
	}

	// Execute invoke inside CLI container
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
		"--tls", "true",
		"--cafile", ordererTLSCA,
	)
	cmdArgs = append(cmdArgs, peerAddresses...)
	cmdArgs = append(cmdArgs, peerTLSRootCerts...)

	output, err := inv.exec.ExecuteCombined(ctx, "docker", cmdArgs...)
	if err != nil {
		return "", nil, errors.WrapWithContext("Invoke", errors.ErrTransactionFailed, map[string]interface{}{
			"chaincode": chaincodeName,
			"function":  functionName,
			"error":     err.Error(),
			"output":    string(output),
		})
	}

	// Extract transaction ID from output
	txID := inv.extractTxID(string(output))

	// Clean up the output - remove ANSI codes and extract payload if present
	cleanOutput := inv.cleanOutput(string(output))

	// Try to extract result payload from the output
	payload := inv.extractPayload(cleanOutput)

	return txID, payload, nil
}

// Query executes a read-only query inside a peer container
func (inv *Invoker) Query(ctx context.Context, chaincodeName, functionName string, args []string) ([]byte, error) {
	// Check context
	if err := ctx.Err(); err != nil {
		return nil, errors.Wrap("Query", err)
	}

	// Use first org for query
	org := inv.network.Orgs[0]
	peer := org.Peers[0]
	containerName := "cli"

	// Build arguments JSON
	argsJSON := inv.buildArgsJSON(functionName, args)

	// Execute query inside CLI container
	env := inv.getPeerEnvArgs(org, peer)
	cmdArgs := []string{"exec"}
	cmdArgs = append(cmdArgs, env...)
	cmdArgs = append(cmdArgs, containerName,
		"peer", "chaincode", "query",
		"-C", inv.network.Channel.Name,
		"-n", chaincodeName,
		"-c", argsJSON,
		"--tls", "true",
		"--cafile", ordererTLSCA,
	)

	output, err := inv.exec.ExecuteCombined(ctx, "docker", cmdArgs...)
	if err != nil {
		return nil, errors.WrapWithContext("Query", err, map[string]interface{}{
			"chaincode": chaincodeName,
			"function":  functionName,
			"error":     err.Error(),
			"output":    string(output),
		})
	}

	// Clean the output
	cleanOutput := inv.cleanOutput(string(output))

	// For queries, the last line typically contains the result
	lines := strings.Split(strings.TrimSpace(cleanOutput), "\n")
	if len(lines) > 0 {
		result := lines[len(lines)-1]
		// If it looks like JSON, return it as-is
		if strings.HasPrefix(strings.TrimSpace(result), "{") || strings.HasPrefix(strings.TrimSpace(result), "[") {
			return []byte(result), nil
		}
	}

	return []byte(cleanOutput), nil
}

// InvokeWithTransient executes a transaction with transient data
func (inv *Invoker) InvokeWithTransient(ctx context.Context, chaincodeName, functionName string, args []string, transient map[string][]byte) (string, []byte, error) {
	// Check context
	if err := ctx.Err(); err != nil {
		return "", nil, errors.Wrap("InvokeWithTransient", err)
	}

	org := inv.network.Orgs[0]
	peer := org.Peers[0]
	containerName := "cli"

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

	output, err := inv.exec.ExecuteCombined(ctx, "docker", cmdArgs...)
	if err != nil {
		return "", nil, errors.WrapWithContext("InvokeWithTransient", errors.ErrTransactionFailed, map[string]interface{}{
			"chaincode": chaincodeName,
			"function":  functionName,
			"error":     err.Error(),
			"output":    string(output),
		})
	}

	txID := inv.extractTxID(string(output))
	cleanOutput := inv.cleanOutput(string(output))
	payload := inv.extractPayload(cleanOutput)

	return txID, payload, nil
}

func (inv *Invoker) getPeerEnvArgs(org *network.Organization, peer *network.Peer) []string {
	return []string{
		"-e", fmt.Sprintf("CORE_PEER_LOCALMSPID=%s", org.MSPID),
		"-e", fmt.Sprintf("CORE_PEER_ADDRESS=%s:%d", peer.Name, peer.Port),
		"-e", fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=/etc/hyperledger/fabric/crypto/peerOrganizations/%s/users/Admin@%s/msp", org.Domain, org.Domain),
		"-e", "CORE_PEER_TLS_ENABLED=true",
		"-e", fmt.Sprintf("CORE_PEER_TLS_ROOTCERT_FILE=/etc/hyperledger/fabric/crypto/peerOrganizations/%s/peers/%s/tls/ca.crt", org.Domain, peer.Name),
		"-e", "FABRIC_CFG_PATH=/etc/hyperledger/fabric/config",
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
	// Format: "... txid [TRANSACTION_ID] committed with status ..."
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "txid") && strings.Contains(line, "committed") {
			// Use regex to extract txid
			re := regexp.MustCompile(`txid\s+\[([a-zA-Z0-9]+)\]`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				return matches[1]
			}
		}
	}
	return "unknown"
}

// cleanOutput removes ANSI escape codes and cleans up docker output
func (inv *Invoker) cleanOutput(output string) string {
	// Remove ANSI escape codes (color codes)
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	cleaned := ansiRegex.ReplaceAllString(output, "")

	// Remove timestamp prefixes like "2025-11-11 02:52:58.506 UTC 0001 INFO"
	timestampRegex := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2}\.\d+\s+UTC\s+\d+\s+INFO\s+`)
	lines := strings.Split(cleaned, "\n")
	var cleanedLines []string
	for _, line := range lines {
		line = timestampRegex.ReplaceAllString(line, "")
		if strings.TrimSpace(line) != "" {
			cleanedLines = append(cleanedLines, line)
		}
	}

	return strings.Join(cleanedLines, "\n")
}

// extractPayload tries to extract the result payload from chaincode output
func (inv *Invoker) extractPayload(output string) []byte {
	// Look for "result: status:200 payload:..." pattern
	re := regexp.MustCompile(`result:\s*status:(\d+)\s+(?:payload:"?([^"]*)"?)?`)
	matches := re.FindStringSubmatch(output)

	if len(matches) > 2 && matches[2] != "" {
		// Found payload
		return []byte(matches[2])
	}

	// Look for lines that might contain chaincode response
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		// Skip lines that are clearly log messages
		if strings.Contains(line, "INFO") ||
			strings.Contains(line, "ClientWait") ||
			strings.Contains(line, "committed with status") {
			continue
		}

		// If line looks like JSON, return it
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
			return []byte(trimmed)
		}
	}

	// Return empty if no payload found (for transactions that don't return data)
	return []byte{}
}

// Helper to get block info
func (inv *Invoker) GetBlockByNumber(ctx context.Context, blockNum uint64) ([]byte, error) {
	// Check context
	if err := ctx.Err(); err != nil {
		return nil, errors.Wrap("GetBlockByNumber", err)
	}

	org := inv.network.Orgs[0]
	peer := org.Peers[0]
	containerName := "cli"

	env := inv.getPeerEnvArgs(org, peer)
	cmdArgs := []string{"exec"}
	cmdArgs = append(cmdArgs, env...)
	cmdArgs = append(cmdArgs, containerName,
		"peer", "channel", "getinfo",
		"-c", inv.network.Channel.Name,
	)

	output, err := inv.exec.ExecuteCombined(ctx, "docker", cmdArgs...)
	if err != nil {
		return nil, errors.WrapWithContext("GetBlockByNumber", err, map[string]interface{}{
			"block_num": blockNum,
			"error":     err.Error(),
			"output":    string(output),
		})
	}

	cleanOutput := inv.cleanOutput(string(output))
	return []byte(cleanOutput), nil
}

// Helper to get transaction by ID
func (inv *Invoker) GetTransactionByID(ctx context.Context, txID string) ([]byte, error) {
	// Check context
	if err := ctx.Err(); err != nil {
		return nil, errors.Wrap("GetTransactionByID", err)
	}

	org := inv.network.Orgs[0]
	peer := org.Peers[0]
	containerName := "cli"

	env := inv.getPeerEnvArgs(org, peer)
	cmdArgs := []string{"exec"}
	cmdArgs = append(cmdArgs, env...)
	cmdArgs = append(cmdArgs, containerName,
		"peer", "chaincode", "query",
		"-C", inv.network.Channel.Name,
		"-n", "qscc",
		"-c", fmt.Sprintf(`{"Args":["GetTransactionByID","%s","%s"]}`, inv.network.Channel.Name, txID),
	)

	output, err := inv.exec.ExecuteCombined(ctx, "docker", cmdArgs...)
	if err != nil {
		return nil, errors.WrapWithContext("GetTransactionByID", err, map[string]interface{}{
			"tx_id":  txID,
			"error":  err.Error(),
			"output": string(output),
		})
	}

	cleanOutput := inv.cleanOutput(string(output))
	return []byte(cleanOutput), nil
}
