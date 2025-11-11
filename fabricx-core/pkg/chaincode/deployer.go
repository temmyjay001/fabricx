// fabricx-core/pkg/chaincode/deployer.go
package chaincode

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/temmyjay001/fabricx-core/pkg/docker"
	"github.com/temmyjay001/fabricx-core/pkg/errors"
	"github.com/temmyjay001/fabricx-core/pkg/executor"
	"github.com/temmyjay001/fabricx-core/pkg/network"
)

const (
	fabricToolsImage = "hyperledger/fabric-tools:2.5"
)

type Deployer struct {
	network   *network.Network
	dockerMgr *docker.Manager
	exec      executor.Executor
}

type DeployRequest struct {
	Name                  string
	Path                  string
	Version               string
	Language              string
	EndorsementPolicyOrgs []string
}

func NewDeployer(net *network.Network, dockerMgr *docker.Manager, exec executor.Executor) *Deployer {
	return &Deployer{
		network:   net,
		dockerMgr: dockerMgr,
		exec:      exec,
	}
}

func (d *Deployer) Deploy(ctx context.Context, req *DeployRequest) (string, error) {
	// Check context
	if err := ctx.Err(); err != nil {
		return "", errors.Wrap("Deploy", err)
	}

	// Set defaults
	if req.Version == "" {
		req.Version = "1.0"
	}
	if req.Language == "" {
		req.Language = "golang"
	}

	ccID := fmt.Sprintf("%s-%s", req.Name, uuid.New().String()[:8])

	// Package chaincode using Docker
	packageFile, err := d.packageChaincode(ctx, req)
	if err != nil {
		return "", errors.Wrap("Deploy.Package", err)
	}

	// Install on all peers using Docker exec
	for _, org := range d.network.Orgs {
		for _, peer := range org.Peers {
			if err := ctx.Err(); err != nil {
				return "", errors.Wrap("Deploy", err)
			}

			if err := d.installChaincode(ctx, org, peer, packageFile); err != nil {
				return "", errors.WrapWithContext("Deploy.Install", err, map[string]interface{}{
					"peer": peer.Name,
					"org":  org.Name,
				})
			}
		}
	}

	// Approve for all orgs using Docker exec
	for _, org := range d.network.Orgs {
		if err := ctx.Err(); err != nil {
			return "", errors.Wrap("Deploy", err)
		}

		if err := d.approveChaincode(ctx, org, req, packageFile); err != nil {
			return "", errors.WrapWithContext("Deploy.Approve", err, map[string]interface{}{
				"org": org.Name,
			})
		}
	}

	// Check context before commit
	if err := ctx.Err(); err != nil {
		return "", errors.Wrap("Deploy", err)
	}

	// Commit chaincode using Docker exec
	if err := d.commitChaincode(ctx, req); err != nil {
		return "", errors.Wrap("Deploy.Commit", err)
	}

	// Initialize chaincode if Init function exists
	if err := d.initChaincode(ctx, req); err != nil {
		// Log warning but don't fail - Init may not be required
		fmt.Printf("Warning: chaincode init returned error (may be expected): %v\n", err)
	}

	return ccID, nil
}

func (d *Deployer) packageChaincode(ctx context.Context, req *DeployRequest) (string, error) {
	packagePath := filepath.Join(d.network.BasePath, "chaincode", fmt.Sprintf("%s.tar.gz", req.Name))

	// Check context
	if err := ctx.Err(); err != nil {
		return "", errors.Wrap("packageChaincode", err)
	}

	// Convert to absolute paths
	absChaincodePath, err := filepath.Abs(req.Path)
	if err != nil {
		return "", errors.WrapWithContext("packageChaincode", err, map[string]interface{}{
			"path": req.Path,
		})
	}

	absPackagePath, err := filepath.Abs(packagePath)
	if err != nil {
		return "", errors.WrapWithContext("packageChaincode", err, map[string]interface{}{
			"path": packagePath,
		})
	}

	// Ensure output directory exists
	packageDir := filepath.Dir(absPackagePath)
	if _, err := d.exec.ExecuteCombined(ctx, "mkdir", "-p", packageDir); err != nil {
		return "", errors.WrapWithContext("packageChaincode", err, map[string]interface{}{
			"dir": packageDir,
		})
	}

	fmt.Printf("📦 Packaging chaincode from: %s\n", absChaincodePath)
	fmt.Printf("   Output: %s\n", absPackagePath)

	// Run peer lifecycle chaincode package inside Docker
	output, err := d.exec.ExecuteCombined(ctx, "docker", "run", "--rm",
		"-v", fmt.Sprintf("%s:/chaincode", absChaincodePath),
		"-v", fmt.Sprintf("%s:/output", packageDir),
		fabricToolsImage,
		"peer", "lifecycle", "chaincode", "package",
		fmt.Sprintf("/output/%s.tar.gz", req.Name),
		"--path", "/chaincode",
		"--lang", req.Language,
		"--label", fmt.Sprintf("%s_%s", req.Name, req.Version),
	)

	if err != nil {
		return "", errors.WrapWithContext("packageChaincode", errors.ErrChaincodeDeployFailed, map[string]interface{}{
			"error":  err.Error(),
			"output": string(output),
		})
	}

	fmt.Printf("✓ Chaincode packaged successfully\n")
	return absPackagePath, nil
}

