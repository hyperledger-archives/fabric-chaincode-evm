/*
Copyright NAVER Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package signing

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/btcsuite/btcd/btcec"
	"golang.org/x/crypto/sha3"
)

type SignedTx struct {
	Nonce     uint64   `json:"nonce"`
	Price     *big.Int `json:"gasPrice"`
	GasLimit  uint64   `json:"gas"`
	Recipient []byte   `json:"to"`
	Amount    *big.Int `json:"value"`
	Payload   []byte   `json:"input"`
	V         *big.Int `json:"v"`
	R         *big.Int `json:"r"`
	S         *big.Int `json:"s"`

	raw      []byte // raw transaction with signature
	unsigned []byte // raw transaction without signature
	msgIndex int    // byte offset where Nonce starts in raw
	sigIndex int    // byte offset where V starts in raw
	signer   []byte // caller address retrieved from signature
}

func (t *SignedTx) String() string {
	b, err := json.Marshal(t)
	if err != nil {
		return err.Error()
	}
	return string(b)
}

// CalleeAddress returns the hex encoded string of recipient address
func (t *SignedTx) CalleeAddress() string {
	if len(t.Recipient) == 0 {
		return "0x0000000000000000000000000000000000000000"
	}
	return "0x" + hex.EncodeToString(t.Recipient)
}

// CallerAddress returns the hex encoded string of caller address calcuated from the signature
func (t *SignedTx) CallerAddress() string {
	if len(t.signer) == 0 {
		return "0x0000000000000000000000000000000000000000"
	}
	return "0x" + hex.EncodeToString(t.signer)
}

// ChainID returns chain ID calcuated from V
// See https://github.com/ethereum/EIPs/blob/master/EIPS/eip-155.md
func (t *SignedTx) ChainID() *big.Int {
	v := t.V.Uint64()
	if v < 27 {
		return t.V
	}
	return new(big.Int).SetUint64((v - 35) / 2)
}

// prepareUnsigned encodes unsigned transaction by resetting v,r,s
func (t *SignedTx) prepareUnsigned() error {
	id := t.ChainID().Bytes()
	if len(id) != 1 || id[0] > 0x7F {
		return errors.New("private chain ID is not supported")
	}
	sigLen := 3
	b := make([]byte, t.sigIndex+sigLen)
	copy(b, t.raw)
	b[len(b)-3] = id[0] // v
	b[len(b)-2] = 0x80  // r
	b[len(b)-1] = 0x80  // s

	// rewrite list length
	msgLen := new(big.Int).SetUint64(uint64(len(b) - t.msgIndex))
	lenBytes := len(msgLen.Bytes())
	var msgIndex int
	if msgLen.Int64() <= 55 {
		b[0] = 0xC0 + byte(msgLen.Int64())
		msgIndex = 1
	} else {
		b[0] = 0xF7 + byte(lenBytes)
		copy(b[1:], msgLen.Bytes())
		msgIndex = 1 + lenBytes
	}

	if t.msgIndex == msgIndex {
		t.unsigned = b
		return t.retrieveSigner()
	}

	t.unsigned = make([]byte, len(b)-(t.msgIndex-msgIndex))
	copy(t.unsigned[:msgIndex], b[:msgIndex])
	copy(t.unsigned[msgIndex:], b[t.msgIndex:])
	return t.retrieveSigner()
}

func (t *SignedTx) hash(b []byte) []byte {
	h := sha3.NewLegacyKeccak256()
	h.Write(b)
	return h.Sum(nil)
}

func (t *SignedTx) retrieveSigner() error {
	r, s := t.R.Bytes(), t.S.Bytes()
	if len(r) == 0 || len(s) == 0 {
		fmt.Println("no signature included. skip retrieving signer")
		return nil
	}
	if len(r) > 32 || len(s) > 32 {
		return errors.New("invalid signature: r,s value is too big")
	}

	sig := make([]byte, 65)
	sig[0] = byte(t.V.Int64() - t.ChainID().Int64()*2 - 8)
	copy(sig[33-len(r):33], r)
	copy(sig[65-len(s):65], s)

	pkey, _, err := btcec.RecoverCompact(btcec.S256(), sig, t.hash(t.unsigned))
	if err != nil {
		return err
	}
	pub := (*btcec.PublicKey)(pkey).SerializeUncompressed()
	if len(pub) == 0 || pub[0] != 4 {
		return errors.New("invalid public key")
	}

	t.signer = make([]byte, 20)
	copy(t.signer, t.hash(pub[1:])[12:])
	return nil
}

// Decode decodes rlp encoded raw transaction
func (t *SignedTx) Decode(raw []byte) error {
	t.raw = raw
	br := &bytesReader{bytes.NewReader(raw), nil}
	ir := &intReader{bytesReader: br}
	xr := &bigIntReader{bytesReader: br}

	// read the length of rlp encoded list
	l := br.readLen()
	if br.err != nil {
		return errors.New("Decode error: cannot read rlp list length")
	}
	if l <= 55 {
		fmt.Println("Raw data is too small. May be unsigned transaction? size=", l)
	}

	// read transaction message
	t.msgIndex = len(raw) - br.Len()
	t.Nonce = ir.getInt()
	t.Price = xr.getInt()
	t.GasLimit = ir.getInt()
	t.Recipient = br.getBytes()
	t.Amount = xr.getInt()
	t.Payload = br.getBytes()

	// read signature
	t.sigIndex = len(raw) - br.Len()
	t.V = xr.getInt()
	t.R = xr.getInt()
	t.S = xr.getInt()
	if br.err != nil {
		return errors.New("Decode error: " + br.err.Error())
	}

	return t.prepareUnsigned()
}

type bytesReader struct {
	*bytes.Reader
	err error
}

func (r *bytesReader) readLen() uint64 {
	var b byte
	b, r.err = r.ReadByte()
	if r.err != nil {
		return 0
	}

	switch {
	case b < 0x80:
		r.UnreadByte()
		return 1
	case b < 0xB8:
		return uint64(b - 0x80)
	case b < 0xC0:
		buf := make([]byte, 8)
		len := b - 0xB7
		_, r.err = r.Read(buf[8-len:])
		if r.err != nil {
			return 0
		}
		return binary.BigEndian.Uint64(buf)
	case b < 0xF8:
		return uint64(b - 0xC0)
	default:
		buf := make([]byte, 8)
		len := b - 0xF7
		_, r.err = r.Read(buf[8-len:])
		if r.err != nil {
			return 0
		}
		return binary.BigEndian.Uint64(buf)
	}
}

func (r *bytesReader) getBytes() []byte {
	if r.err != nil {
		return nil
	}

	len := r.readLen()
	if r.err != nil {
		r.err = errors.New("cannot read data length: " + r.err.Error())
		return nil
	}

	buf := make([]byte, len)
	n, _ := r.Read(buf)
	if n < int(len) {
		r.err = errors.New("cannot read data fully")
		return nil
	}

	return buf
}

type intReader struct {
	*bytesReader
}

func (r *intReader) getInt() uint64 {
	b := r.getBytes()
	if r.err != nil {
		return 0
	}
	if len(b) > 8 {
		r.err = errors.New("int64 size too big")
		return 0
	}

	buf := make([]byte, 8)
	copy(buf[8-len(b):], b)
	return binary.BigEndian.Uint64(buf)
}

type bigIntReader struct {
	*bytesReader
}

func (r *bigIntReader) getInt() *big.Int {
	b := r.getBytes()
	if r.err != nil {
		return nil
	}

	val := new(big.Int)
	val.SetBytes(b)
	return val
}
