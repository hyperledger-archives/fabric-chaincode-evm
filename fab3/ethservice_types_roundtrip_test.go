/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fab3_test

import (
	"encoding/json"
	"reflect"

	"github.com/hyperledger/fabric-chaincode-evm/fab3"

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

		It("for TxReceipt with the proper cases", func() {
			fieldNames := []string{"transactionHash", "transactionIndex",
				"blockHash", "blockNumber", "contractAddress", "gasUsed",
				"cumulativeGasUsed", "to", "status", "logs"}
			assertTypeMarshalsJSONFields(fieldNames, fab3.TxReceipt{})
		})
		It("for Log subobjects in TxReceipt with the proper cases", func() {
			fieldNames := []string{"address", "topics", "data", "blockNumber",
				"transactionHash", "transactionIndex", "blockHash", "logIndex"}
			assertTypeMarshalsJSONFields(fieldNames, fab3.Log{Data: "somedata"})
		})
		It("for Transaction with the proper cases", func() {
			fieldNames := []string{"blockHash", "blockNumber", "to", "input", "transactionIndex", "hash"}
			assertTypeMarshalsJSONFields(fieldNames, fab3.Transaction{})
		})
		It("for Block with the proper cases", func() {
			fieldNames := []string{"number", "hash", "parentHash", "transactions"}
			assertTypeMarshalsJSONFields(fieldNames, fab3.Block{})
		})
	})
})
