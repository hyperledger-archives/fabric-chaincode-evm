/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package statemanager

import (
	"encoding/hex"
	"strings"

	"github.com/hyperledger/burrow/acm"
	"github.com/hyperledger/burrow/binary"
	"github.com/hyperledger/burrow/crypto"
	"github.com/hyperledger/fabric/core/chaincode/shim"
)

type StateManager interface {
	GetAccount(address crypto.Address) (*acm.Account, error)
	GetStorage(address crypto.Address, key binary.Word256) (binary.Word256, error)
	UpdateAccount(updatedAccount *acm.Account) error
	RemoveAccount(address crypto.Address) error
	SetStorage(address crypto.Address, key, value binary.Word256) error
}

type stateManager struct {
	stub shim.ChaincodeStubInterface
	// We will be looking into adding a cache for accounts later
	// The cache can be single threaded because the statemanager is 1-1 with the evm which is single threaded.
	cache map[string]binary.Word256
}

func NewStateManager(stub shim.ChaincodeStubInterface) StateManager {
	return &stateManager{
		stub:  stub,
		cache: make(map[string]binary.Word256),
	}
}

func (s *stateManager) GetAccount(address crypto.Address) (*acm.Account, error) {
	acctBytes, err := s.stub.GetState(strings.ToLower(address.String()))
	if err != nil {
		return nil, err
	}

	if len(acctBytes) == 0 {
		return nil, nil
	}

	return acm.Decode(acctBytes)
}

func (s *stateManager) GetStorage(address crypto.Address, key binary.Word256) (binary.Word256, error) {
	compKey := strings.ToLower(address.String()) + hex.EncodeToString(key.Bytes())

	if val, ok := s.cache[compKey]; ok {
		return val, nil
	}

	val, err := s.stub.GetState(compKey)
	if err != nil {
		return binary.Word256{}, err
	}

	return binary.LeftPadWord256(val), nil
}

func (s *stateManager) UpdateAccount(updatedAccount *acm.Account) error {
	encodedAcct, err := updatedAccount.Encode()
	if err != nil {
		return err
	}
	return s.stub.PutState(hex.EncodeToString(updatedAccount.Address.Bytes()), encodedAcct)
}

func (s *stateManager) RemoveAccount(address crypto.Address) error {
	return s.stub.DelState(strings.ToLower(address.String()))
}

func (s *stateManager) SetStorage(address crypto.Address, key, value binary.Word256) error {
	compKey := strings.ToLower(address.String()) + hex.EncodeToString(key.Bytes())

	var err error
	if value == binary.Zero256 {
		return s.stub.DelState(compKey)
	}

	if err = s.stub.PutState(compKey, value.Bytes()); err == nil {
		s.cache[compKey] = value
	}

	return err
}
