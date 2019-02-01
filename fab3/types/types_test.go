/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package types

import (
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo/extensions/table"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("GetLogsArgs UnmarshalJSON", func() {
	DescribeTable("Valid JSON",
		func(bytes []byte, from, to string, addresses []string) {
			var target GetLogsArgs
			err := json.Unmarshal(bytes, &target)
			Expect(err).ShouldNot(HaveOccurred())

			Expect(target.FromBlock).Should(Equal(from))
			Expect(target.ToBlock).Should(Equal(to))
			for _, address := range addresses {
				Expect(target.Address).Should(ContainElement(address))
			}
			fmt.Sprintln(GinkgoWriter, target.Address, addresses)
		},

		Entry("empty json",
			[]byte(`{}`),
			"", "", []string{}),
		Entry("block refs",
			[]byte(`{"fromBlock":"0x0","toBlock":"0x1"}`),
			"0", "1", []string{}),
		Entry("slack block refs",
			[]byte(`{"fromBlock":"0","toBlock":"1"}`),
			"0", "1", []string{}),
		Entry("textual block refs",
			[]byte(`{"fromBlock":"earliest","toBlock":"latest"}`),
			"earliest", "latest", []string{}),
		Entry("from-to order not checked, from is greater than to",
			[]byte(`{"fromBlock":"0x1","toBlock":"0x0"}`),
			"1", "0", []string{}),
		Entry("garbage block refs not validated",
			[]byte(`{"toBlock":"there","fromBlock":"hello"}`),
			"hello", "there", []string{}),
		Entry("single address",
			[]byte(`{"address":"0x3832333733343538313634383230393437383931"}`),
			"", "", []string{"3832333733343538313634383230393437383931"}),
		Entry("single address in array",
			[]byte(`{"address":["0x3832333733343538313634383230393437383931"]}`),
			"", "", []string{"3832333733343538313634383230393437383931"}),
		Entry("multiple address in array",
			[]byte(`{"address":["0x3832333733343538313634383230393437383931","0x3832333733343538313634383230393437383932"]}`),
			"", "", []string{"3832333733343538313634383230393437383931", "3832333733343538313634383230393437383932"}),
		Entry("slack validation of multiple address in array",
			[]byte(`{"address":["3832333733343538313634383230393437383931","3832333733343538313634383230393437383932"]}`),
			"", "", []string{"3832333733343538313634383230393437383931", "3832333733343538313634383230393437383932"}),
		Entry("all fields, no matter the order",
			[]byte(`{"address":["0x3832333733343538313634383230393437383931","0x3832333733343538313634383230393437383932"],"fromBlock":"0x1","toBlock":"earliest"}`),
			"1", "earliest", []string{"3832333733343538313634383230393437383931", "3832333733343538313634383230393437383931"}),
	)

	DescribeTable("Invalid input JSON",
		func(bytes []byte) {
			var target GetLogsArgs
			err := json.Unmarshal(bytes, &target)
			Expect(err).Should(HaveOccurred())
		},
		Entry("wrong type",
			[]byte(`{"fromBlock":4}`)),
		Entry("not json",
			[]byte(`adrsen@(*P@*#J`)),
		Entry("bad formatted single address",
			[]byte(`{"address":"343538313634383230393437383932"}`)),
		Entry("bad formatted addresses",
			[]byte(`{"address":["0x38323","343538313634383230393437383932"]}`)),
		Entry("mixed good and bad addresses",
			[]byte(`{"address":["0x3832333733343538313634383230393437383931","634383230393437383932"]}`)),
		Entry("empty single address",
			[]byte(`{"address":""}`)),
		Entry("empty address in array",
			[]byte(`{"address":["0x3832333733343538313634383230393437383931",""]}`)),
		Entry("bad formatted addresses",
			[]byte(`{"address":1233456}`)),
	)
})
