/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package statemanager_test

import (
	"encoding/hex"
	"errors"

	"github.com/hyperledger/burrow/acm"
	"github.com/hyperledger/burrow/binary"
	"github.com/hyperledger/burrow/crypto"
	"github.com/hyperledger/fabric-chaincode-evm/mocks/evmcc"
	"github.com/hyperledger/fabric-chaincode-evm/statemanager"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Statemanager", func() {

	var (
		sm            statemanager.StateManager
		mockStub      *evmcc.MockStub
		addr          crypto.Address
		fakeGetLedger map[string][]byte
		fakePutLedger map[string][]byte
	)

	BeforeEach(func() {
		mockStub = &evmcc.MockStub{}
		sm = statemanager.NewStateManager(mockStub)

		var err error
		addr, err = crypto.AddressFromBytes([]byte("0000000000000address"))
		Expect(err).ToNot(HaveOccurred())
		fakeGetLedger = make(map[string][]byte)
		fakePutLedger = make(map[string][]byte)

		//Writing to a separate ledger so that writes to the ledger cannot be read in the same transaction.
		// This is more consistent with the behavior fo the ledger
		mockStub.PutStateStub = func(key string, value []byte) error {
			fakePutLedger[key] = value
			return nil
		}

		mockStub.GetStateStub = func(key string) ([]byte, error) {
			return fakeGetLedger[key], nil
		}

		mockStub.DelStateStub = func(key string) error {
			delete(fakePutLedger, key)
			return nil
		}
	})

	Describe("GetAccount", func() {
		It("returns the account associated with the address", func() {
			expectedAcct := &acm.Account{
				Address: addr,
				Code:    []byte("account code"),
			}

			encodedAcct, err := expectedAcct.Encode()
			Expect(err).ToNot(HaveOccurred())
			fakeGetLedger[addr.String()] = encodedAcct

			acct, err := sm.GetAccount(addr)
			Expect(err).ToNot(HaveOccurred())

			Expect(acct).To(Equal(expectedAcct))
		})

		Context("when no account exists", func() {
			It("returns nil", func() {
				acct, err := sm.GetAccount(addr)
				Expect(err).ToNot(HaveOccurred())

				Expect(acct).To(BeNil())
			})
		})

		Context("when GetState errors out", func() {
			BeforeEach(func() {
				mockStub.GetStateReturns(nil, errors.New("boom!"))
			})

			It("returns a nil account and an error", func() {
				acct, err := sm.GetAccount(addr)
				Expect(err).To(HaveOccurred())

				Expect(acct).To(BeNil())
			})
		})
	})

	Describe("GetStorage", func() {
		var expectedVal, key binary.Word256
		BeforeEach(func() {
			expectedVal = binary.LeftPadWord256([]byte("storage-value"))
			key = binary.LeftPadWord256([]byte("key"))
		})

		It("returns the value associated with the key", func() {
			fakeGetLedger[addr.String()+hex.EncodeToString(key.Bytes())] = expectedVal.Bytes()

			val, err := sm.GetStorage(addr, key)
			Expect(err).ToNot(HaveOccurred())

			Expect(val).To(Equal(expectedVal))
		})

		Context("when GetState returns an error", func() {
			BeforeEach(func() {
				mockStub.GetStateReturns(nil, errors.New("boom!"))
			})

			It("returns an error", func() {
				val, err := sm.GetStorage(addr, key)
				Expect(err).To(HaveOccurred())

				Expect(val).To(Equal(binary.Word256{}))
			})
		})

		Context("when a GetStorage is called after an SetStorage on the same key in the same tx", func() {
			var initialVal, updatedVal binary.Word256
			BeforeEach(func() {
				initialVal = binary.LeftPadWord256([]byte("storage-value"))
				updatedVal = binary.LeftPadWord256([]byte("updated-storage-value"))

				fakeGetLedger[addr.String()+hex.EncodeToString(key.Bytes())] = initialVal.Bytes()

				val, err := sm.GetStorage(addr, key)
				Expect(err).ToNot(HaveOccurred())
				Expect(val).To(Equal(initialVal))

				err = sm.SetStorage(addr, key, updatedVal)
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns the account that was previously written in the same tx", func() {
				val, err := sm.GetStorage(addr, key)
				Expect(err).ToNot(HaveOccurred())
				Expect(val).To(Equal(updatedVal))
			})
		})
	})

	Describe("UpdateAccount", func() {
		var initialCode []byte
		BeforeEach(func() {
			initialCode = []byte("account code")
		})

		Context("when the account didn't exist", func() {
			It("creates the account", func() {

				expectedAcct := &acm.Account{
					Address: addr,
					Code:    initialCode,
				}
				encodedAcct, err := expectedAcct.Encode()
				Expect(err).ToNot(HaveOccurred())

				err = sm.UpdateAccount(expectedAcct)
				Expect(err).ToNot(HaveOccurred())

				Expect(mockStub.PutStateCallCount()).To(Equal(1))

				key, code := mockStub.PutStateArgsForCall(0)

				Expect(key).To(Equal(addr.String()))
				Expect(code).To(Equal(encodedAcct))
			})
		})

		Context("when the account exists", func() {
			It("updates the account", func() {
				fakeGetLedger[addr.String()] = initialCode

				updatedAccount := &acm.Account{
					Address: addr,
					Code:    []byte("updated account code"),
				}

				encodedAcct, err := updatedAccount.Encode()
				Expect(err).ToNot(HaveOccurred())

				err = sm.UpdateAccount(updatedAccount)
				Expect(err).ToNot(HaveOccurred())

				Expect(mockStub.PutStateCallCount()).To(Equal(1))
				putAddr, putVal := mockStub.PutStateArgsForCall(0)
				Expect(putAddr).To(Equal(addr.String()))
				Expect(putVal).To(Equal(encodedAcct))
			})

		})

		Context("when stub throws an error", func() {
			BeforeEach(func() {
				mockStub.PutStateReturns(errors.New("boom!"))
			})

			It("returns an error", func() {
				expectedAcct := &acm.Account{
					Address: addr,
					Code:    initialCode,
				}

				err := sm.UpdateAccount(expectedAcct)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("RemoveAccount", func() {
		Context("when the account existed previously", func() {
			It("removes the account", func() {
				fakeGetLedger[addr.String()] = []byte("account code")

				err := sm.RemoveAccount(addr)
				Expect(err).ToNot(HaveOccurred())

				Expect(mockStub.DelStateCallCount()).To(Equal(1))
				delAddr := mockStub.DelStateArgsForCall(0)
				Expect(delAddr).To(Equal(addr.String()))
			})
		})

		Context("when the account did not exists previously", func() {
			It("does not return an error", func() {
				err := sm.RemoveAccount(addr)
				Expect(err).ToNot(HaveOccurred())

				Expect(mockStub.DelStateCallCount()).To(Equal(1))
				delAddr := mockStub.DelStateArgsForCall(0)
				Expect(delAddr).To(Equal(addr.String()))
			})
		})

		Context("when stub throws an error", func() {
			BeforeEach(func() {
				mockStub.DelStateReturns(errors.New("boom!"))
			})

			It("returns an error", func() {
				err := sm.RemoveAccount(addr)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("SetStorage", func() {
		var (
			key, initialVal binary.Word256
			compKey         string
		)

		BeforeEach(func() {

			initialVal = binary.LeftPadWord256([]byte("storage-value"))
			key = binary.LeftPadWord256([]byte("key"))
			compKey = addr.String() + hex.EncodeToString(key.Bytes())
		})

		Context("when key already exists", func() {
			It("updates the key value pair", func() {
				err := mockStub.PutState(compKey, initialVal.Bytes())
				Expect(err).ToNot(HaveOccurred())

				updatedVal := binary.LeftPadWord256([]byte("updated-storage-value"))

				err = sm.SetStorage(addr, key, updatedVal)
				Expect(err).ToNot(HaveOccurred())

				Expect(mockStub.PutStateCallCount()).To(Equal(2))
				putKey, putVal := mockStub.PutStateArgsForCall(1)
				Expect(putKey).To(Equal(compKey))
				Expect(putVal).To(Equal(updatedVal.Bytes()))
			})
		})

		Context("when the key does not exist", func() {
			It("creates the key value pair", func() {
				err := sm.SetStorage(addr, key, initialVal)
				Expect(err).ToNot(HaveOccurred())

				Expect(mockStub.PutStateCallCount()).To(Equal(1))
				putKey, putVal := mockStub.PutStateArgsForCall(0)
				Expect(putKey).To(Equal(compKey))
				Expect(putVal).To(Equal(initialVal.Bytes()))
			})
		})

		Context("when stub throws an error", func() {
			BeforeEach(func() {
				mockStub.PutStateReturns(errors.New("boom!"))
			})

			It("returns an error", func() {
				err := sm.SetStorage(addr, key, initialVal)
				Expect(err).To(HaveOccurred())

				val, err := mockStub.GetState(compKey)
				Expect(err).ToNot(HaveOccurred())
				Expect(val).To(BeEmpty())
			})
		})

		Context("when value is the zero value", func() {
			It("deletes the key", func() {
				err := sm.SetStorage(addr, key, initialVal)
				Expect(err).ToNot(HaveOccurred())
				Expect(mockStub.PutStateCallCount()).To(Equal(1))

				err = sm.SetStorage(addr, key, binary.Zero256)
				Expect(err).ToNot(HaveOccurred())
				Expect(mockStub.DelStateCallCount()).To(Equal(1))
				deleteKey := mockStub.DelStateArgsForCall(0)
				Expect(deleteKey).To(Equal(compKey))

			})
		})
	})
})
