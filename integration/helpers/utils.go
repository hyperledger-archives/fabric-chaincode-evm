/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package helpers

import (
	"fmt"
	"net"

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

type JsonRPCLogArrayResponse struct {
	JsonRPC string       `json:"jsonrpc"`
	ID      int          `json:"id"`
	Result  []types.Log  `json:"result,omitempty"`
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

func SimpleSoloNetwork() *nwo.Config {
	return &nwo.Config{
		Organizations: []*nwo.Organization{{
			Name:          "OrdererOrg",
			MSPID:         "OrdererMSP",
			Domain:        "example.com",
			EnableNodeOUs: false,
			Users:         0,
			CA:            &nwo.CA{Hostname: "ca"},
		}, {
			Name:          "Org1",
			MSPID:         "Org1MSP",
			Domain:        "org1.example.com",
			EnableNodeOUs: true,
			Users:         2,
			CA:            &nwo.CA{Hostname: "ca"},
		}},
		Consortiums: []*nwo.Consortium{{
			Name: "SampleConsortium",
			Organizations: []string{
				"Org1",
			},
		}},
		Consensus: &nwo.Consensus{
			Type: "solo",
		},
		SystemChannel: &nwo.SystemChannel{
			Name:    "systemchannel",
			Profile: "OneOrgOrdererGenesis",
		},
		Orderers: []*nwo.Orderer{
			{Name: "orderer", Organization: "OrdererOrg"},
		},
		Channels: []*nwo.Channel{
			{Name: "testchannel", Profile: "OneOrgChannel"},
		},
		Peers: []*nwo.Peer{{
			Name:         "peer0",
			Organization: "Org1",
			Channels: []*nwo.PeerChannel{
				{Name: "testchannel", Anchor: true},
			},
		}},
		Profiles: []*nwo.Profile{{
			Name:     "OneOrgOrdererGenesis",
			Orderers: []string{"orderer"},
		}, {
			Name:          "OneOrgChannel",
			Consortium:    "SampleConsortium",
			Organizations: []string{"Org1"},
		}},
	}
}

func WaitForFab3(port uint16) {
	Eventually(func() error {
		conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
		if err == nil {
			defer conn.Close()
		}
		return err
	}).Should(Succeed())
}
