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
