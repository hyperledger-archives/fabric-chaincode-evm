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
	"strconv"
	"strings"

	"github.com/hyperledger/fabric-chaincode-evm/fabproxy"
	fabproxy_mocks "github.com/hyperledger/fabric-chaincode-evm/mocks/fabproxy"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
)

var _ = Describe("Fabproxy", func() {

	var (
		proxy          *fabproxy.FabProxy
		proxyAddr      string
		mockEthService *fabproxy_mocks.MockEthService
		req            *http.Request
		proxyDoneChan  chan struct{}
		client         *http.Client
		port           int
	)

	BeforeEach(func() {
		port = config.GinkgoConfig.ParallelNode + 5000
		mockEthService = &fabproxy_mocks.MockEthService{}
		client = &http.Client{}

		proxyDoneChan = make(chan struct{}, 1)
		var err error
		proxy = fabproxy.NewFabProxy(mockEthService)
		Expect(err).ToNot(HaveOccurred())

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

	It("starts a server that uses the hardcoded netservice", func() {
		var err error
		body := strings.NewReader(`{"jsonrpc":"2.0","method":"net_version","id":1}`)
		req, err = http.NewRequest("POST", proxyAddr, body)
		Expect(err).ToNot(HaveOccurred())

		resp, err := client.Do(req)
		Expect(err).ToNot(HaveOccurred())

		type responseBody struct {
			JsonRPC string `json:"jsonrpc"`
			ID      int    `json:"id"`
			Result  string `json:"result"`
		}
		expectedBody := responseBody{JsonRPC: "2.0", ID: 1, Result: "fabric-evm"}

		rBody, err := ioutil.ReadAll(resp.Body)
		Expect(err).ToNot(HaveOccurred())

		var respBody responseBody
		err = json.Unmarshal(rBody, &respBody)
		Expect(err).ToNot(HaveOccurred())

		Expect(respBody).To(Equal(expectedBody))
	})

	Context("when the request has Cross-Origin Resource Sharing Headers", func() {
		BeforeEach(func() {
			var err error
			body := strings.NewReader("")
			//OPTIONS pre-check used to see the CORS options of the server
			req, err = http.NewRequest("OPTIONS", proxyAddr, body)
			Expect(err).ToNot(HaveOccurred())
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Origin", "[http://example.com]")
		})

		Context("when the request method is POST", func() {
			BeforeEach(func() {
				//curl -X OPTIONS http://localhost:5000 -H "Origin: http://example.com" -H "Access-Control-Request-Method: POST"
				req.Header.Set("Access-Control-Request-Method", "POST")
			})

			It("successfully processes the request", func() {
				client := &http.Client{}
				resp, err := client.Do(req)
				Expect(err).ToNot(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(resp.Header.Get("Access-Control-Allow-Origin")).To(Equal("*"))
			})
		})

		Context("when the request method is not POST", func() {
			BeforeEach(func() {
				//curl -X OPTIONS http://localhost:5000 -H "Origin: http://example.com" -H "Access-Control-Request-Method: GET"
				req.Header.Set("Access-Control-Request-Method", "GET")
			})

			It("successfully processes the request", func() {
				client := &http.Client{}
				resp, err := client.Do(req)
				Expect(err).ToNot(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusMethodNotAllowed))
			})
		})
	})
})

var _ = Describe("fabproxy fails to start", func() {
	var (
		ln  net.Listener
		err error
	)
	Context("when the port is already bound", func() {

		port := config.GinkgoConfig.ParallelNode + 5000
		portstr := strconv.Itoa(port)

		BeforeEach(func() {
			By("binds the port " + portstr)
			ln, err = net.Listen("tcp", ":"+portstr)
			Expect(err).ToNot(HaveOccurred())
		})

		mockEthService := &fabproxy_mocks.MockEthService{}
		proxy := fabproxy.NewFabProxy(mockEthService)

		It("exits instead of starting", func() {
			err := proxy.Start(port)
			Expect(err).To(HaveOccurred())
		})

		AfterEach(func() {
			By("releasing the port")
			err := ln.Close()
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
