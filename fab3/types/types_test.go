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

// slack input validation refers to having a 0x as prefix for input values

var _ = Describe("Types JSON Marshaling and Unmarshaling", func() {
	Context("GetLogsArgs JSON", func() {
		type valid struct {
			from      string
			to        string
			addresses []string
			topics    [][]string
		}
		DescribeTable("Valid JSON",
			func(bytes []byte, check valid) {
				var target GetLogsArgs
				err := json.Unmarshal(bytes, &target)
				Expect(err).ToNot(HaveOccurred())

				Expect(target.FromBlock).To(Equal(check.from))
				Expect(target.ToBlock).To(Equal(check.to))
				Expect(len(target.Address)).To(Equal(len(check.addresses)))
				for _, address := range check.addresses {
					Expect(target.Address).To(ContainElement(address)) // order not important
					fmt.Fprintln(GinkgoWriter, target.Address, address)
				}
				fmt.Fprintln(GinkgoWriter, target.Topics, check.topics)
				Expect(len(target.Topics)).To(Equal(len(check.topics)))
				for i, topicFilters := range check.topics {
					Expect(len(target.Topics[i])).To(Equal(len(topicFilters)))
					for _, topic := range topicFilters {
						fmt.Fprintln(GinkgoWriter, target.Topics[i], topic)
						Expect(target.Topics[i]).To(ContainElement(topic))
					}
				}
			},

			Entry("empty json",
				[]byte(`{}`),
				valid{"", "", nil, nil}),
			// block references
			Entry("block refs",
				[]byte(`{"fromBlock":"0x0","toBlock":"0x1"}`),
				valid{"0", "1", nil, nil}),
			Entry("block hash",
				[]byte(`{"blockHash":"0x0"}`),
				valid{"", "", nil, nil}),
			Entry("slack block refs",
				[]byte(`{"fromBlock":"0","toBlock":"1"}`),
				valid{"0", "1", nil, nil}),
			Entry("textual block refs",
				[]byte(`{"fromBlock":"earliest","toBlock":"latest"}`),
				valid{"earliest", "latest", nil, nil}),
			Entry("from-to order not checked, from is greater than to",
				[]byte(`{"fromBlock":"0x1","toBlock":"0x0"}`),
				valid{"1", "0", nil, nil}),
			Entry("garbage block refs not validated",
				[]byte(`{"toBlock":"there","fromBlock":"hello"}`),
				valid{"hello", "there", nil, nil}),
			// addresses
			Entry("single address",
				[]byte(`{"address":"0x3832333733343538313634383230393437383931"}`),
				valid{"", "", []string{"3832333733343538313634383230393437383931"}, nil}),
			Entry("single address in array",
				[]byte(`{"address":["0x3832333733343538313634383230393437383931"]}`),
				valid{"", "", []string{"3832333733343538313634383230393437383931"}, nil}),
			Entry("multiple address in array",
				[]byte(`{"address":["0x3832333733343538313634383230393437383931","0x3832333733343538313634383230393437383932"]}`),
				valid{"", "", []string{"3832333733343538313634383230393437383931", "3832333733343538313634383230393437383932"}, nil}),
			Entry("slack validation of multiple address in array",
				[]byte(`{"address":["3832333733343538313634383230393437383931","3832333733343538313634383230393437383932"]}`),
				valid{"", "", []string{"3832333733343538313634383230393437383931", "3832333733343538313634383230393437383932"}, nil}),
			// all addresses are lowered cased
			Entry("mixed case single addresses",
				[]byte(`{"address":"0x38323337333435383136343832303934373839aA"}`),
				valid{"", "", []string{"38323337333435383136343832303934373839aa"}, nil}),
			Entry("mixed cases single address in array",
				[]byte(`{"address":["0x38323337333435383136343832303934373839bB"]}`),
				valid{"", "", []string{"38323337333435383136343832303934373839bb"}, nil}),
			Entry("mixed cases multiple address in array",
				[]byte(`{"address":["0x38323337333435383136343832303934373839aA","0x38323337333435383136343832303934373839bB"]}`),
				valid{"", "", []string{"38323337333435383136343832303934373839aa", "38323337333435383136343832303934373839bb"}, nil}),
			// topics
			Entry("any topic",
				[]byte(`{"topics":[]}`),
				valid{"", "", nil, [][]string{}}),
			Entry("single null topic",
				[]byte(`{"topics":[null]}`),
				valid{"", "", nil, [][]string{{""}}}),
			Entry("multiple null topic",
				[]byte(`{"topics":[null,null]}`),
				valid{"", "", nil, [][]string{{""}, {""}}}),
			Entry("single separate nulls topic",
				[]byte(`{"topics":[[null],[null]]}`),
				valid{"", "", nil, [][]string{{""}, {""}}}),
			Entry("multiple nulls in arrays topic",
				[]byte(`{"topics":[[null,null],[null,null]]}`),
				valid{"", "", nil, [][]string{{"", ""}, {"", ""}}}),
			Entry("single topic",
				[]byte(`{"topics":["0x1234567890123456789012345678901234567890123456789012345678901234"]}`),
				valid{"", "", nil, [][]string{{"1234567890123456789012345678901234567890123456789012345678901234"}}}),
			Entry("single topic, slack input",
				[]byte(`{"topics":["1234567890123456789012345678901234567890123456789012345678901234"]}`),
				valid{"", "", nil, [][]string{{"1234567890123456789012345678901234567890123456789012345678901234"}}}),
			Entry("single topic with or'd options",
				[]byte(`{"topics":[["0x1234567890123456789012345678901234567890123456789012345678901234","0x1234567890123456789012345678901234567890123456789012345678901235"]]}`),
				valid{"", "", nil, [][]string{{"1234567890123456789012345678901234567890123456789012345678901234", "1234567890123456789012345678901234567890123456789012345678901235"}}}),
			Entry("single topic with or'd options, slack input",
				[]byte(`{"topics":[["1234567890123456789012345678901234567890123456789012345678901234","1234567890123456789012345678901234567890123456789012345678901235"]]}`),
				valid{"", "", nil, [][]string{{"1234567890123456789012345678901234567890123456789012345678901234", "1234567890123456789012345678901234567890123456789012345678901235"}}}),

			Entry("single topic in multi array",
				[]byte(`{"topics":[["0x1234567890123456789012345678901234567890123456789012345678901234"]]}`),
				valid{"", "", nil, [][]string{{"1234567890123456789012345678901234567890123456789012345678901234"}}}),
			Entry("single topic in multi array, slack input",
				[]byte(`{"topics":[["1234567890123456789012345678901234567890123456789012345678901234"]]}`),
				valid{"", "", nil, [][]string{{"1234567890123456789012345678901234567890123456789012345678901234"}}}),
			Entry("multi topic",
				[]byte(`{"topics":["0x1234567890123456789012345678901234567890123456789012345678901234", "0x1234567890123456789012345678901234567890123456789012345678901234"]}`),
				valid{"", "", nil, [][]string{{"1234567890123456789012345678901234567890123456789012345678901234"}, {"1234567890123456789012345678901234567890123456789012345678901234"}}}),
			Entry("multi topic, slack input",
				[]byte(`{"topics":["1234567890123456789012345678901234567890123456789012345678901234", "1234567890123456789012345678901234567890123456789012345678901234"]}`),
				valid{"", "", nil, [][]string{{"1234567890123456789012345678901234567890123456789012345678901234"}, {"1234567890123456789012345678901234567890123456789012345678901234"}}}),

			Entry("multi topic with or'd options, some slack mixed in",
				[]byte(`{"topics":[["0x1234567890123456789012345678901234567890123456789012345678901234","1234567890123456789012345678901234567890123456789012345678901235"], ["1234567890123456789012345678901234567890123456789012345678901234","0x1234567890123456789012345678901234567890123456789012345678901235"]]}`),
				valid{"", "", nil, [][]string{{"1234567890123456789012345678901234567890123456789012345678901234", "1234567890123456789012345678901234567890123456789012345678901235"}, {"1234567890123456789012345678901234567890123456789012345678901234", "1234567890123456789012345678901234567890123456789012345678901235"}}}),
			// everything
			Entry("all fields, no matter the order",
				[]byte(`{"address":["0x3832333733343538313634383230393437383931","0x3832333733343538313634383230393437383932"],"fromBlock":"0x1","toBlock":"earliest"}`),
				valid{"1", "earliest", []string{"3832333733343538313634383230393437383931", "3832333733343538313634383230393437383931"}, nil}),
		)

		DescribeTable("Invalid input JSON",
			func(bytes []byte) {
				var target GetLogsArgs
				err := json.Unmarshal(bytes, &target)
				Expect(err).To(HaveOccurred())
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
			Entry("must be slice",
				[]byte(`{"topics":"must be slice"}`)),
			Entry("bad single topic",
				[]byte(`{"topics":["not a topic"]}`)),
			Entry("bad array topic",
				[]byte(`{"topics":[["not a topic"]]}`)),
			Entry("non string array topic",
				[]byte(`{"topics":[[1234]]}`)),
			Entry("incorrect topics format",
				[]byte(`{"topics":[{"trash":"object"}]}`)),
			Entry("mixed block specifier and blockhash",
				[]byte(`{"fromBlock":"0x1", "blockHash":"0x47"}`)),
		)

	})

	DescribeTable("Transaction MarshalJSON",
		func(tx Transaction, bytes []byte) {
			marshalledData, err := tx.MarshalJSON()
			Expect(err).ToNot(HaveOccurred())

			var target Transaction
			err = json.Unmarshal(marshalledData, &target)
			Expect(err).ToNot(HaveOccurred())

			var expectedTx Transaction
			err = json.Unmarshal(bytes, &expectedTx)
			Expect(err).ToNot(HaveOccurred())

			Expect(target).To(Equal(expectedTx))
		},

		Entry("empty transaction object",
			Transaction{},
			[]byte(`{"blockHash":"", "blockNumber":"", "to":"", "input":"","transactionIndex":"", "hash":"", "gasPrice":"0x0", "value":"0x0"}`)),

		Entry("non-empty transaction object",
			Transaction{BlockHash: "0x1234567"},
			[]byte(`{"blockHash":"0x1234567", "blockNumber":"", "to":"", "input":"","transactionIndex":"", "hash":"", "gasPrice":"0x0", "value":"0x0"}`)),

		Entry("non-empty gasPrice and value fields",
			Transaction{BlockHash: "0x1234567", GasPrice: "some-value", Value: "some-price"},
			[]byte(`{"blockHash":"0x1234567", "blockNumber":"", "to":"", "input":"","transactionIndex":"", "hash":"", "gasPrice":"0x0", "value":"0x0"}`)),
	)

	DescribeTable("TxReceipt MarshalJSON",
		func(txReceipt TxReceipt, bytes []byte) {
			marshalledData, err := txReceipt.MarshalJSON()
			Expect(err).ToNot(HaveOccurred())

			var target TxReceipt
			err = json.Unmarshal(marshalledData, &target)
			Expect(err).ToNot(HaveOccurred())

			var expectedReceipt TxReceipt
			err = json.Unmarshal(bytes, &expectedReceipt)
			Expect(err).ToNot(HaveOccurred())

			Expect(target).To(Equal(expectedReceipt))
		},

		Entry("empty receipt object",
			TxReceipt{},
			[]byte(`{"transactionHash":"","transactionIndex":"","blockHash":"","blockNumber":"","gasUsed":0,"cumulativeGasUsed":0,"to":"","logs":[],"status":"","from":""}`)),

		Entry("non-empty receipt object",
			TxReceipt{Status: "0x1", TransactionHash: "0x9fc76417374aa880d4449a1f7f31ec597f00b1f6f3dd2d66f4c9c6c445836d8b", CumulativeGasUsed: 314159, GasUsed: 30234},
			[]byte(`{"transactionHash":"0x9fc76417374aa880d4449a1f7f31ec597f00b1f6f3dd2d66f4c9c6c445836d8b","transactionIndex":"","blockHash":"","blockNumber":"","gasUsed":30234,"cumulativeGasUsed":314159,"to":"","logs":[],"status":"0x1","from":""}`)),

		Entry("non-empty logs field",
			TxReceipt{Logs: make([]Log, 1), Status: "0x1", TransactionHash: "0x9fc76417374aa880d4449a1f7f31ec597f00b1f6f3dd2d66f4c9c6c445836d8b", CumulativeGasUsed: 314159, GasUsed: 30234},
			[]byte(`{"transactionHash":"0x9fc76417374aa880d4449a1f7f31ec597f00b1f6f3dd2d66f4c9c6c445836d8b","transactionIndex":"","blockHash":"","blockNumber":"","gasUsed":30234,"cumulativeGasUsed":314159,"to":"","contractAddress":"","gasUsed":30234,"cumulativeGasUsed":314159,"to":"","logs":[{"address":"","topics":null,"blockNumber":"","transactionHash":"","transactionIndex":"","blockHash":"","logIndex":""}],"status":"0x1","from":""}`)),
	)

	DescribeTable("Block MarshalJSON",
		func(blk Block, bytes []byte) {
			By("copying the original struct")
			originalBlock := blk

			marshalledData, err := json.Marshal(&blk)
			Expect(err).ToNot(HaveOccurred())

			var target Block
			err = json.Unmarshal(marshalledData, &target)
			Expect(err).ToNot(HaveOccurred())

			var expectedBlk Block
			err = json.Unmarshal(bytes, &expectedBlk)
			Expect(err).ToNot(HaveOccurred())

			Expect(target).To(Equal(expectedBlk))

			Expect(blk).To(Equal(originalBlock), "the struct should not have been modified")
		},

		Entry("block with no transaction",
			Block{},
			[]byte(`{"number":"", "parentHash":"", "hash":"", "gasLimit":"0x0"}`)),

		Entry("block with full transactions",
			Block{
				Transactions: []interface{}{Transaction{}}},
			[]byte(`{"number":"", "parentHash":"", "hash":"", "gasLimit":"0x0", "transactions":[{"blockHash":"", "blockNumber":"", "to":"", "from":"", "input":"","transactionIndex":"", "hash":"", "gasPrice":"0x0", "value":"0x0"}]}`)),

		Entry("block with not full transactions",
			Block{
				Transactions: []interface{}{"0x12345678"}},
			[]byte(`{"number":"", "parentHash":"", "hash":"", "gasLimit":"0x0", "transactions":["0x12345678"]}`)),
	)
})
