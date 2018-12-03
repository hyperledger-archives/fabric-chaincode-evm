/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package web3_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"syscall"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/hyperledger/fabric-chaincode-evm/integration/helpers"
	"github.com/hyperledger/fabric/integration/nwo"
	"github.com/onsi/gomega/gbytes"
	"github.com/tedsuo/ifrit"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const LongEventualTimeout = 2 * time.Minute

var _ = Describe("Web3 Integration", func() {
	var (
		testDir         string
		client          *docker.Client
		network         *nwo.Network
		chaincode       nwo.Chaincode
		process         ifrit.Process
		channelName     string
		proxyConfigPath string
		ccid            string

		user1Proxy ifrit.Process
		user2Proxy ifrit.Process
	)

	BeforeEach(func() {
		var err error
		testDir, err = ioutil.TempDir("", "web3-e2e")
		Expect(err).NotTo(HaveOccurred())

		client, err = docker.NewClientFromEnv()
		Expect(err).NotTo(HaveOccurred())

		ccid = "evmcc"
		chaincode = nwo.Chaincode{
			Name:    ccid,
			Version: "0.0",
			Path:    "github.com/hyperledger/fabric-chaincode-evm/evmcc",
			Ctor:    `{"Args":[]}`,
			Policy:  `AND ('Org1MSP.member','Org2MSP.member')`,
		}
		network = nwo.New(nwo.BasicSolo(), testDir, client, 30000, components)
		network.GenerateConfigTree()
		network.Bootstrap()

		networkRunner := network.NetworkGroupRunner()
		process = ifrit.Invoke(networkRunner)
		Eventually(process.Ready()).Should(BeClosed())
		channelName = "testchannel"

		proxyConfigPath, err = helpers.CreateProxyConfig(testDir, channelName, network.CryptoPath(),
			network.PeerPort(network.Peer("Org1", "peer0"), nwo.ListenPort),
			network.PeerPort(network.Peer("Org1", "peer1"), nwo.ListenPort),
			network.PeerPort(network.Peer("Org2", "peer0"), nwo.ListenPort),
			network.PeerPort(network.Peer("Org2", "peer1"), nwo.ListenPort),
			network.OrdererPort(network.Orderer("orderer"), nwo.ListenPort),
		)
		Expect(err).ToNot(HaveOccurred())

		//Set up the network
		By("getting the orderer by name")
		orderer := network.Orderer("orderer")

		By("setting up the channel")
		network.CreateAndJoinChannel(orderer, "testchannel")
		network.UpdateChannelAnchors(orderer, "testchannel")

		By("deploying the chaincode")
		nwo.DeployChaincode(network, "testchannel", orderer, chaincode)

	})

	AfterEach(func() {
		if process != nil {
			process.Signal(syscall.SIGTERM)
			Eventually(process.Wait(), LongEventualTimeout).Should(Receive())
		}
		if network != nil {
			network.Cleanup()
		}
		if user1Proxy != nil {
			user1Proxy.Signal(syscall.SIGTERM)
			Eventually(user1Proxy.Wait(), LongEventualTimeout).Should(Receive())
		}
		if user2Proxy != nil {
			user2Proxy.Signal(syscall.SIGTERM)
			Eventually(user2Proxy.Wait(), LongEventualTimeout).Should(Receive())
		}
		os.RemoveAll(testDir)
	})

	It("web3 can deploy and interact with smart contracts", func() {
		By("starting up a fabproxy for user 1")
		user1ProxyPort := network.ReservePort()
		user1ProxyRunner := helpers.FabProxyRunner(components.Paths["fabproxy"], proxyConfigPath, "Org1", "User1", channelName, ccid, user1ProxyPort)
		user1Proxy = ifrit.Invoke(user1ProxyRunner)
		Eventually(user1Proxy.Ready(), LongEventualTimeout).Should(BeClosed())

		By("starting up a fabproxy for user 2")
		user2ProxyPort := network.ReservePort()
		user2ProxyRunner := helpers.FabProxyRunner(components.Paths["fabproxy"], proxyConfigPath, "Org2", "User2", channelName, ccid, user2ProxyPort)
		user2Proxy = ifrit.Invoke(user2ProxyRunner)
		Eventually(user2Proxy.Ready(), LongEventualTimeout).Should(BeClosed())

		By("running the web3 tests")
		web3TestRunner := helpers.Web3TestRunner(
			fmt.Sprintf("http://127.0.0.1:%d", user1ProxyPort),
			fmt.Sprintf("http://127.0.0.1:%d", user2ProxyPort),
		)

		web3Process := ifrit.Invoke(web3TestRunner)
		Eventually(web3Process.Ready()).Should(BeClosed())

		Eventually(web3Process.Wait(), LongEventualTimeout).Should(Receive())
		Expect(web3TestRunner.ExitCode()).Should(Equal(0))

		Expect(web3TestRunner.Buffer()).To(gbytes.Say("Successfully able to deploy Voting Smart Contract and interact with it"))
	})
})
