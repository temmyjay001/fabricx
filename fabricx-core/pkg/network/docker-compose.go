// pkg/network/docker-compose.go
package network

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/temmyjay001/core/pkg/utils"
)

func generateDockerCompose(net *Network) error {
	composePath := filepath.Join(net.ConfigPath, "docker-compose.yaml")

	compose := map[string]interface{}{
		"version": "3.7",
		"networks": map[string]interface{}{
			"fabricx": map[string]interface{}{
				"name": fmt.Sprintf("fabricx_%s", net.ID),
			},
		},
		"volumes":  generateVolumes(net),
		"services": generateServices(net),
	}

	return utils.WriteYAML(composePath, compose)
}

func generateVolumes(net *Network) map[string]interface{} {
	volumes := make(map[string]interface{})

	// Orderer volumes - use just the hostname, not the full FQDN
	for _, orderer := range net.Orderers {
		// Extract hostname from full name (e.g., "orderer.example.com" -> "orderer")
		volumeName := orderer.Name
		volumes[volumeName] = nil
	}

	// Peer volumes
	for i, org := range net.Orgs {
		for j := range org.Peers {
			peerVolumeName := fmt.Sprintf("peer%d.%s", j, org.Domain)
			volumes[peerVolumeName] = nil

			if org.Peers[j].CouchDB {
				couchVolumeName := fmt.Sprintf("couchdb%d.%s", j, org.Domain)
				volumes[couchVolumeName] = nil
			}
		}
		_ = i
	}

	return volumes
}

func generateServices(net *Network) map[string]interface{} {
	services := make(map[string]interface{})

	// Add orderer services
	for _, orderer := range net.Orderers {
		services[orderer.Name] = generateOrdererService(net, orderer)
	}

	// Add CA, peer, and CouchDB services for each org
	globalPeerIndex := 0
	for _, org := range net.Orgs {
		// CA service
		caName := fmt.Sprintf("ca.%s", org.Domain)
		services[caName] = generateCAService(net, org)

		// Peer and CouchDB services
		for i, peer := range org.Peers {
			if peer.CouchDB {
				couchName := fmt.Sprintf("couchdb%d.%s", i, org.Domain)
				services[couchName] = generateCouchDBService(net, org, peer, i)
			}
			services[peer.Name] = generatePeerService(net, org, peer, i, globalPeerIndex)
			globalPeerIndex++
		}
	}

	// Add CLI tool service for executing commands
	services["cli"] = generateCLIService(net)

	return services
}

func generateOrdererService(net *Network, orderer *Orderer) map[string]interface{} {
	return map[string]interface{}{
		"container_name": orderer.Name,
		"image":          "hyperledger/fabric-orderer:2.5",
		"environment": []string{
			"FABRIC_LOGGING_SPEC=INFO",
			"ORDERER_GENERAL_LISTENADDRESS=0.0.0.0",
			fmt.Sprintf("ORDERER_GENERAL_LISTENPORT=%d", orderer.Port),
			"ORDERER_GENERAL_LOCALMSPID=OrdererMSP",
			"ORDERER_GENERAL_LOCALMSPDIR=/var/hyperledger/orderer/msp",
			"ORDERER_GENERAL_TLS_ENABLED=false",
			"ORDERER_GENERAL_GENESISMETHOD=file",
			"ORDERER_GENERAL_GENESISFILE=/var/hyperledger/orderer/orderer.genesis.block",
			"ORDERER_GENERAL_CLUSTER_CLIENTCERTIFICATE=/var/hyperledger/orderer/tls/server.crt",
			"ORDERER_GENERAL_CLUSTER_CLIENTPRIVATEKEY=/var/hyperledger/orderer/tls/server.key",
			"ORDERER_GENERAL_CLUSTER_ROOTCAS=[/var/hyperledger/orderer/tls/ca.crt]",
			"ORDERER_OPERATIONS_LISTENADDRESS=0.0.0.0:8443",
		},
		"working_dir": "/opt/gopath/src/github.com/hyperledger/fabric",
		"command":     "orderer",
		"volumes": []string{
			fmt.Sprintf("%s/genesis.block:/var/hyperledger/orderer/orderer.genesis.block", net.ConfigPath),
			fmt.Sprintf("%s/ordererOrganizations/example.com/orderers/%s/msp:/var/hyperledger/orderer/msp", net.CryptoPath, orderer.Name),
			fmt.Sprintf("%s/ordererOrganizations/example.com/orderers/%s/tls:/var/hyperledger/orderer/tls", net.CryptoPath, orderer.Name),
			fmt.Sprintf("%s:/var/hyperledger/production/orderer", orderer.Name),
		},
		"ports": []string{
			fmt.Sprintf("%d:%d", orderer.Port, orderer.Port),
			"8443:8443",
		},
		"networks": []string{"fabricx"},
	}
}

