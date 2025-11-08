// pkg/chaincode/deployer.go
package chaincode

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/temmyjay001/fabricx-core/pkg/docker"
	"github.com/temmyjay001/fabricx-core/pkg/network"

	"github.com/google/uuid"
)

const (
	fabricToolsImage = "hyperledger/fabric-tools:2.5"
)

type Deployer struct {
	network   *network.Network
	dockerMgr *docker.Manager
}

type DeployRequest struct {
	Name                  string
	Path                  string
	Version               string
	Language              string
	EndorsementPolicyOrgs []string
}

func NewDeployer(net *network.Network, dockerMgr *docker.Manager) *Deployer {
	return &Deployer{
		network:   net,
		dockerMgr: dockerMgr,
	}
}

func (d *Deployer) Deploy(ctx context.Context, req *DeployRequest) (string, error) {
	// Set defaults
	if req.Version == "" {
		req.Version = "1.0"
	}
	if req.Language == "" {
		req.Language = "golang"
	}

	ccID := fmt.Sprintf("%s-%s", req.Name, uuid.New().String()[:8])

	// Package chaincode using Docker
	packageFile, err := d.packageChaincode(req)
	if err != nil {
		return "", fmt.Errorf("failed to package chaincode: %w", err)
	}

	// Install on all peers using Docker exec
	for _, org := range d.network.Orgs {
		for _, peer := range org.Peers {
			if err := d.installChaincode(org, peer, packageFile); err != nil {
				return "", fmt.Errorf("failed to install on %s: %w", peer.Name, err)
			}
		}
	}

	// Approve for all orgs using Docker exec
	for _, org := range d.network.Orgs {
		if err := d.approveChaincode(org, req, packageFile); err != nil {
			return "", fmt.Errorf("failed to approve for %s: %w", org.Name, err)
		}
	}

	// Commit chaincode using Docker exec
	if err := d.commitChaincode(req); err != nil {
		return "", fmt.Errorf("failed to commit chaincode: %w", err)
	}

	// Initialize chaincode if Init function exists
	if err := d.initChaincode(req); err != nil {
		// Log warning but don't fail - Init may not be required
		fmt.Printf("Warning: chaincode init returned error (may be expected): %v\n", err)
	}

	return ccID, nil
}

func (d *Deployer) packageChaincode(req *DeployRequest) (string, error) {
	packagePath := filepath.Join(d.network.BasePath, "chaincode", fmt.Sprintf("%s.tar.gz", req.Name))

	// Run peer lifecycle chaincode package inside Docker
	cmd := exec.Command("docker", "run", "--rm",
		"-v", fmt.Sprintf("%s:/chaincode", filepath.Dir(req.Path)),
		"-v", fmt.Sprintf("%s:/output", filepath.Dir(packagePath)),
		fabricToolsImage,
		"peer", "lifecycle", "chaincode", "package",
		fmt.Sprintf("/output/%s.tar.gz", req.Name),
		"--path", fmt.Sprintf("/chaincode/%s", filepath.Base(req.Path)),
		"--lang", req.Language,
		"--label", fmt.Sprintf("%s_%s", req.Name, req.Version),
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("package command failed: %w\nOutput: %s", err, string(output))
	}

	return packagePath, nil
}

func (d *Deployer) installChaincode(org *network.Organization, peer *network.Peer, packageFile string) error {
	// Copy package to peer container
	containerName := peer.Name
	cmd := exec.Command("docker", "cp", packageFile, fmt.Sprintf("%s:/tmp/chaincode.tar.gz", containerName))
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to copy package: %w\nOutput: %s", err, string(output))
	}

	// Execute install inside peer container
	env := d.getPeerEnvArgs(org, peer)
	args := []string{"exec"}
	args = append(args, env...)
	args = append(args, containerName,
		"peer", "lifecycle", "chaincode", "install", "/tmp/chaincode.tar.gz")

	cmd = exec.Command("docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("install failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

func (d *Deployer) approveChaincode(org *network.Organization, req *DeployRequest, packageFile string) error {
	// Get package ID from the peer container
	packageID, err := d.getPackageID(org, req.Name, req.Version)
	if err != nil {
		return fmt.Errorf("failed to get package ID: %w", err)
	}

	peer := org.Peers[0]
	containerName := peer.Name

	// Build endorsement policy
	policy := d.buildEndorsementPolicy(req.EndorsementPolicyOrgs)

	// Execute approve inside peer container
	env := d.getPeerEnvArgs(org, peer)
	args := []string{"exec"}
	args = append(args, env...)
	args = append(args, containerName,
		"peer", "lifecycle", "chaincode", "approveformyorg",
		"-o", fmt.Sprintf("%s:%d", d.network.Orderers[0].Name, d.network.Orderers[0].Port),
		"--channelID", d.network.Channel.Name,
		"--name", req.Name,
		"--version", req.Version,
		"--package-id", packageID,
		"--sequence", "1",
		"--signature-policy", policy,
	)

	cmd := exec.Command("docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("approve failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

func (d *Deployer) commitChaincode(req *DeployRequest) error {
	// Use first org for commit
	org := d.network.Orgs[0]
	peer := org.Peers[0]
	containerName := peer.Name

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

	cmd := exec.Command("docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("commit failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

func (d *Deployer) initChaincode(req *DeployRequest) error {
	// Attempt to invoke Init function
	org := d.network.Orgs[0]
	peer := org.Peers[0]
	containerName := peer.Name

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

	cmd := exec.Command("docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Don't return error - Init may not be required
		fmt.Printf("Init output: %s\n", string(output))
	}

	return nil
}

func (d *Deployer) getPackageID(org *network.Organization, name, version string) (string, error) {
	peer := org.Peers[0]
	containerName := peer.Name

	env := d.getPeerEnvArgs(org, peer)
	args := []string{"exec"}
	args = append(args, env...)
	args = append(args, containerName,
		"peer", "lifecycle", "chaincode", "queryinstalled")

	cmd := exec.Command("docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("query installed failed: %w\nOutput: %s", err, string(output))
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

	return "", fmt.Errorf("package ID not found for %s", label)
}

func (d *Deployer) getPeerEnvArgs(org *network.Organization, peer *network.Peer) []string {
	_ = fmt.Sprintf("/etc/hyperledger/fabric/msp")

	return []string{
		"-e", fmt.Sprintf("CORE_PEER_LOCALMSPID=%s", org.MSPID),
		"-e", fmt.Sprintf("CORE_PEER_ADDRESS=%s:%d", peer.Name, peer.Port),
		"-e", fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=/etc/hyperledger/fabric/users/Admin@%s/msp", org.Domain),
		"-e", "CORE_PEER_TLS_ENABLED=false",
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