func (d *Deployer) installChaincode(ctx context.Context, org *network.Organization, peer *network.Peer, packageFile string) error {
	// Check context
	if err := ctx.Err(); err != nil {
		return errors.Wrap("installChaincode", err)
	}

	fmt.Printf("📥 Installing on %s...\n", peer.Name)

	// Copy package to cli container
	containerName := "cli"
	output, err := d.exec.ExecuteCombined(ctx, "docker", "cp", packageFile, fmt.Sprintf("%s:/tmp/chaincode.tar.gz", containerName))
	if err != nil {
		return errors.WrapWithContext("installChaincode.Copy", err, map[string]interface{}{
			"container": containerName,
			"output":    string(output),
		})
	}

	// Execute install inside cli container
	env := d.getPeerEnvArgs(org, peer)
	args := []string{"exec"}
	args = append(args, env...)
	args = append(args, containerName,
		"peer", "lifecycle", "chaincode", "install", "/tmp/chaincode.tar.gz")

	output, err = d.exec.ExecuteCombined(ctx, "docker", args...)
	if err != nil {
		return errors.WrapWithContext("installChaincode.Execute", errors.ErrChaincodeDeployFailed, map[string]interface{}{
			"error":     err.Error(),
			"output":    string(output),
			"container": containerName,
		})
	}

	fmt.Printf("✓ Installed on %s\n", peer.Name)
	return nil
}

func (d *Deployer) approveChaincode(ctx context.Context, org *network.Organization, req *DeployRequest, packageFile string) error {
	// Check context
	if err := ctx.Err(); err != nil {
		return errors.Wrap("approveChaincode", err)
	}

	fmt.Printf("✅ Approving for %s...\n", org.Name)

	// Get package ID from the peer container
	packageID, err := d.getPackageID(ctx, org, req.Name, req.Version)
	if err != nil {
		return errors.Wrap("approveChaincode.GetPackageID", err)
	}

	peer := org.Peers[0]
	// containerName := peer.Name

	// Build endorsement policy
	policy := d.buildEndorsementPolicy(req.EndorsementPolicyOrgs)

	env := d.getPeerEnvArgs(org, peer)

	args := []string{"exec"}
	args = append(args, env...)
	args = append(args, "cli",
		"peer", "lifecycle", "chaincode", "approveformyorg",
		"-o", fmt.Sprintf("%s:%d", d.network.Orderers[0].Name, d.network.Orderers[0].Port),
		"--channelID", d.network.Channel.Name,
		"--name", req.Name,
		"--version", req.Version,
		"--package-id", packageID,
		"--sequence", "1",
		"--signature-policy", policy,
	)

	output, err := d.exec.ExecuteCombined(ctx, "docker", args...)
	if err != nil {
		return errors.WrapWithContext("approveChaincode.Execute", errors.ErrChaincodeDeployFailed, map[string]interface{}{
			"error":  err.Error(),
			"output": string(output),
			"org":    org.Name,
		})
	}

	fmt.Printf("✓ Approved for %s\n", org.Name)
	return nil
}

func (d *Deployer) commitChaincode(ctx context.Context, req *DeployRequest) error {
	// Check context
	if err := ctx.Err(); err != nil {
		return errors.Wrap("commitChaincode", err)
	}

	fmt.Printf("💾 Committing chaincode to channel...\n")

	// Use first org for commit
	org := d.network.Orgs[0]
	peer := org.Peers[0]
	containerName := "cli"

	// Build peer addresses
	peerAddresses := []string{}
	for _, org := range d.network.Orgs {
		for _, peer := range org.Peers {
			peerAddresses = append(peerAddresses, "--peerAddresses", fmt.Sprintf("%s:%d", peer.Name, peer.Port))
		}
	}

	policy := d.buildEndorsementPolicy(req.EndorsementPolicyOrgs)

	// Execute commit inside peer container
	env := d.getPeerEnvArgs(org, peer)
	args := []string{"exec"}
	args = append(args, env...)
	args = append(args, containerName,
		"peer", "lifecycle", "chaincode", "commit",
		"-o", fmt.Sprintf("%s:%d", d.network.Orderers[0].Name, d.network.Orderers[0].Port),
		"--channelID", d.network.Channel.Name,
		"--name", req.Name,
		"--version", req.Version,
		"--sequence", "1",
		"--signature-policy", policy,
	)
	args = append(args, peerAddresses...)

	output, err := d.exec.ExecuteCombined(ctx, "docker", args...)
	if err != nil {
		return errors.WrapWithContext("commitChaincode", errors.ErrChaincodeDeployFailed, map[string]interface{}{
			"error":  err.Error(),
			"output": string(output),
		})
	}

	fmt.Printf("✓ Chaincode committed\n")
	return nil
}

