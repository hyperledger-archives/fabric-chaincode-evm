/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fab3

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"syscall"

	"github.com/hyperledger/fabric-chaincode-evm/fab3/types"
	"github.com/hyperledger/fabric-chaincode-evm/integration/helpers"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"
)

func sendRPCRequest(client *http.Client, method, proxyAddress string, id int, params interface{}) (*http.Response, error) {
	request := helpers.JsonRPCRequest{
		JsonRPC: "2.0",
		Method:  method,
		ID:      id,
		Params:  params,
	}
	reqBody, err := json.Marshal(request)
	Expect(err).ToNot(HaveOccurred())

	body := strings.NewReader(string(reqBody))
	fmt.Fprintln(GinkgoWriter, string(reqBody))
	req, err := http.NewRequest("POST", proxyAddress, body)
	Expect(err).ToNot(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")

	return client.Do(req)
}

var _ = Describe("Fab3", func() {
	var (
		proxy         ifrit.Process
		proxyRunner   *ginkgomon.Runner
		proxyAddress  string
		client        *http.Client
		SimpleStorage helpers.Contract
	)

	BeforeEach(func() {
		SimpleStorage = helpers.SimpleStorageContract()
		client = &http.Client{}

		//Start up Proxy
		proxyPort := uint16(6000 + config.GinkgoConfig.ParallelNode)
		proxyRunner = helpers.Fab3Runner(components.Paths["fab3"], components.Paths["Fab3Config"], "Org1", "User1", channelName, ccid, proxyPort)
		proxy = ifrit.Invoke(proxyRunner)
		Eventually(proxy.Ready(), LongEventualTimeout, LongPollingInterval).Should(BeClosed())
		proxyAddress = fmt.Sprintf("http://127.0.0.1:%d", proxyPort)
		helpers.WaitForFab3(proxyPort)
	})

	AfterEach(func() {
		if proxy != nil {
			proxy.Signal(syscall.SIGTERM)
			Eventually(proxy.Wait(), LongEventualTimeout, LongPollingInterval).Should(Receive())
		}
	})

	It("implements the ethereum json rpc api", func() {
		By("querying for an account")
		resp, err := sendRPCRequest(client, "eth_accounts", proxyAddress, 5, []interface{}{})
		Expect(err).ToNot(HaveOccurred())

		expectedArrayBody := helpers.JsonRPCArrayResponse{JsonRPC: "2.0", ID: 5}

		rBody, err := ioutil.ReadAll(resp.Body)
		Expect(err).ToNot(HaveOccurred())

		var respArrayBody helpers.JsonRPCArrayResponse
		err = json.Unmarshal(rBody, &respArrayBody)
		Expect(err).ToNot(HaveOccurred())

		Expect(respArrayBody.Error).To(BeZero())

		Expect(respArrayBody.Result).To(HaveLen(1))
		account := respArrayBody.Result[0]

		checkHexEncoded(account)
		// Set the same result so that next expectation can check all other fields
		expectedArrayBody.Result = respArrayBody.Result
		Expect(respArrayBody).To(Equal(expectedArrayBody))

		By("Deploying the Simple Storage Contract")
		params := helpers.MessageParams{
			To:   "0000000000000000000000000000000000000000",
			Data: SimpleStorage.CompiledBytecode,
		}

		resp, err = sendRPCRequest(client, "eth_sendTransaction", proxyAddress, 6, params)
		Expect(err).ToNot(HaveOccurred())

		expectedBody := helpers.JsonRPCResponse{JsonRPC: "2.0", ID: 6}
		rBody, err = ioutil.ReadAll(resp.Body)
		Expect(err).ToNot(HaveOccurred())

		var respBody helpers.JsonRPCResponse
		err = json.Unmarshal(rBody, &respBody)
		Expect(err).ToNot(HaveOccurred())

		Expect(respBody.Error).To(BeZero())

		// Set the same result so that next expectation can check all other fields
		expectedBody.Result = respBody.Result
		Expect(respBody).To(Equal(expectedBody))

		By("Getting the Transaction Receipt")
		txHash := respBody.Result
		var rpcResp helpers.JsonRPCTxReceipt

		// It takes a couple seconds for the transaction to be found
		Eventually(func() helpers.JsonRPCError {
			resp, err = sendRPCRequest(client, "eth_getTransactionReceipt", proxyAddress, 16, []string{txHash})
			Expect(err).ToNot(HaveOccurred())

			rBody, err = ioutil.ReadAll(resp.Body)
			Expect(err).ToNot(HaveOccurred())
			rpcResp = helpers.JsonRPCTxReceipt{}
			err = json.Unmarshal(rBody, &rpcResp)
			Expect(err).ToNot(HaveOccurred())
			return rpcResp.Error
		}, LongEventualTimeout, LongPollingInterval).Should(BeZero())

		receipt := rpcResp.Result

		checkHexEncoded(receipt.ContractAddress)
		Expect(receipt.TransactionHash).To(Equal("0x" + txHash))
		checkHexEncoded(receipt.BlockNumber)
		checkHexEncoded(receipt.BlockHash)
		checkHexEncoded(receipt.TransactionIndex)

		By("verifying the code")
		contractAddr := receipt.ContractAddress
		resp, err = sendRPCRequest(client, "eth_getCode", proxyAddress, 17, []string{contractAddr})
		Expect(err).ToNot(HaveOccurred())

		rBody, err = ioutil.ReadAll(resp.Body)
		Expect(err).ToNot(HaveOccurred())

		err = json.Unmarshal(rBody, &respBody)
		Expect(err).ToNot(HaveOccurred())

		Expect(rpcResp.Error).To(BeZero())
		Expect(respBody.Result).To(Equal(SimpleStorage.RuntimeBytecode))

		By("interacting with the contract")
		val := "000000000000000000000000000000000000000000000000000000000000002a"
		params = helpers.MessageParams{
			To:   contractAddr,
			Data: SimpleStorage.FunctionHashes["set"] + val,
		}
		resp, err = sendRPCRequest(client, "eth_sendTransaction", proxyAddress, 18, params)
		Expect(err).ToNot(HaveOccurred())

		rBody, err = ioutil.ReadAll(resp.Body)
		Expect(err).ToNot(HaveOccurred())

		err = json.Unmarshal(rBody, &respBody)
		Expect(err).ToNot(HaveOccurred())

		Expect(respBody.Error).To(BeZero())
		txHash = respBody.Result

		By("verifying it returned a valid transaction hash")
		Eventually(func() helpers.JsonRPCError {
			resp, err = sendRPCRequest(client, "eth_getTransactionReceipt", proxyAddress, 16, []string{txHash})
			Expect(err).ToNot(HaveOccurred())

			rBody, err = ioutil.ReadAll(resp.Body)
			Expect(err).ToNot(HaveOccurred())
			rpcResp = helpers.JsonRPCTxReceipt{}
			err = json.Unmarshal(rBody, &rpcResp)
			Expect(err).ToNot(HaveOccurred())
			return rpcResp.Error
		}, LongEventualTimeout, LongPollingInterval).Should(BeZero())
		receipt = rpcResp.Result
		Expect(receipt.TransactionHash).To(Equal("0x" + txHash))
		checkHexEncoded(receipt.BlockNumber)
		checkHexEncoded(receipt.BlockHash)
		checkHexEncoded(receipt.TransactionIndex)
		Expect(receipt.ContractAddress).To(BeEmpty())

		By("querying the contract")
		params = helpers.MessageParams{
			To:   contractAddr,
			Data: SimpleStorage.FunctionHashes["get"],
		}
		resp, err = sendRPCRequest(client, "eth_call", proxyAddress, 19, params)
		Expect(err).ToNot(HaveOccurred())

		rBody, err = ioutil.ReadAll(resp.Body)
		Expect(err).ToNot(HaveOccurred())

		err = json.Unmarshal(rBody, &respBody)
		Expect(err).ToNot(HaveOccurred())

		Expect(respBody.Error).To(BeZero())
		Expect(respBody.Result).To(Equal("0x" + val))

		By("querying the latest block number")
		resp, err = sendRPCRequest(client, "eth_blockNumber", proxyAddress, 20, []interface{}{})
		Expect(err).ToNot(HaveOccurred())

		rBody, err = ioutil.ReadAll(resp.Body)
		Expect(err).ToNot(HaveOccurred())

		err = json.Unmarshal(rBody, &respBody)
		Expect(err).ToNot(HaveOccurred())

		Expect(respBody.Error).To(BeZero())
		Expect(respBody.Result).To(Equal(receipt.BlockNumber))
		checkHexEncoded(respBody.Result)

		By("querying for logs of a transaction with no logs, we get no logs")
		resp, err = sendRPCRequest(client, "eth_getLogs", proxyAddress, 20, []interface{}{})
		Expect(err).ToNot(HaveOccurred())
		rBody, err = ioutil.ReadAll(resp.Body)
		Expect(err).ToNot(HaveOccurred())
		err = json.Unmarshal(rBody, &respArrayBody)
		Expect(err).ToNot(HaveOccurred())
		Expect(respArrayBody.Result).To(HaveLen(0))

		By("querying for logs of the genesis block, we get no logs")
		resp, err = sendRPCRequest(client, "eth_getLogs", proxyAddress,
			28, types.GetLogsArgs{FromBlock: "earliest", ToBlock: "0x0"})
		Expect(err).ToNot(HaveOccurred())
		rBody, err = ioutil.ReadAll(resp.Body)
		fmt.Fprintln(GinkgoWriter, string(rBody))
		Expect(err).ToNot(HaveOccurred())
		err = json.Unmarshal(rBody, &respArrayBody)
		Expect(err).ToNot(HaveOccurred(), "problem unmarshalling", string(rBody))
		Expect(respArrayBody.Result).To(HaveLen(0))
	})

	It("implements the ethereum json rpc api with logs", func() {
		By("Deploying the Simple Storage With Logs Contract")
		params := helpers.MessageParams{
			To:   "0000000000000000000000000000000000000000",
			Data: helpers.SimpleStorageWithLog().CompiledBytecode,
		}

		resp, err := sendRPCRequest(client, "eth_sendTransaction", proxyAddress, 6, params)
		Expect(err).ToNot(HaveOccurred())

		rBody, err := ioutil.ReadAll(resp.Body)
		Expect(err).ToNot(HaveOccurred())

		var respBody helpers.JsonRPCResponse
		err = json.Unmarshal(rBody, &respBody)
		Expect(err).ToNot(HaveOccurred())

		Expect(respBody.Error).To(BeZero())
		txHash := respBody.Result

		// It takes a couple seconds for the transaction to be found
		var rpcResp helpers.JsonRPCTxReceipt
		Eventually(func() helpers.JsonRPCError {
			resp, err = sendRPCRequest(client, "eth_getTransactionReceipt", proxyAddress, 16, []string{txHash})
			Expect(err).ToNot(HaveOccurred())

			rBody, err = ioutil.ReadAll(resp.Body)
			Expect(err).ToNot(HaveOccurred())
			rpcResp = helpers.JsonRPCTxReceipt{}
			err = json.Unmarshal(rBody, &rpcResp)
			Expect(err).ToNot(HaveOccurred())
			return rpcResp.Error
		}, LongEventualTimeout, LongPollingInterval).Should(BeZero())
		receipt := rpcResp.Result
		contractAddr := receipt.ContractAddress
		By("interacting with the contract")
		val := "3737373737373737373737373737373737373737373737373737373737373737"
		params = helpers.MessageParams{
			To:   contractAddr,
			Data: helpers.SimpleStorageWithLog().FunctionHashes["set"] + val,
		}
		resp, err = sendRPCRequest(client, "eth_sendTransaction", proxyAddress, 18, params)
		rBody, err = ioutil.ReadAll(resp.Body)
		Expect(err).ToNot(HaveOccurred())

		err = json.Unmarshal(rBody, &respBody)
		Expect(err).ToNot(HaveOccurred())

		Expect(respBody.Error).To(BeZero())
		txHash = respBody.Result

		// It takes a couple seconds for the transaction to be found
		Eventually(func() helpers.JsonRPCError {
			resp, err = sendRPCRequest(client, "eth_getTransactionReceipt", proxyAddress, 16, []string{txHash})
			Expect(err).ToNot(HaveOccurred())

			rBody, err = ioutil.ReadAll(resp.Body)
			Expect(err).ToNot(HaveOccurred())
			rpcResp = helpers.JsonRPCTxReceipt{}
			err = json.Unmarshal(rBody, &rpcResp)
			Expect(err).ToNot(HaveOccurred())
			return rpcResp.Error
		}, LongEventualTimeout, LongPollingInterval).Should(BeZero())

		By("querying for logs of a contract with logs, we get a log")
		resp, err = sendRPCRequest(client, "eth_getLogs", proxyAddress, 23,
			types.GetLogsArgs{ToBlock: "latest", // FromBlock as a default argument
				Address: types.AddressFilter{contractAddr},
				Topics: types.TopicsFilter{
					types.TopicFilter{}, // a null topic filter, and a no 0x prefix topic
					types.TopicFilter{"0000000000000000000000000000000000000000000000000000000000000000"}}})
		Expect(err).ToNot(HaveOccurred())
		rBody, err = ioutil.ReadAll(resp.Body)
		Expect(err).ToNot(HaveOccurred())

		var respArrayBody helpers.JsonRPCLogArrayResponse
		err = json.Unmarshal(rBody, &respArrayBody)
		Expect(err).ToNot(HaveOccurred())
		logs := respArrayBody.Result
		Expect(logs).To(HaveLen(1), "this contract only emits one log entry")
		log := logs[0]
		Expect(log.Address).To(Equal(contractAddr), "logs come from the contract")
		Expect(log.Topics).To(HaveLen(3))
		Expect(log.Index).To(Equal("0x0"))
		topics := log.Topics
		// topics, are hash of function, 0 from initial setting, & the value we set earlier
		Expect(topics[0]).To(Equal("0xd81ec364c58bcc9b49b6c953fc8e1f1c158ee89255bae73029133234a2936aad"))
		Expect(topics[1]).To(Equal("0x0000000000000000000000000000000000000000000000000000000000000000"))
		Expect(topics[2]).To(Equal("0x" + val))
		txReceipt := rpcResp.Result
		Expect(log.BlockNumber).To(Equal(txReceipt.BlockNumber))
		Expect(log.BlockHash).To(Equal(txReceipt.BlockHash))
		Expect(log.TxIndex).To(Equal(txReceipt.TransactionIndex))
		Expect(log.TxHash).To(Equal(txReceipt.TransactionHash))
	})

	It("shuts down gracefully when it receives an Interrupt signal", func() {
		proxy.Signal(os.Interrupt)

		Eventually(proxy.Wait()).Should(Receive())
		Eventually(proxyRunner.Err()).Should(gbytes.Say("Fab3 exited"))
	})

	It("shuts down gracefully when it receives an SIGTERM signal", func() {
		proxy.Signal(syscall.SIGTERM)

		Eventually(proxy.Wait()).Should(Receive())
		Eventually(proxyRunner.Err()).Should(gbytes.Say("Fab3 exited"))
	})
})

func checkHexEncoded(value string) {
	// Check to see that the result is a hexadecimal string
	// Check if the prefix is provided
	Expect(value[0:2]).To(Equal("0x"))

	// Check that the string is not empty
	Expect(len(value)).To(BeNumerically(">=", 3), value+" is an empty hex string")

	// Ensure the string is hex
	Expect(value).To(MatchRegexp(fmt.Sprintf(`[0-9A-Fa-f]{%d}`, len(value[2:]))))
}
