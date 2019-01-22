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
	"strings"
	"syscall"

	"github.com/hyperledger/fabric-chaincode-evm/integration/helpers"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
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
	req, err := http.NewRequest("POST", proxyAddress, body)
	Expect(err).ToNot(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")

	return client.Do(req)
}

var _ = Describe("Fab3", func() {
	var (
		proxy         ifrit.Process
		proxyAddress  string
		client        *http.Client
		SimpleStorage helpers.Contract
	)

	BeforeEach(func() {
		SimpleStorage = helpers.SimpleStorageContract()
		client = &http.Client{}

		//Start up Proxy
		proxyPort := uint16(6000 + config.GinkgoConfig.ParallelNode)
		proxyRunner := helpers.Fab3Runner(components.Paths["fab3"], components.Paths["Fab3Config"], "Org1", "User1", channelName, ccid, proxyPort)
		proxy = ifrit.Invoke(proxyRunner)
		Eventually(proxy.Ready(), LongEventualTimeout, LongPollingInterval).Should(BeClosed())
		proxyAddress = fmt.Sprintf("http://127.0.0.1:%d", proxyPort)
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

			err = json.Unmarshal(rBody, &rpcResp)
			Expect(err).ToNot(HaveOccurred())
			return rpcResp.Error
		}, LongEventualTimeout, LongPollingInterval).Should(BeZero())

		receipt := rpcResp.Result

		Expect(receipt.ContractAddress).ToNot(Equal(""))
		Expect(receipt.TransactionHash).To(Equal("0x" + txHash))
		checkHexEncoded(receipt.BlockNumber)
		checkHexEncoded(receipt.BlockHash)
		checkHexEncoded(receipt.TransactionIndex)

		By("verifying the code")
		contractAddr := receipt.ContractAddress
		resp, err = sendRPCRequest(client, "eth_getCode", proxyAddress, 17, []string{contractAddr})
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

			err = json.Unmarshal(rBody, &rpcResp)
			Expect(err).ToNot(HaveOccurred())
			return rpcResp.Error
		}, LongEventualTimeout, LongPollingInterval).Should(BeZero())
		receipt = rpcResp.Result
		Expect(receipt.TransactionHash).To(Equal("0x" + txHash))
		checkHexEncoded(receipt.BlockNumber)
		checkHexEncoded(receipt.BlockHash)
		checkHexEncoded(receipt.TransactionIndex)
		Expect(receipt.ContractAddress).To(Equal(""))

		By("querying the contract")
		params = helpers.MessageParams{
			To:   contractAddr,
			Data: SimpleStorage.FunctionHashes["get"],
		}
		resp, err = sendRPCRequest(client, "eth_call", proxyAddress, 19, params)
		rBody, err = ioutil.ReadAll(resp.Body)
		Expect(err).ToNot(HaveOccurred())

		err = json.Unmarshal(rBody, &respBody)
		Expect(err).ToNot(HaveOccurred())

		Expect(respBody.Error).To(BeZero())
		Expect(respBody.Result).To(Equal("0x" + val))
	})
})

func checkHexEncoded(value string) {
	// Check to see that the result is a hexadecimal string
	// Check if the prefix is provided
	Expect(value[0:2]).To(Equal("0x"))

	// Ensure the string is hex
	Expect(value).To(MatchRegexp(fmt.Sprintf(`[0-9A-Fa-f]{%d}`, len(value[2:]))))
}
