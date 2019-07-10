/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package addressgenerator_test

import (
	"encoding/hex"

	"github.com/gogo/protobuf/proto"
	"github.com/hyperledger/fabric-chaincode-evm/addressgenerator"
	"github.com/hyperledger/fabric/protos/msp"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("IdentityToAddr", func() {
	var (
		cert = `-----BEGIN CERTIFICATE-----
MIIB/zCCAaWgAwIBAgIRAKaex32sim4PQR6kDPEPVnwwCgYIKoZIzj0EAwIwaTEL
MAkGA1UEBhMCVVMxEzARBgNVBAgTCkNhbGlmb3JuaWExFjAUBgNVBAcTDVNhbiBG
cmFuY2lzY28xFDASBgNVBAoTC2V4YW1wbGUuY29tMRcwFQYDVQQDEw5jYS5leGFt
cGxlLmNvbTAeFw0xNzA3MjYwNDM1MDJaFw0yNzA3MjQwNDM1MDJaMEoxCzAJBgNV
BAYTAlVTMRMwEQYDVQQIEwpDYWxpZm9ybmlhMRYwFAYDVQQHEw1TYW4gRnJhbmNp
c2NvMQ4wDAYDVQQDEwVwZWVyMDBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABPzs
BSdIIB0GrKmKWn0N8mMfxWs2s1D6K+xvTvVJ3wUj3znNBxj+k2j2tpPuJUExt61s
KbpP3GF9/crEahpXXRajTTBLMA4GA1UdDwEB/wQEAwIHgDAMBgNVHRMBAf8EAjAA
MCsGA1UdIwQkMCKAIEvLfQX685pz+rh2q5yCA7e0a/a5IGDuJVHRWfp++HThMAoG
CCqGSM49BAMCA0gAMEUCIH5H9W3tsCrti6tsN9UfY1eeTKtExf/abXhfqfVeRChk
AiEA0GxTPOXVHo0gJpMbHc9B73TL5ZfDhujoDyjb8DToWPQ=
-----END CERTIFICATE-----`
		creator []byte
	)
	BeforeEach(func() {
		var err error
		creator, err = proto.Marshal(&msp.SerializedIdentity{IdBytes: []byte(cert)})
		Expect(err).ToNot(HaveOccurred())
	})

	It("returns a 160 bit address from a public key", func() {
		address, err := addressgenerator.IdentityToAddr([]byte(creator))
		Expect(err).ToNot(HaveOccurred())

		Expect(hex.EncodeToString(address)).To(Equal("b3778bcee2b9c349702e5832928730d2aed0ac07"),
			"address generation has changed. Please update test.")
	})
})
