/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package e2e

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"syscall"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/hyperledger/fabric-chaincode-evm/integration/helpers"
	"github.com/hyperledger/fabric/integration/nwo"
	"github.com/hyperledger/fabric/integration/nwo/commands"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/tedsuo/ifrit"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const LongEventualTimeout = time.Minute

var _ = Describe("EndToEnd", func() {
	var (
		testDir        string
		client         *docker.Client
		network        *nwo.Network
		chaincode      nwo.Chaincode
		process        ifrit.Process
		zeroAddress    = "0000000000000000000000000000000000000000"
		SimpleStorage  = helpers.SimpleStorageContract()
		InvokeContract = helpers.InvokeContract()
	)

	BeforeEach(func() {
		var err error
		testDir, err = ioutil.TempDir("", "e2e")
		Expect(err).NotTo(HaveOccurred())

		client, err = docker.NewClientFromEnv()
		Expect(err).NotTo(HaveOccurred())

		chaincode = nwo.Chaincode{
			Name:    "evmcc",
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
	})

	AfterEach(func() {
		if process != nil {
			process.Signal(syscall.SIGTERM)
			Eventually(process.Wait(), LongEventualTimeout).Should(Receive())
		}
		if network != nil {
			network.Cleanup()
		}
		os.RemoveAll(testDir)
	})

	It("is able to run evm bytecode contracts", func() {
		By("getting the orderer by name")
		orderer := network.Orderer("orderer")

		By("setting up the channel")
		network.CreateAndJoinChannel(orderer, "testchannel")

		By("deploying the chaincode")
		nwo.DeployChaincode(network, "testchannel", orderer, chaincode)

		By("getting the client peer by name")
		peer := network.Peer("Org1", "peer1")

		By("installing a Simple Storage SmartContract")
		sess, err := network.PeerUserSession(peer, "User1", commands.ChaincodeInvoke{
			ChannelID: "testchannel",
			Orderer:   network.OrdererAddress(orderer, nwo.ListenPort),
			Name:      "evmcc",
			Ctor:      fmt.Sprintf(`{"Args":["%s","%s"]}`, zeroAddress, SimpleStorage.CompiledBytecode),
			PeerAddresses: []string{
				network.PeerAddress(network.Peer("Org1", "peer0"), nwo.ListenPort),
				network.PeerAddress(network.Peer("Org2", "peer1"), nwo.ListenPort),
			},
			WaitForEvent: true,
		})
		Expect(err).NotTo(HaveOccurred())
		Eventually(sess, LongEventualTimeout).Should(gexec.Exit(0))
		Expect(sess.Err).To(gbytes.Say("Chaincode invoke successful. result: status:200"))

		output := sess.Err.Contents()
		contractAddr := string(regexp.MustCompile(`Chaincode invoke successful. result: status:200 payload:"([0-9a-fA-F]{40})"`).FindSubmatch(output)[1])
		Expect(contractAddr).ToNot(BeEmpty())

		By("invoking the smart contract")
		sess, err = network.PeerUserSession(peer, "User1", commands.ChaincodeInvoke{
			ChannelID: "testchannel",
			Orderer:   network.OrdererAddress(orderer, nwo.ListenPort),
			Name:      "evmcc",
			//set(3)
			Ctor: fmt.Sprintf(`{"Args":["%s","%s0000000000000000000000000000000000000000000000000000000000000003"]}`, contractAddr, SimpleStorage.FunctionHashes["set"]),
			PeerAddresses: []string{
				network.PeerAddress(network.Peer("Org1", "peer0"), nwo.ListenPort),
				network.PeerAddress(network.Peer("Org2", "peer1"), nwo.ListenPort),
			},
			WaitForEvent: true,
		})
		Expect(err).NotTo(HaveOccurred())
		Eventually(sess, LongEventualTimeout).Should(gexec.Exit(0))
		Expect(sess.Err).To(gbytes.Say("Chaincode invoke successful. result: status:200"))

		By("verifying SimpleStorage runtime bytecode")
		sess, err = network.PeerUserSession(peer, "User1", commands.ChaincodeQuery{
			ChannelID: "testchannel",
			Name:      "evmcc",
			//get()
			Ctor: fmt.Sprintf(`{"Args":["getCode","%s"]}`, contractAddr),
		})
		Expect(err).NotTo(HaveOccurred())
		Eventually(sess, LongEventualTimeout).Should(gexec.Exit(0))
		output, _ = sess.Command.CombinedOutput()
		fmt.Println(string(output))
		Expect(sess.Out).To(gbytes.Say(SimpleStorage.RuntimeBytecode))

		By("querying the smart contract")
		sess, err = network.PeerUserSession(peer, "User1", helpers.ChaincodeQueryWithHex{
			ChannelID: "testchannel",
			Name:      "evmcc",
			//get()
			Ctor: fmt.Sprintf(`{"Args":["%s","%s"]}`, contractAddr, SimpleStorage.FunctionHashes["get"]),
		})
		Expect(err).NotTo(HaveOccurred())
		Eventually(sess, LongEventualTimeout).Should(gexec.Exit(0))
		output, _ = sess.Command.CombinedOutput()
		fmt.Println(string(output))
		Expect(sess.Out).To(gbytes.Say("0000000000000000000000000000000000000000000000000000000000000003"))

		By("deploying an InvokeContract to invoke SimpleStorage")
		sess, err = network.PeerUserSession(peer, "User1", commands.ChaincodeInvoke{
			ChannelID: "testchannel",
			Orderer:   network.OrdererAddress(orderer, nwo.ListenPort),
			Name:      "evmcc",
			Ctor:      fmt.Sprintf(`{"Args":["%s","%s"]}`, zeroAddress, InvokeContract.CompiledBytecode+"000000000000000000000000"+contractAddr),
			PeerAddresses: []string{
				network.PeerAddress(network.Peer("Org1", "peer0"), nwo.ListenPort),
				network.PeerAddress(network.Peer("Org2", "peer1"), nwo.ListenPort),
			},
			WaitForEvent: true,
		})
		Expect(err).NotTo(HaveOccurred())
		Eventually(sess, LongEventualTimeout).Should(gexec.Exit(0))
		Expect(sess.Err).To(gbytes.Say("Chaincode invoke successful. result: status:200"))

		output = sess.Err.Contents()
		invokeAddr := string(regexp.MustCompile(`Chaincode invoke successful. result: status:200 payload:"([0-9a-fA-F]{40})"`).FindSubmatch(output)[1])
		Expect(invokeAddr).ToNot(BeEmpty())

		By("invoking SimpleStorage through the InvokeContract")
		sess, err = network.PeerUserSession(peer, "User1", commands.ChaincodeInvoke{
			ChannelID: "testchannel",
			Orderer:   network.OrdererAddress(orderer, nwo.ListenPort),
			Name:      "evmcc",
			//InvokeContract.setVal(8) which will cause SimpleStorage.set(8) to be invoked.
			Ctor: fmt.Sprintf(`{"Args":["%s","%s0000000000000000000000000000000000000000000000000000000000000008"]}`, invokeAddr, InvokeContract.FunctionHashes["setVal"]),
			PeerAddresses: []string{
				network.PeerAddress(network.Peer("Org1", "peer0"), nwo.ListenPort),
				network.PeerAddress(network.Peer("Org2", "peer1"), nwo.ListenPort),
			},
			WaitForEvent: true,
		})
		Expect(err).NotTo(HaveOccurred())
		Eventually(sess, LongEventualTimeout).Should(gexec.Exit(0))
		Expect(sess.Err).To(gbytes.Say("Chaincode invoke successful. result: status:200"))

		By("querying the SimpleStorage smart contract")
		sess, err = network.PeerUserSession(peer, "User1", helpers.ChaincodeQueryWithHex{
			ChannelID: "testchannel",
			Name:      "evmcc",
			//get()
			Ctor: fmt.Sprintf(`{"Args":["%s","%s"]}`, contractAddr, SimpleStorage.FunctionHashes["get"]),
		})
		Expect(err).NotTo(HaveOccurred())
		Eventually(sess, LongEventualTimeout).Should(gexec.Exit(0))
		output, _ = sess.Command.CombinedOutput()
		fmt.Println(string(output))
		Expect(sess.Out).To(gbytes.Say("0000000000000000000000000000000000000000000000000000000000000008"))
	})
})
