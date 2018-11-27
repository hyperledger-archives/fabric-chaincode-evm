/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabproxy_test

import (
	"encoding/json"
	"reflect"

	"github.com/hyperledger/fabric-chaincode-evm/fabproxy"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ethereum json rpc struct fields", func() {
	By("that match case exactly, in camelCase starting with a lowercase letter")
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
		assertTypeMarshalsJSONFields(fieldNames, fabproxy.TxReceipt{})
	})
	It("for Log subobjects in TxReceipt with the proper cases", func() {
		fieldNames := []string{"address", "topics", "data", "blockNumber",
			"transactionHash", "transactionIndex", "blockHash", "logIndex"}
		assertTypeMarshalsJSONFields(fieldNames, fabproxy.Log{})
	})
	It("for Transaction with the proper cases", func() {
		fieldNames := []string{"blockHash", "blockNumber", "to", "input", "transactionIndex", "hash"}
		assertTypeMarshalsJSONFields(fieldNames, fabproxy.Transaction{})
	})
	It("for Block with the proper cases", func() {
		fieldNames := []string{"number", "hash", "parentHash", "transactions"}
		assertTypeMarshalsJSONFields(fieldNames, fabproxy.Block{})
	})
})
