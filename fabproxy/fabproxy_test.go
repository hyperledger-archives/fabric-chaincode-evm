/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabproxy_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"

	"github.com/hyperledger/fabric-chaincode-evm/fabproxy"
	"github.com/hyperledger/fabric-chaincode-evm/mocks"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
)

var _ = Describe("Fabproxy", func() {

	var (
		proxy          *fabproxy.FabProxy
		proxyAddr      string
		mockEthService *mocks.MockEthService
		req            *http.Request
		proxyDoneChan  chan struct{}
		client         *http.Client
	)

	BeforeEach(func() {
		port := config.GinkgoConfig.ParallelNode + 5000
		mockEthService = &mocks.MockEthService{}
		client = &http.Client{}

		proxyDoneChan = make(chan struct{}, 1)
		proxy = fabproxy.NewFabProxy(mockEthService)

		go func(proxy *fabproxy.FabProxy, proxyDoneChan chan struct{}) {
			proxy.Start(port)

			// Close proxy done chan to signify proxy has exited
			close(proxyDoneChan)
		}(proxy, proxyDoneChan)

		Eventually(proxyDoneChan).ShouldNot(Receive())

		proxyAddr = fmt.Sprintf("http://localhost:%d", port)

		//Ensure the server is up before starting the test
		Eventually(func() error {
			conn, err := net.Dial("tcp", "golang.org:80")
			defer conn.Close()
			return err
		}).Should(Succeed())

		mockEthService.GetCodeStub = func(r *http.Request, arg *string, reply *string) error {
			*reply = "0x11110"
			return nil
		}

		//curl -X POST --data '{"jsonrpc":"2.0","method":"eth_getCode","params":["0xa94f5374fce5edbc8e2a8697c15331677e6ebf0b", "0x2"],"id":1}'
		var err error
		body := strings.NewReader(`{"jsonrpc":"2.0","method":"eth_getCode","params":["0xa94f5374fce5edbc8e2a8697c15331677e6ebf0b"],"id":1}`)
		req, err = http.NewRequest("POST", proxyAddr, body)
		Expect(err).ToNot(HaveOccurred())
		req.Header.Set("Content-Type", "application/json")
	})

	AfterEach(func() {
		err := proxy.Shutdown()
		Expect(err).ToNot(HaveOccurred())

		Eventually(proxyDoneChan).Should(BeClosed())
	})

	It("starts a server that uses the provided ethservice", func() {
		resp, err := client.Do(req)
		Expect(err).ToNot(HaveOccurred())

		type responseBody struct {
			JsonRPC string `json:"jsonrpc"`
			ID      int    `json:"id"`
			Result  string `json:"result"`
		}
		expectedBody := responseBody{JsonRPC: "2.0", ID: 1, Result: "0x11110"}

		rBody, err := ioutil.ReadAll(resp.Body)
		Expect(err).ToNot(HaveOccurred())

		var respBody responseBody
		err = json.Unmarshal(rBody, &respBody)
		Expect(err).ToNot(HaveOccurred())

		Expect(respBody).To(Equal(expectedBody))
	})
})
