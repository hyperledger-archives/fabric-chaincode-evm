/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package helpers

import (
	"github.com/hyperledger/fabric/integration/nwo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

type JsonRPCRequest struct {
	JsonRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type JsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data"`
}

type JsonRPCResponse struct {
	JsonRPC string       `json:"jsonrpc"`
	ID      int          `json:"id"`
	Result  string       `json:"result,omitempty"`
	Error   JsonRPCError `json:"error,omitempty"`
}

type JsonRPCArrayResponse struct {
	JsonRPC string       `json:"jsonrpc"`
	ID      int          `json:"id"`
	Result  []string     `json:"result,omitempty"`
	Error   JsonRPCError `json:"error,omitempty"`
}

type JsonRPCTxReceipt struct {
	JsonRPC string       `json:"jsonrpc"`
	ID      int          `json:"id"`
	Result  TxReceipt    `json:"result"`
	Error   JsonRPCError `json:"error,omitempty"`
}

type TxReceipt struct {
	TransactionHash  string `json:"transactionHash"`
	TransactionIndex string `json:"transactionIndex"`
	BlockNumber      string `json:"blockNumber"`
	BlockHash        string `json:"blockHash"`
	ContractAddress  string `json:"contractAddress,omitempty"`
}

type MessageParams struct {
	To   string `json:"to"`
	From string `json:"from,omitempty"`
	Data string `json:"data,omitempty"`
}

type ChaincodeQueryWithHex struct {
	ChannelID string
	Name      string
	Ctor      string
}

func (c ChaincodeQueryWithHex) SessionName() string {
	return "peer-chaincode-query"
}

func (c ChaincodeQueryWithHex) Args() []string {
	return []string{
		"chaincode", "query",
		"--channelID", c.ChannelID,
		"--name", c.Name,
		"--ctor", c.Ctor,
		"--hex",
	}
}

func Build() *nwo.Components {
	components := &nwo.Components{}
	components.Build()

	proxyBinPath, err := gexec.Build("github.com/hyperledger/fabric-chaincode-evm/fabproxy/cmd")
	Expect(err).ToNot(HaveOccurred())
	components.Paths["fabproxy"] = proxyBinPath

	return components
}
