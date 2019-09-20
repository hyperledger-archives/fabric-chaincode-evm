/*
Copyright NAVER Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package signing

import (
	"encoding/hex"
	"math/big"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

/* rawTx is RLP encoding of this:
  {
	"nonce": 324,
	"gasPrice": 1000000000,
	"gasLimit": 21000,
	"to": "0xb78777860637d56543da23312c7865024833f7d1",
	"value": 100000000000000000,
	"from": "0x17da6a8b86578cec4525945a355e8384025fa5af",
	"data": "",
	"v": "2b",
	"r": "e2539a5d9f056d7095bd19d6b77b850910eeafb71534ebd45159915fab202e91",
	"s": "07484420f3968697974413fc55d1142dc76285d30b1b9231ccb71ed1e720faae"
  }
*/
// rawTx is picked from https://flightwallet.org/decode-eth-tx
var rawTx = `f86d820144843b9aca0082520894b78777860637d56543da23312c7865024833f7d188016345785d8a0000802ba0e2539a5d9f056d7095bd19d6b77b850910eeafb71534ebd45159915fab202e91a007484420f3968697974413fc55d1142dc76285d30b1b9231ccb71ed1e720faae`
var brokenTx = `f86d820144843b9aca0082520894b78777860637d56543da23312c`

// eip155Tx is picked from https://github.com/ethereum/EIPs/blob/master/EIPS/eip-155.md
var eip155Tx = `f86c098504a817c800825208943535353535353535353535353535353535353535880de0b6b3a76400008025a028ef61340bd939bc2195fe537567866003e1a15d3c71ff63e1590620aa636276a067cbe9d8997f761aecb703304b3800ccf555c9f3dc64214b297fb1966a3b6d83`

var _ = Describe("SignedTx", func() {
	It("decodes signed raw transaction and prepares unsigned raw transaction", func() {
		tx := &SignedTx{}
		b, err := hex.DecodeString(eip155Tx)
		Expect(err).NotTo(HaveOccurred())

		By("Check it decode correctly")

		err = tx.Decode(b)
		Expect(err).NotTo(HaveOccurred())
		Expect(tx.Nonce).To(Equal(uint64(9)))
		price := big.NewInt(20 * 1000000000)
		Expect(tx.Price.Cmp(price)).To(Equal(0))
		Expect(tx.GasLimit).To(Equal(uint64(21000)))
		Expect(tx.CalleeAddress()).To(Equal("0x3535353535353535353535353535353535353535"))
		value := big.NewInt(1000000000000000000)
		Expect(tx.Amount.Cmp(value)).To(Equal(0))
		Expect(tx.Payload).To(Equal([]byte{}))
		v := big.NewInt(37)
		Expect(tx.V.Cmp(v)).To(Equal(0))
		r := big.NewInt(0)
		r.SetString("18515461264373351373200002665853028612451056578545711640558177340181847433846", 10)
		Expect(tx.R.Cmp(r)).To(Equal(0))
		s := big.NewInt(0)
		s.SetString("46948507304638947509940763649030358759909902576025900602547168820602576006531", 10)
		Expect(tx.S.Cmp(s)).To(Equal(0))
		Expect(tx.CallerAddress()).To(Equal("0x9d8a62f656a8d1615c1294fd71e9cfb3e4855a4f"))

		By("Check it encodes unsigned raw transaction correctly")

		Expect(tx.unsigned).NotTo(Equal(nil))
		unsigned := &SignedTx{}
		err = unsigned.Decode(tx.unsigned)
		Expect(err).NotTo(HaveOccurred())
		Expect(unsigned.Nonce).To(Equal(uint64(9)))
		Expect(unsigned.Price.Cmp(price)).To(Equal(0))
		Expect(unsigned.GasLimit).To(Equal(uint64(21000)))
		Expect(unsigned.CalleeAddress()).To(Equal("0x3535353535353535353535353535353535353535"))
		Expect(tx.Amount.Cmp(value)).To(Equal(0))
		Expect(tx.Payload).To(Equal([]byte{}))
		Expect(unsigned.V.Cmp(tx.ChainID())).To(Equal(0))
		Expect(unsigned.R.Cmp(big.NewInt(0))).To(Equal(0))
		Expect(unsigned.S.Cmp(big.NewInt(0))).To(Equal(0))

		Expect(unsigned.raw).To(Equal(unsigned.unsigned))
	})

	It("decodes another signed raw transaction", func() {
		tx := &SignedTx{}
		b, err := hex.DecodeString(rawTx)
		Expect(err).NotTo(HaveOccurred())

		err = tx.Decode(b)
		Expect(err).NotTo(HaveOccurred())
		Expect(tx.Nonce).To(Equal(uint64(324)))
		price := big.NewInt(1000000000)
		Expect(tx.Price.Cmp(price)).To(Equal(0))
		Expect(tx.GasLimit).To(Equal(uint64(21000)))
		Expect(tx.CalleeAddress()).To(Equal("0xb78777860637d56543da23312c7865024833f7d1"))
		value := big.NewInt(100000000000000000)
		Expect(tx.Amount.Cmp(value)).To(Equal(0))
		Expect(tx.Payload).To(Equal([]byte{}))
		v := big.NewInt(0x2b)
		Expect(tx.V.Cmp(v)).To(Equal(0))
		r := big.NewInt(0)
		r.SetString("e2539a5d9f056d7095bd19d6b77b850910eeafb71534ebd45159915fab202e91", 16)
		Expect(tx.R.Cmp(r)).To(Equal(0))
		s := big.NewInt(0)
		s.SetString("07484420f3968697974413fc55d1142dc76285d30b1b9231ccb71ed1e720faae", 16)
		Expect(tx.S.Cmp(s)).To(Equal(0))
		Expect(tx.CallerAddress()).To(Equal("0x17da6a8b86578cec4525945a355e8384025fa5af"))
	})

	It("returns error on decoding broken raw transaction", func() {
		tx := &SignedTx{}
		b, err := hex.DecodeString(brokenTx)
		Expect(err).NotTo(HaveOccurred())

		err = tx.Decode(b)
		Expect(err).To(HaveOccurred())
	})
})