func generateCAService(net *Network, org *Organization) map[string]interface{} {
	caName := fmt.Sprintf("ca.%s", org.Domain)
	return map[string]interface{}{
		"container_name": caName,
		"image":          "hyperledger/fabric-ca:1.5",
		"environment": []string{
			"FABRIC_CA_HOME=/etc/hyperledger/fabric-ca-server",
			fmt.Sprintf("FABRIC_CA_SERVER_CA_NAME=%s", caName),
			"FABRIC_CA_SERVER_TLS_ENABLED=false",
			fmt.Sprintf("FABRIC_CA_SERVER_PORT=%d", org.CAPort),
		},
		"ports": []string{
			fmt.Sprintf("%d:%d", org.CAPort, org.CAPort),
		},
		"command": "sh -c 'fabric-ca-server start -b admin:adminpw -d'",
		"volumes": []string{
			fmt.Sprintf("%s/peerOrganizations/%s/ca/:/etc/hyperledger/fabric-ca-server-config", net.CryptoPath, org.Domain),
		},
		"networks": []string{"fabricx"},
	}
}

func generatePeerService(net *Network, org *Organization, peer *Peer, index int, globalIndex int) map[string]interface{} {
	service := map[string]interface{}{
		"container_name": peer.Name,
		"image":          "hyperledger/fabric-peer:2.5",
		"environment": []string{
			"CORE_VM_ENDPOINT=unix:///host/var/run/docker.sock",
			"CORE_VM_DOCKER_HOSTCONFIG_NETWORKMODE=" + fmt.Sprintf("fabricx_%s", net.ID),
			"FABRIC_LOGGING_SPEC=INFO",
			fmt.Sprintf("CORE_PEER_ID=%s", peer.Name),
			fmt.Sprintf("CORE_PEER_ADDRESS=%s:%d", peer.Name, peer.Port),
			fmt.Sprintf("CORE_PEER_LISTENADDRESS=0.0.0.0:%d", peer.Port),
			fmt.Sprintf("CORE_PEER_CHAINCODEADDRESS=%s:%d", peer.Name, peer.Port+1),
			fmt.Sprintf("CORE_PEER_CHAINCODELISTENADDRESS=0.0.0.0:%d", peer.Port+1),
			fmt.Sprintf("CORE_PEER_GOSSIP_EXTERNALENDPOINT=%s:%d", peer.Name, peer.Port),
			fmt.Sprintf("CORE_PEER_GOSSIP_BOOTSTRAP=%s:%d", peer.Name, peer.Port),
			fmt.Sprintf("CORE_PEER_LOCALMSPID=%s", org.MSPID),
			// fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=/etc/hyperledger/fabric/users/Admin@%s/msp", org.Domain),
			"CORE_PEER_MSPCONFIGPATH=/etc/hyperledger/fabric/msp",
			"CORE_PEER_TLS_ENABLED=false",
			"CORE_OPERATIONS_LISTENADDRESS=0.0.0.0:9443",
		},
		"working_dir": "/opt/gopath/src/github.com/hyperledger/fabric/peer",
		"command":     "peer node start",
		"volumes": []string{
			"/var/run/docker.sock:/host/var/run/docker.sock",
			fmt.Sprintf("%s/peerOrganizations/%s/peers/%s/msp:/etc/hyperledger/fabric/msp", net.CryptoPath, org.Domain, peer.Name),
			fmt.Sprintf("%s/peerOrganizations/%s/peers/%s/tls:/etc/hyperledger/fabric/tls", net.CryptoPath, org.Domain, peer.Name),
			// Mount admin MSP for CLI operations inside container
			fmt.Sprintf("%s/peerOrganizations/%s/users:/etc/hyperledger/fabric/users", net.CryptoPath, org.Domain),
			fmt.Sprintf("peer%d.%s:/var/hyperledger/production", index, org.Domain),
			fmt.Sprintf("%s:/etc/hyperledger/fabric/config", net.ConfigPath),
		},
		"ports": []string{
			fmt.Sprintf("%d:%d", peer.Port, peer.Port),
			fmt.Sprintf("%d:9443", 9443+(globalIndex*1000)),
		},
		"networks": []string{"fabricx"},
	}

	// Add CouchDB dependency if enabled
	if peer.CouchDB {
		couchName := fmt.Sprintf("couchdb%d.%s", index, org.Domain)
		service["depends_on"] = []string{couchName}

		// Add CouchDB environment variables
		envs := service["environment"].([]string)
		envs = append(envs,
			"CORE_LEDGER_STATE_STATEDATABASE=CouchDB",
			fmt.Sprintf("CORE_LEDGER_STATE_COUCHDBCONFIG_COUCHDBADDRESS=%s:5984", couchName),
			"CORE_LEDGER_STATE_COUCHDBCONFIG_USERNAME=admin",
			"CORE_LEDGER_STATE_COUCHDBCONFIG_PASSWORD=adminpw",
		)
		service["environment"] = envs
	}

	return service
}

