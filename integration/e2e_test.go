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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/tedsuo/ifrit"

	"github.com/hyperledger/fabric/integration/nwo"
	"github.com/hyperledger/fabric/integration/nwo/commands"
)

var _ = Describe("EndToEnd", func() {
	var (
		testDir     string
		client      *docker.Client
		network     *nwo.Network
		chaincode   nwo.Chaincode
		process     ifrit.Process
		zeroAddress = "0000000000000000000000000000000000000000"
		/* SimpleStorage Contract
		pragma solidity ^0.4.0;

		contract SimpleStorage {
		    uint storedData;

		    function set(uint x) public {
		        storedData = x;
		    }

		    function get() public constant returns (uint) {
		        return storedData;
		    }
		}
		*/

		//Compiled SimpleStorage contract
		compileBytecode = "6060604052341561000f57600080fd5b60d38061001d6000396000f3006060604052600436106049576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806360fe47b114604e5780636d4ce63c14606e575b600080fd5b3415605857600080fd5b606c60048080359060200190919050506094565b005b3415607857600080fd5b607e609e565b6040518082815260200191505060405180910390f35b8060008190555050565b600080549050905600a165627a7a72305820122f55f799d70b5f6dbfd4312efb65cdbfaacddedf7c36249b8b1e915a8dd85b0029"
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
			Eventually(process.Wait(), time.Minute).Should(Receive())
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
			Ctor:      fmt.Sprintf(`{"Args":["%s","%s"]}`, zeroAddress, compileBytecode),
			PeerAddresses: []string{
				network.PeerAddress(network.Peer("Org1", "peer0"), nwo.ListenPort),
				network.PeerAddress(network.Peer("Org2", "peer1"), nwo.ListenPort),
			},
			WaitForEvent: true,
		})
		Expect(err).NotTo(HaveOccurred())
		Eventually(sess, time.Minute).Should(gexec.Exit(0))
		Expect(sess.Err).To(gbytes.Say("Chaincode invoke successful. result: status:200"))

		output := sess.Err.Contents()
		contractAddr := string(regexp.MustCompile(`Chaincode invoke successful. result: status:200 payload:"([0-9a-fA-F]{40})"`).FindSubmatch(output)[1])
		Expect(contractAddr).ToNot(BeEmpty())

		By("invoking the smart contract")
		sess, err = network.PeerUserSession(peer, "User1", commands.ChaincodeInvoke{
			ChannelID: "testchannel",
			Orderer:   network.OrdererAddress(orderer, nwo.ListenPort),
			Name:      "evmcc",
			//Set function hash: 60fe47b1
			//set(3)
			Ctor: fmt.Sprintf(`{"Args":["%s","60fe47b10000000000000000000000000000000000000000000000000000000000000003"]}`, contractAddr),
			PeerAddresses: []string{
				network.PeerAddress(network.Peer("Org1", "peer0"), nwo.ListenPort),
				network.PeerAddress(network.Peer("Org2", "peer1"), nwo.ListenPort),
			},
			WaitForEvent: true,
		})
		Expect(err).NotTo(HaveOccurred())
		Eventually(sess, time.Minute).Should(gexec.Exit(0))
		Expect(sess.Err).To(gbytes.Say("Chaincode invoke successful. result: status:200"))

		By("querying the smart contract")
		sess, err = network.PeerUserSession(peer, "User1", ChaincodeQueryWithHex{
			ChannelID: "testchannel",
			Name:      "evmcc",
			//get function hash: 6d4ce63c
			//get()
			Ctor: fmt.Sprintf(`{"Args":["%s","6d4ce63c"]}`, contractAddr),
		})
		Expect(err).NotTo(HaveOccurred())
		Eventually(sess, time.Minute).Should(gexec.Exit(0))
		output, _ = sess.Command.CombinedOutput()
		fmt.Println(string(output))
		Expect(sess.Out).To(gbytes.Say("0000000000000000000000000000000000000000000000000000000000000003"))
	})
})