func (d *Deployer) initChaincode(ctx context.Context, req *DeployRequest) error {
	// Check context
	if err := ctx.Err(); err != nil {
		return errors.Wrap("initChaincode", err)
	}

	// Attempt to invoke Init function
	org := d.network.Orgs[0]
	peer := org.Peers[0]
	containerName := "cli"

	env := d.getPeerEnvArgs(org, peer)
	args := []string{"exec"}
	args = append(args, env...)
	args = append(args, containerName,
		"peer", "chaincode", "invoke",
		"-o", fmt.Sprintf("%s:%d", d.network.Orderers[0].Name, d.network.Orderers[0].Port),
		"-C", d.network.Channel.Name,
		"-n", req.Name,
		"--isInit",
		"-c", `{"Args":["Init"]}`,
	)

	output, err := d.exec.ExecuteCombined(ctx, "docker", args...)
	if err != nil {
		// Don't return error - Init may not be required
		fmt.Printf("Init output: %s\n", string(output))
	}

	return nil
}

func (d *Deployer) getPackageID(ctx context.Context, org *network.Organization, name, version string) (string, error) {
	// Check context
	if err := ctx.Err(); err != nil {
		return "", errors.Wrap("getPackageID", err)
	}

	peer := org.Peers[0]
	containerName := "cli"

	env := d.getPeerEnvArgs(org, peer)
	args := []string{"exec"}
	args = append(args, env...)
	args = append(args, containerName,
		"peer", "lifecycle", "chaincode", "queryinstalled")

	output, err := d.exec.ExecuteCombined(ctx, "docker", args...)
	if err != nil {
		return "", errors.WrapWithContext("getPackageID", err, map[string]interface{}{
			"error":  err.Error(),
			"output": string(output),
		})
	}

	// Parse output to find package ID
	label := fmt.Sprintf("%s_%s", name, version)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, label) {
			// Extract package ID from line like: "Package ID: label:hash, Label: label"
			parts := strings.Split(line, ",")
			if len(parts) > 0 {
				idPart := strings.TrimSpace(parts[0])
				if strings.HasPrefix(idPart, "Package ID: ") {
					return strings.TrimPrefix(idPart, "Package ID: "), nil
				}
			}
		}
	}

	return "", errors.WrapWithContext("getPackageID", fmt.Errorf("package ID not found"), map[string]interface{}{
		"label": label,
	})
}

func (d *Deployer) getPeerEnvArgs(org *network.Organization, peer *network.Peer) []string {
	return []string{
		"-e", fmt.Sprintf("CORE_PEER_LOCALMSPID=%s", org.MSPID),
		"-e", fmt.Sprintf("CORE_PEER_ADDRESS=%s:%d", peer.Name, peer.Port),
		"-e", fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=/etc/hyperledger/fabric/crypto/peerOrganizations/%s/users/Admin@%s/msp", org.Domain, org.Domain),
		"-e", "CORE_PEER_TLS_ENABLED=false",
		"-e", "FABRIC_CFG_PATH=/etc/hyperledger/fabric/config",
	}
}

func (d *Deployer) buildEndorsementPolicy(orgs []string) string {
	if len(orgs) == 0 {
		// Default: require any org
		mspids := []string{}
		for _, org := range d.network.Orgs {
			mspids = append(mspids, fmt.Sprintf("'%s.member'", org.MSPID))
		}
		return fmt.Sprintf("OR(%s)", strings.Join(mspids, ","))
	}

	// Build policy from specified orgs
	mspids := []string{}
	for _, orgName := range orgs {
		for _, org := range d.network.Orgs {
			if org.Name == orgName {
				mspids = append(mspids, fmt.Sprintf("'%s.member'", org.MSPID))
			}
		}
	}

	if len(mspids) == 0 {
		// Fallback to all orgs
		for _, org := range d.network.Orgs {
			mspids = append(mspids, fmt.Sprintf("'%s.member'", org.MSPID))
		}
	}

	return fmt.Sprintf("OR(%s)", strings.Join(mspids, ","))
}
