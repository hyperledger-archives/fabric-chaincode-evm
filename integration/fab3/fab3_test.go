/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fab3

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
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

func sendRPCRequest(client *http.Client, method, proxyAddress string, id int, params interface{}) []byte {
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

	resp, err := client.Do(req)

	Expect(err).ToNot(HaveOccurred())
	responseBody, err := ioutil.ReadAll(resp.Body)
	fmt.Fprintln(GinkgoWriter, string(responseBody))
	Expect(err).ToNot(HaveOccurred())
	return responseBody
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
		var err error
		var respBody helpers.JsonRPCResponse
		var respBlockBody helpers.JsonRPCBlockResponse
		var respArrayBody helpers.JsonRPCArrayResponse

		By("querying for an account")
		rBody := sendRPCRequest(client, "eth_accounts", proxyAddress, 5, []interface{}{})
		err = json.Unmarshal(rBody, &respArrayBody)
		Expect(err).ToNot(HaveOccurred())
		Expect(respArrayBody.Error).To(BeZero())
		Expect(respArrayBody.Result).To(HaveLen(1))
		account := respArrayBody.Result[0]
		checkHexEncoded(account)

		expectedArrayBody := helpers.JsonRPCArrayResponse{JsonRPC: "2.0", ID: 5}
		// Set the same result so that next expectation can check all other fields
		expectedArrayBody.Result = respArrayBody.Result
		Expect(respArrayBody).To(Equal(expectedArrayBody))

		By("Deploying the Simple Storage Contract")
		params := helpers.MessageParams{
			To:   "0000000000000000000000000000000000000000",
			Data: SimpleStorage.CompiledBytecode,
		}

		rBody = sendRPCRequest(client, "eth_sendTransaction", proxyAddress, 6, params)
		err = json.Unmarshal(rBody, &respBody)
		Expect(err).ToNot(HaveOccurred())

		Expect(respBody.Error).To(BeZero())
		expectedBody := helpers.JsonRPCResponse{JsonRPC: "2.0", ID: 6}
		// Set the same result so that next expectation can check all other fields
		expectedBody.Result = respBody.Result
		Expect(respBody).To(Equal(expectedBody))

		By("Getting the Transaction Receipt")
		txHash := respBody.Result
		var rpcResp helpers.JsonRPCTxReceipt

		// It takes a couple seconds for the transaction to be found
		Eventually(func() helpers.JsonRPCError {
			rBody = sendRPCRequest(client, "eth_getTransactionReceipt", proxyAddress, 16, []string{txHash})
			rpcResp = helpers.JsonRPCTxReceipt{}
			err = json.Unmarshal(rBody, &rpcResp)
			Expect(err).ToNot(HaveOccurred())
			return rpcResp.Error
		}, LongEventualTimeout, LongPollingInterval).Should(BeZero())

		receipt := rpcResp.Result

		Expect(receipt.Logs).ToNot(BeNil())
		checkHexEncoded(receipt.ContractAddress)
		Expect(receipt.TransactionHash).To(Equal("0x" + txHash))
		checkHexEncoded(receipt.BlockNumber)
		checkHexEncoded(receipt.BlockHash)
		checkHexEncoded(receipt.TransactionIndex)
		Expect(receipt.From).To(Equal(account))

		By("verifying the code")
		contractAddr := receipt.ContractAddress
		rBody = sendRPCRequest(client, "eth_getCode", proxyAddress, 17, []string{contractAddr})
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
		rBody = sendRPCRequest(client, "eth_sendTransaction", proxyAddress, 18, params)
		Expect(err).ToNot(HaveOccurred())

		err = json.Unmarshal(rBody, &respBody)
		Expect(err).ToNot(HaveOccurred())

		Expect(respBody.Error).To(BeZero())
		txHash = respBody.Result

		By("verifying it returned a valid transaction hash")
		Eventually(func() helpers.JsonRPCError {
			rBody = sendRPCRequest(client, "eth_getTransactionReceipt", proxyAddress, 16, []string{txHash})
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
		Expect(receipt.From).To(Equal(account))

		By("querying the contract")
		params = helpers.MessageParams{
			To:   contractAddr,
			Data: SimpleStorage.FunctionHashes["get"],
		}
		rBody = sendRPCRequest(client, "eth_call", proxyAddress, 19, params)
		Expect(err).ToNot(HaveOccurred())

		err = json.Unmarshal(rBody, &respBody)
		Expect(err).ToNot(HaveOccurred())

		Expect(respBody.Error).To(BeZero())
		Expect(respBody.Result).To(Equal("0x" + val))

		By("querying the latest block number")
		rBody = sendRPCRequest(client, "eth_blockNumber", proxyAddress, 20, []interface{}{})
		Expect(err).ToNot(HaveOccurred())

		err = json.Unmarshal(rBody, &respBody)
		Expect(err).ToNot(HaveOccurred())

		Expect(respBody.Error).To(BeZero())
		Expect(respBody.Result).To(Equal(receipt.BlockNumber))
		checkHexEncoded(respBody.Result)

		By("getting the block by number we just got")
		rBody = sendRPCRequest(client, "eth_getBlockByNumber", proxyAddress, rand.Int(), []interface{}{receipt.BlockNumber, false})

		err = json.Unmarshal(rBody, &respBlockBody)
		Expect(err).ToNot(HaveOccurred())
		latestBlock := respBlockBody.Result
		Expect(latestBlock.Number).To(Equal(receipt.BlockNumber))

		By("querying for logs of a transaction with no logs, we get no logs")
		rBody = sendRPCRequest(client, "eth_getLogs", proxyAddress, 20, []interface{}{})

		err = json.Unmarshal(rBody, &respArrayBody)
		Expect(err).ToNot(HaveOccurred())
		Expect(respArrayBody.Result).To(HaveLen(0))

		By("querying for logs of the genesis block, we get no logs")
		rBody = sendRPCRequest(client, "eth_getLogs", proxyAddress,
			28, types.GetLogsArgs{FromBlock: "earliest", ToBlock: "0x0"})
		fmt.Fprintln(GinkgoWriter, string(rBody))
		err = json.Unmarshal(rBody, &respArrayBody)
		Expect(err).ToNot(HaveOccurred(), "problem unmarshalling", string(rBody))
		Expect(respArrayBody.Result).To(HaveLen(0))
	})

	It("implements the ethereum json rpc api with logs", func() {
		var err error
		var rBody []byte
		var respBody helpers.JsonRPCResponse
		var rpcResp helpers.JsonRPCTxReceipt
		var respArrayBody helpers.JsonRPCLogArrayResponse

		By("Deploying the Simple Storage With Logs Contract")
		params := helpers.MessageParams{
			To:   "0000000000000000000000000000000000000000",
			Data: helpers.SimpleStorageWithLog().CompiledBytecode,
		}

		rBody = sendRPCRequest(client, "eth_sendTransaction", proxyAddress, 6, params)
		err = json.Unmarshal(rBody, &respBody)
		Expect(err).ToNot(HaveOccurred())

		Expect(respBody.Error).To(BeZero())
		txHash := respBody.Result

		// It takes a couple seconds for the transaction to be found
		Eventually(func() helpers.JsonRPCError {
			rBody = sendRPCRequest(client, "eth_getTransactionReceipt", proxyAddress, 16, []string{txHash})
			rpcResp = helpers.JsonRPCTxReceipt{}
			err = json.Unmarshal(rBody, &rpcResp)
			Expect(err).ToNot(HaveOccurred())
			return rpcResp.Error
		}, LongEventualTimeout, LongPollingInterval).Should(BeZero())
		receipt := rpcResp.Result
		contractAddr := receipt.ContractAddress

		By("using one log arg for both log gets")
		logFilter := types.GetLogsArgs{ToBlock: "latest", // FromBlock as a default argument
			Address: types.AddressFilter{contractAddr},
			Topics: types.TopicsFilter{
				types.TopicFilter{}, // a null topic filter, and a no 0x prefix topic
				types.TopicFilter{"0000000000000000000000000000000000000000000000000000000000000000"}}}

		By("starting a logs filter before creating new blocks with logs")
		rBody = sendRPCRequest(client, "eth_newFilter", proxyAddress, rand.Int(), logFilter)
		err = json.Unmarshal(rBody, &respBody)
		Expect(err).ToNot(HaveOccurred())
		// logsFilterID := respBody.Result

		By("interacting with the contract")
		val := "3737373737373737373737373737373737373737373737373737373737373737"
		params = helpers.MessageParams{
			To:   contractAddr,
			Data: helpers.SimpleStorageWithLog().FunctionHashes["set"] + val,
		}
		rBody = sendRPCRequest(client, "eth_sendTransaction", proxyAddress, 18, params)

		err = json.Unmarshal(rBody, &respBody)
		Expect(err).ToNot(HaveOccurred())

		Expect(respBody.Error).To(BeZero())
		txHash = respBody.Result

		// It takes a couple seconds for the transaction to be found
		Eventually(func() helpers.JsonRPCError {
			rBody = sendRPCRequest(client, "eth_getTransactionReceipt", proxyAddress, 16, []string{txHash})
			rpcResp = helpers.JsonRPCTxReceipt{}
			err = json.Unmarshal(rBody, &rpcResp)
			Expect(err).ToNot(HaveOccurred())
			return rpcResp.Error
		}, LongEventualTimeout, LongPollingInterval).Should(BeZero())

		By("querying for logs of a contract with logs, we get a log")
		rBody = sendRPCRequest(client, "eth_getLogs", proxyAddress, 23, logFilter)

		err = json.Unmarshal(rBody, &respArrayBody)
		Expect(err).ToNot(HaveOccurred())
		logs := respArrayBody.Result
		Expect(logs).To(HaveLen(1), "this contract only emits one log entry")
		log := logs[0]
		Expect(log.Address).To(Equal(contractAddr), "logs come from the contract")
		Expect(log.Topics).To(HaveLen(3))
		Expect(log.Index).To(Equal("0x0"))
		topics := log.Topics
		// topics are, hash of event signature, 0 from initial setting, & the value we set earlier
		Expect(topics[0]).To(Equal("0xd81ec364c58bcc9b49b6c953fc8e1f1c158ee89255bae73029133234a2936aad"))
		Expect(topics[1]).To(Equal("0x0000000000000000000000000000000000000000000000000000000000000000"))
		Expect(topics[2]).To(Equal("0x" + val))
		txReceipt := rpcResp.Result
		Expect(log.BlockNumber).To(Equal(txReceipt.BlockNumber))
		Expect(log.BlockHash).To(Equal(txReceipt.BlockHash))
		Expect(log.TxIndex).To(Equal(txReceipt.TransactionIndex))
		Expect(log.TxHash).To(Equal(txReceipt.TransactionHash))

		By("doing the same thing with the known blockhash instead of block parameters, we get the same results")
		rBody = sendRPCRequest(client, "eth_getLogs", proxyAddress, 23,
			types.GetLogsArgs{BlockHash: txReceipt.BlockHash,
				Address: types.AddressFilter{contractAddr},
				Topics: types.TopicsFilter{
					types.TopicFilter{}, // a null topic filter, and a no 0x prefix topic
					types.TopicFilter{"0000000000000000000000000000000000000000000000000000000000000000"}}})

		respArrayBody = helpers.JsonRPCLogArrayResponse{}
		err = json.Unmarshal(rBody, &respArrayBody)
		Expect(err).ToNot(HaveOccurred())
		logs = respArrayBody.Result
		Expect(logs).To(HaveLen(1), "this contract only emits one log entry")
		log = logs[0]
		Expect(log.Address).To(Equal(contractAddr), "logs come from the contract")
		Expect(log.Topics).To(HaveLen(3))
		Expect(log.Index).To(Equal("0x0"))
		topics = log.Topics
		// topics are, hash of event signature, 0 from initial setting, & the value we set earlier
		Expect(topics[0]).To(Equal("0xd81ec364c58bcc9b49b6c953fc8e1f1c158ee89255bae73029133234a2936aad"))
		Expect(topics[1]).To(Equal("0x0000000000000000000000000000000000000000000000000000000000000000"))
		Expect(topics[2]).To(Equal("0x" + val))
		txReceipt = rpcResp.Result
		Expect(log.BlockNumber).To(Equal(txReceipt.BlockNumber))
		Expect(log.BlockHash).To(Equal(txReceipt.BlockHash))
		Expect(log.TxIndex).To(Equal(txReceipt.TransactionIndex))
		Expect(log.TxHash).To(Equal(txReceipt.TransactionHash))
	})

	It("implements the ethereum json rpc api for async-logs", func() {
		var err error
		var respBody helpers.JsonRPCResponse
		var uninstallBody helpers.JsonRPCBoolResponse

		rBody := sendRPCRequest(client, "eth_newFilter", proxyAddress, 37, []interface{}{})
		err = json.Unmarshal(rBody, &respBody)
		Expect(err).ToNot(HaveOccurred())
		filterID := respBody.Result

		rBody = sendRPCRequest(client, "eth_uninstallFilter", proxyAddress, 38, filterID)
		err = json.Unmarshal(rBody, &uninstallBody)
		Expect(err).ToNot(HaveOccurred())
		Expect(uninstallBody.Result).To(BeTrue())

		uninstallBody = helpers.JsonRPCBoolResponse{}
		rBody = sendRPCRequest(client, "eth_uninstallFilter", proxyAddress, 39, filterID)
		err = json.Unmarshal(rBody, &uninstallBody)
		Expect(err).ToNot(HaveOccurred())
		Expect(uninstallBody.Result).To(BeFalse(), "filter just now removed")
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
