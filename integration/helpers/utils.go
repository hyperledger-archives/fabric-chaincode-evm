/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package helpers

import (
	"github.com/hyperledger/fabric/integration/nwo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"github.com/hyperledger/fabric-chaincode-evm/fab3/types"
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
	JsonRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  types.TxReceipt `json:"result"`
	Error   JsonRPCError    `json:"error,omitempty"`
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

	proxyBinPath, err := gexec.Build("github.com/hyperledger/fabric-chaincode-evm/fab3/cmd")
	Expect(err).ToNot(HaveOccurred())
	components.Paths["fab3"] = proxyBinPath

	return components
}
