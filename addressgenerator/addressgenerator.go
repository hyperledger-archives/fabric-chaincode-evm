/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package addressgenerator

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/protos/msp"
	"golang.org/x/crypto/sha3"
)

// IdentityToAddr takes in the bytes of serialized identity
// It will return a byte slice with length 20 (160 bits) that is the address associated
// with the public key
func IdentityToAddr(creator []byte) ([]byte, error) {
	si := &msp.SerializedIdentity{}
	if err := proto.Unmarshal(creator, si); err != nil {
		return nil, fmt.Errorf("failed to unmarshal serialized identity: %s", err)
	}

	bl, _ := pem.Decode(si.IdBytes)
	if bl == nil {
		return nil, fmt.Errorf("no pem data found")
	}

	cert, err := x509.ParseCertificate(bl.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %s", err)
	}

	pubkeyBytes, err := x509.MarshalPKIXPublicKey(cert.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal public key: %s", err)
	}

	// We want the last 160 bits of the sha3-256 sum as the address
	sum := sha3.Sum256(pubkeyBytes)
	return sum[12:], nil
}
