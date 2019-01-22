/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fab3

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"syscall"
	"testing"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/hyperledger/fabric-chaincode-evm/integration/helpers"
	"github.com/hyperledger/fabric/integration/nwo"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
)

func TestFab3(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Fab3")
}

const LongEventualTimeout = time.Minute
const LongPollingInterval = 500 * time.Millisecond
const DefaultEventuallyTimeout = time.Second
const DefaultEventuallyPollingInterval = 10 * time.Millisecond

var (
	components   *nwo.Components
	testDir      string
	dockerClient *docker.Client
	network      *nwo.Network
	process      ifrit.Process
	ccid         = "evmcc"
	channelName  = "testchannel"
)

var _ = SynchronizedBeforeSuite(func() []byte {
	components = helpers.Build()

	var err error
	testDir, err = ioutil.TempDir("", "fab3-e2e")
	Expect(err).NotTo(HaveOccurred())

	dockerClient, err = docker.NewClientFromEnv()
	Expect(err).NotTo(HaveOccurred())

	network = nwo.New(nwo.BasicSolo(), testDir, dockerClient, 30000, components)
	network.GenerateConfigTree()
	network.Bootstrap()

	networkRunner := network.NetworkGroupRunner()
	process = ifrit.Invoke(networkRunner)
	Eventually(process.Ready(), DefaultEventuallyTimeout, DefaultEventuallyPollingInterval).Should(BeClosed())

	components.Paths["Fab3Config"], err = helpers.CreateProxyConfig(testDir, channelName, network.CryptoPath(),
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
	network.CreateAndJoinChannel(orderer, channelName)
	network.UpdateChannelAnchors(orderer, channelName)

	By("deploying the chaincode")
	chaincode := nwo.Chaincode{
		Name:    ccid,
		Version: "0.0",
		Path:    "github.com/hyperledger/fabric-chaincode-evm/evmcc",
		Ctor:    `{"Args":[]}`,
		Policy:  `AND ('Org1MSP.member','Org2MSP.member')`,
	}
	nwo.DeployChaincode(network, channelName, orderer, chaincode)

	payload, err := json.Marshal(components)
	Expect(err).ToNot(HaveOccurred())

	return payload
}, func(payload []byte) {
	err := json.Unmarshal(payload, &components)
	Expect(err).NotTo(HaveOccurred())
})

var _ = SynchronizedAfterSuite(func() {
}, func() {
	if process != nil {
		process.Signal(syscall.SIGTERM)
		Eventually(process.Wait(), LongEventualTimeout, LongPollingInterval).Should(Receive())
	}
	if network != nil {
		network.Cleanup()
	}
	components.Cleanup()
	os.RemoveAll(testDir)
})
