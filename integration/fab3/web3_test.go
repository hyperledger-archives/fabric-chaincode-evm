/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fab3

import (
	"fmt"
	"syscall"
	"time"

	"github.com/hyperledger/fabric-chaincode-evm/integration/helpers"
	"github.com/onsi/ginkgo/config"
	"github.com/onsi/gomega/gbytes"
	"github.com/tedsuo/ifrit"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const Web3EventuallyTimeout = 5 * time.Minute
const Web3EventuallyPollingInterval = 1 * time.Second

var _ = Describe("Web3 Integration", func() {
	var (
		user1Proxy ifrit.Process
		user2Proxy ifrit.Process
	)

	AfterEach(func() {
		if user1Proxy != nil {
			user1Proxy.Signal(syscall.SIGTERM)
			Eventually(user1Proxy.Wait(), LongEventualTimeout, LongPollingInterval).Should(Receive())
		}
		if user2Proxy != nil {
			user2Proxy.Signal(syscall.SIGTERM)
			Eventually(user2Proxy.Wait(), LongEventualTimeout, LongPollingInterval).Should(Receive())
		}
	})

	It("web3 can deploy and interact with smart contracts", func() {
		By("starting up a fab3 for user 1")
		user1ProxyPort := uint16(2000 + config.GinkgoConfig.ParallelNode)
		user1ProxyRunner := helpers.Fab3Runner(components.Paths["fab3"], components.Paths["Fab3Config"], "Org1", "User1", channelName, ccid, user1ProxyPort)
		user1Proxy = ifrit.Invoke(user1ProxyRunner)
		Eventually(user1Proxy.Ready(), LongEventualTimeout, LongPollingInterval).Should(BeClosed())
		helpers.WaitForFab3(user1ProxyPort)

		By("starting up a fab3 for user 2")
		user2ProxyPort := uint16(3000 + config.GinkgoConfig.ParallelNode)
		user2ProxyRunner := helpers.Fab3Runner(components.Paths["fab3"], components.Paths["Fab3Config"], "Org1", "User2", channelName, ccid, user2ProxyPort)
		user2Proxy = ifrit.Invoke(user2ProxyRunner)
		Eventually(user2Proxy.Ready(), LongEventualTimeout, LongPollingInterval).Should(BeClosed())
		helpers.WaitForFab3(user2ProxyPort)

		By("running the web3 tests")
		web3TestRunner := helpers.Web3TestRunner(
			fmt.Sprintf("http://127.0.0.1:%d", user1ProxyPort),
			fmt.Sprintf("http://127.0.0.1:%d", user2ProxyPort),
		)

		web3Process := ifrit.Invoke(web3TestRunner)
		Eventually(web3Process.Ready()).Should(BeClosed())

		// This runs the entire web3 test in one shot, and since go receives no input during the test run until
		// the test finally ends, the timeout should be especially long, and we should poll rather infrequently
		Eventually(web3Process.Wait(), Web3EventuallyTimeout, Web3EventuallyPollingInterval).Should(Receive())
		Expect(web3TestRunner.ExitCode()).Should(Equal(0))

		Expect(web3TestRunner.Buffer()).To(gbytes.Say("Successfully able to deploy Voting Smart Contract and interact with it"))
	})
})