func generateCouchDBService(net *Network, org *Organization, peer *Peer, index int) map[string]interface{} {
	couchName := fmt.Sprintf("couchdb%d.%s", index, org.Domain)
	return map[string]interface{}{
		"container_name": couchName,
		"image":          "couchdb:3.3",
		"environment": []string{
			"COUCHDB_USER=admin",
			"COUCHDB_PASSWORD=adminpw",
		},
		"ports": []string{
			fmt.Sprintf("%d:5984", peer.DBPort),
		},
		"volumes": []string{
			fmt.Sprintf("couchdb%d.%s:/opt/couchdb/data", index, org.Domain),
		},
		"networks": []string{"fabricx"},
	}
}

// generateCoreYAML creates a minimal core.yaml for the CLI container
func generateCoreYAML(net *Network) error {
	coreYAMLPath := filepath.Join(net.ConfigPath, "core.yaml")

	coreConfig := map[string]interface{}{
		"peer": map[string]interface{}{
			"id":                "cli",
			"networkId":         "fabricx",
			"address":           "0.0.0.0:7051",
			"addressAutoDetect": false,
			"gomaxprocs":        -1,
			"keepalive": map[string]interface{}{
				"minInterval": "60s",
				"client": map[string]interface{}{
					"interval": "60s",
					"timeout":  "20s",
				},
				"deliveryClient": map[string]interface{}{
					"interval": "60s",
					"timeout":  "20s",
				},
			},
			"gossip": map[string]interface{}{
				"bootstrap":                  "127.0.0.1:7051",
				"useLeaderElection":          true,
				"orgLeader":                  false,
				"endpoint":                   "",
				"maxBlockCountToStore":       100,
				"maxPropagationBurstLatency": "10ms",
				"maxPropagationBurstSize":    10,
				"propagateIterations":        1,
				"propagatePeerNum":           3,
				"pullInterval":               "4s",
				"pullPeerNum":                3,
				"requestStateInfoInterval":   "4s",
				"publishStateInfoInterval":   "4s",
				"stateInfoRetentionInterval": "",
				"publishCertPeriod":          "10s",
				"skipBlockVerification":      false,
				"dialTimeout":                "3s",
				"connTimeout":                "2s",
				"recvBuffSize":               20,
				"sendBuffSize":               200,
				"digestWaitTime":             "1s",
				"requestWaitTime":            "1500ms",
				"responseWaitTime":           "2s",
				"aliveTimeInterval":          "5s",
				"aliveExpirationTimeout":     "25s",
				"reconnectInterval":          "25s",
				"election": map[string]interface{}{
					"startupGracePeriod":       "15s",
					"membershipSampleInterval": "1s",
					"leaderAliveThreshold":     "10s",
					"leaderElectionDuration":   "5s",
				},
			},
			"tls": map[string]interface{}{
				"enabled": false,
			},
			"bccsp": map[string]interface{}{
				"default": "SW",
				"sw": map[string]interface{}{
					"hash":     "SHA2",
					"security": 256,
				},
			},
			"fileSystemPath": "/var/hyperledger/production",
		},
		"vm": map[string]interface{}{
			"endpoint": "unix:///host/var/run/docker.sock",
		},
		"chaincode": map[string]interface{}{
			"builder": "$(DOCKER_NS)/fabric-ccenv:$(TWO_DIGIT_VERSION)",
			"pull":    false,
			"golang": map[string]interface{}{
				"runtime":     "$(DOCKER_NS)/fabric-baseos:$(TWO_DIGIT_VERSION)",
				"dynamicLink": false,
			},
			"java": map[string]interface{}{
				"runtime": "$(DOCKER_NS)/fabric-javaenv:$(TWO_DIGIT_VERSION)",
			},
			"node": map[string]interface{}{
				"runtime": "$(DOCKER_NS)/fabric-nodeenv:$(TWO_DIGIT_VERSION)",
			},
			"startuptimeout": "300s",
			"executetimeout": "30s",
			"mode":           "net",
			"keepalive":      0,
		},
		"ledger": map[string]interface{}{
			"state": map[string]interface{}{
				"stateDatabase": "goleveldb",
				"couchDBConfig": map[string]interface{}{
					"couchDBAddress":          "127.0.0.1:5984",
					"username":                "",
					"password":                "",
					"maxRetries":              3,
					"maxRetriesOnStartup":     10,
					"requestTimeout":          "35s",
					"queryLimit":              10000,
					"maxBatchUpdateSize":      1000,
					"warmIndexesAfterNBlocks": 1,
				},
			},
		},
	}

	return utils.WriteYAML(coreYAMLPath, coreConfig)
}

