/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabproxy_test

import (
	"net/http"

	"github.com/hyperledger/fabric-chaincode-evm/fabproxy"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("NetService", func() {
	Describe("Version", func() {
		It("returns the fabric evm network id", func() {
			netservice := fabproxy.NetService{}
			var reply string
			err := netservice.Version(&http.Request{}, nil, &reply)
			Expect(err).ToNot(HaveOccurred())
			Expect(reply).To(Equal(fabproxy.NetworkID))
		})
	})
})
