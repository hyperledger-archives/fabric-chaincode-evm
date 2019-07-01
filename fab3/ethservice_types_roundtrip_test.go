/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fab3_test

import (
	"encoding/json"
	"reflect"

	"github.com/hyperledger/fabric-chaincode-evm/fab3/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ethereum json rpc struct fields", func() {
	Context("match case exactly, in camelCase starting with a lowercase letter", func() {
		assertTypeMarshalsJSONFields := func(fieldNames []string, itype interface{}) {
			b, err := json.Marshal(itype)
			Expect(err).ToNot(HaveOccurred())
			m := make(map[string]interface{})
			err = json.Unmarshal(b, &m)
			Expect(err).ToNot(HaveOccurred())
			Expect(m).To(HaveLen(len(fieldNames)), "did you add a field to this type? %q", reflect.TypeOf(itype))
			for _, fieldName := range fieldNames {
				_, ok := m[fieldName]
				Expect(ok).To(BeTrue(), "couldn't find an expected field %q in this type", fieldName)
			}
		}
		Context("TxReceipt CamelCase Keys", func() {

			It("includes `contractAddress` when the contract address is not empty", func() {
				fieldNames := []string{"transactionHash", "transactionIndex",
					"blockHash", "blockNumber", "contractAddress", "gasUsed",
					"cumulativeGasUsed", "to", "status", "logs"}
				assertTypeMarshalsJSONFields(fieldNames, types.TxReceipt{ContractAddress: "some-address"})
			})

			It("does not include `contractAddress` as a key when the contract address is empty", func() {
				fieldNames := []string{"transactionHash", "transactionIndex",
					"blockHash", "blockNumber", "gasUsed",
					"cumulativeGasUsed", "to", "status", "logs"}
				assertTypeMarshalsJSONFields(fieldNames, types.TxReceipt{})
			})

		})

		It("for Log subobjects in TxReceipt with the proper cases", func() {
			fieldNames := []string{"address", "topics", "data", "blockNumber",
				"transactionHash", "transactionIndex", "blockHash", "logIndex"}
			assertTypeMarshalsJSONFields(fieldNames, types.Log{Data: "somedata"})
		})

		It("for Transaction with the proper cases", func() {
			fieldNames := []string{"blockHash", "blockNumber", "to", "input", "transactionIndex", "hash", "gasPrice", "value"}
			assertTypeMarshalsJSONFields(fieldNames, types.Transaction{})
		})

		It("for Block with the proper cases", func() {
			fieldNames := []string{"number", "hash", "parentHash", "transactions"}
			assertTypeMarshalsJSONFields(fieldNames, types.Block{})
		})
	})
})