// generateCLIService creates a CLI container with fabric-tools for executing commands
func generateCLIService(net *Network) map[string]interface{} {
	// Use first org for CLI
	org := net.Orgs[0]

	log.Println(net.Config)
	log.Println(net.CryptoPath)

	// Build volume mounts for all orgs
	volumes := []string{
		"/var/run/docker.sock:/host/var/run/docker.sock",
		fmt.Sprintf("%s:/etc/hyperledger/fabric/config", net.ConfigPath),
		fmt.Sprintf("%s:/etc/hyperledger/fabric/crypto", net.CryptoPath),
	}

	log.Println(volumes)

	// Mount all org MSPs so CLI can switch between orgs
	for _, o := range net.Orgs {
		volumes = append(volumes,
			fmt.Sprintf("%s/peerOrganizations/%s/users:/etc/hyperledger/fabric/crypto/peerOrganizations/%s/users",
				net.CryptoPath, o.Domain, o.Domain))
	}

	return map[string]interface{}{
		"container_name": "cli",
		"image":          "hyperledger/fabric-tools:2.5",
		"tty":            true,
		"stdin_open":     true,
		"environment": []string{
			"GOPATH=/opt/gopath",
			"FABRIC_LOGGING_SPEC=INFO",
			"FABRIC_CFG_PATH=/etc/hyperledger/fabric/config",
			fmt.Sprintf("CORE_PEER_LOCALMSPID=%s", org.MSPID),
			fmt.Sprintf("CORE_PEER_ADDRESS=%s:%d", org.Peers[0].Name, org.Peers[0].Port),
			fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=/etc/hyperledger/fabric/crypto/peerOrganizations/%s/users/Admin@%s/msp", org.Domain, org.Domain),
			"CORE_PEER_TLS_ENABLED=false",
		},
		"working_dir": "/opt/gopath/src/github.com/hyperledger/fabric/peer",
		"command":     "/bin/bash",
		"volumes":     volumes,
		"networks":    []string{"fabricx"},
		"depends_on": func() []string {
			deps := []string{}
			for _, orderer := range net.Orderers {
				deps = append(deps, orderer.Name)
			}
			for _, org := range net.Orgs {
				for _, peer := range org.Peers {
					deps = append(deps, peer.Name)
				}
			}
			return deps
		}(),
	}
}
