/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nwo

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/integration/nwo/commands"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/utils"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

type Chaincode struct {
	Name              string
	Version           string
	Path              string
	Ctor              string
	Policy            string
	Lang              string
	CollectionsConfig string // optional
	PackageFile       string
}

// DeployChaincode is a helper that will install chaincode to all peers that
// are connected to the specified channel, instantiate the chaincode on one of
// the peers, and wait for the instantiation to complete on all of the peers.
//
// NOTE: This helper should not be used to deploy the same chaincode on
// multiple channels as the install will fail on subsequent calls. Instead,
// simply use InstantiateChaincode().
func DeployChaincode(n *Network, channel string, orderer *Orderer, chaincode Chaincode, peers ...*Peer) {
	if len(peers) == 0 {
		peers = n.PeersWithChannel(channel)
	}
	if len(peers) == 0 {
		return
	}

	// create temp file for chaincode package if not provided
	if chaincode.PackageFile == "" {
		tempFile, err := ioutil.TempFile("", "chaincode-package")
		Expect(err).NotTo(HaveOccurred())
		tempFile.Close()
		defer os.Remove(tempFile.Name())
		chaincode.PackageFile = tempFile.Name()
	}

	// package using the first peer
	PackageChaincode(n, chaincode, peers[0])

	// install on all peers
	InstallChaincode(n, chaincode, peers...)

	// instantiate on the first peer
	InstantiateChaincode(n, channel, orderer, chaincode, peers[0], peers...)
}

func PackageChaincode(n *Network, chaincode Chaincode, peer *Peer) {
	sess, err := n.PeerAdminSession(peer, commands.ChaincodePackage{
		Name:       chaincode.Name,
		Version:    chaincode.Version,
		Path:       chaincode.Path,
		Lang:       chaincode.Lang,
		OutputFile: chaincode.PackageFile,
	})
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, n.EventuallyTimeout).Should(gexec.Exit(0))
}

func InstallChaincode(n *Network, chaincode Chaincode, peers ...*Peer) {
	for _, p := range peers {
		sess, err := n.PeerAdminSession(p, commands.ChaincodeInstall{
			Name:        chaincode.Name,
			Version:     chaincode.Version,
			Path:        chaincode.Path,
			Lang:        chaincode.Lang,
			PackageFile: chaincode.PackageFile,
		})
		Expect(err).NotTo(HaveOccurred())
		Eventually(sess, n.EventuallyTimeout).Should(gexec.Exit(0))

		sess, err = n.PeerAdminSession(p, commands.ChaincodeListInstalled{})
		Expect(err).NotTo(HaveOccurred())
		Eventually(sess, n.EventuallyTimeout).Should(gexec.Exit(0))
		Expect(sess).To(gbytes.Say(fmt.Sprintf("Name: %s, Version: %s,", chaincode.Name, chaincode.Version)))
	}
}

func InstantiateChaincode(n *Network, channel string, orderer *Orderer, chaincode Chaincode, peer *Peer, checkPeers ...*Peer) {
	sess, err := n.PeerAdminSession(peer, commands.ChaincodeInstantiate{
		ChannelID:         channel,
		Orderer:           n.OrdererAddress(orderer, ListenPort),
		Name:              chaincode.Name,
		Version:           chaincode.Version,
		Ctor:              chaincode.Ctor,
		Policy:            chaincode.Policy,
		Lang:              chaincode.Lang,
		CollectionsConfig: chaincode.CollectionsConfig,
	})
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, n.EventuallyTimeout).Should(gexec.Exit(0))

	EnsureInstantiated(n, channel, chaincode.Name, chaincode.Version, checkPeers...)
}

func EnsureInstantiated(n *Network, channel, name, version string, peers ...*Peer) {
	for _, p := range peers {
		Eventually(listInstantiated(n, p, channel), n.EventuallyTimeout).Should(
			gbytes.Say(fmt.Sprintf("Name: %s, Version: %s,", name, version)),
		)
	}
}

func UpgradeChaincode(n *Network, channel string, orderer *Orderer, chaincode Chaincode, peers ...*Peer) {
	if len(peers) == 0 {
		peers = n.PeersWithChannel(channel)
	}
	if len(peers) == 0 {
		return
	}

	// install on all peers
	InstallChaincode(n, chaincode, peers...)

	// upgrade from the first peer
	sess, err := n.PeerAdminSession(peers[0], commands.ChaincodeUpgrade{
		ChannelID:         channel,
		Orderer:           n.OrdererAddress(orderer, ListenPort),
		Name:              chaincode.Name,
		Version:           chaincode.Version,
		Ctor:              chaincode.Ctor,
		Policy:            chaincode.Policy,
		CollectionsConfig: chaincode.CollectionsConfig,
	})
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, n.EventuallyTimeout).Should(gexec.Exit(0))

	EnsureInstantiated(n, channel, chaincode.Name, chaincode.Version, peers...)
}

func listInstantiated(n *Network, peer *Peer, channel string) func() *gbytes.Buffer {
	return func() *gbytes.Buffer {
		sess, err := n.PeerAdminSession(peer, commands.ChaincodeListInstantiated{
			ChannelID: channel,
		})
		Expect(err).NotTo(HaveOccurred())
		Eventually(sess, n.EventuallyTimeout).Should(gexec.Exit(0))
		return sess.Buffer()
	}
}

// EnableCapabilities enables a specific capabilities flag for a running network.
// It generates the config update using the first peer, signs the configuration
// with the subsequent peers, and then submits the config update using the
// first peer.
func EnableCapabilities(network *Network, channel, capabilitiesGroup, capabilitiesVersion string, orderer *Orderer, peers ...*Peer) {
	if len(peers) == 0 {
		return
	}

	config := GetConfig(network, peers[0], orderer, channel)
	updatedConfig := proto.Clone(config).(*common.Config)

	updatedConfig.ChannelGroup.Groups[capabilitiesGroup].Values["Capabilities"] = &common.ConfigValue{
		ModPolicy: "Admins",
		Value: utils.MarshalOrPanic(
			&common.Capabilities{
				Capabilities: map[string]*common.Capability{
					capabilitiesVersion: {},
				},
			},
		),
	}

	UpdateConfig(network, orderer, channel, config, updatedConfig, false, peers[0], peers...)
}

// EnableCapabilitiesOrdererAdmin enables a specific capabilities flag for a running network,
// using an Orderer Admin Session. This is required to make changes on the system channel, for example.
func EnableCapabilitiesOrdererAdmin(network *Network, channel, capabilitiesGroup, capabilitiesVersion string, orderer *Orderer, peer *Peer, additionalSigners ...*Orderer) {
	config := GetConfig(network, peer, orderer, channel)
	updatedConfig := proto.Clone(config).(*common.Config)

	if capabilitiesGroup == "Channel" {
		updatedConfig.ChannelGroup.Values["Capabilities"] = &common.ConfigValue{
			ModPolicy: "Admins",
			Value: utils.MarshalOrPanic(
				&common.Capabilities{
					Capabilities: map[string]*common.Capability{
						capabilitiesVersion: {},
					},
				},
			),
		}
	} else {
		updatedConfig.ChannelGroup.Groups[capabilitiesGroup].Values["Capabilities"] = &common.ConfigValue{
			ModPolicy: "Admins",
			Value: utils.MarshalOrPanic(
				&common.Capabilities{
					Capabilities: map[string]*common.Capability{
						capabilitiesVersion: {},
					},
				},
			),
		}
	}

	UpdateOrdererConfig(network, orderer, channel, config, updatedConfig, peer, additionalSigners...)
}
