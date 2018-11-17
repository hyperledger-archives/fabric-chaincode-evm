/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package statemanager

import (
	"encoding/hex"

	"github.com/hyperledger/burrow/account"
	"github.com/hyperledger/burrow/binary"
	"github.com/hyperledger/burrow/permission"
	"github.com/hyperledger/burrow/permission/types"
	"github.com/hyperledger/fabric/core/chaincode/shim"
)

//Permissions for contract to send CallTx or SendTx to another contract
const ContractPermFlags = permission.Call | permission.Send

var ContractPerms = types.AccountPermissions{
	Base: types.BasePermissions{
		Perms:  ContractPermFlags,
		SetBit: ContractPermFlags,
	},
	Roles: []string{},
}

type StateManager interface {
	GetAccount(address account.Address) (account.Account, error)
	GetStorage(address account.Address, key binary.Word256) (binary.Word256, error)
	UpdateAccount(updatedAccount account.Account) error
	RemoveAccount(address account.Address) error
	SetStorage(address account.Address, key, value binary.Word256) error
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

func (s *stateManager) GetAccount(address account.Address) (account.Account, error) {
	code, err := s.stub.GetState(address.String())
	if err != nil {
		return account.ConcreteAccount{}.Account(), err
	}

	if len(code) == 0 {
		return account.ConcreteAccount{}.Account(), nil
	}

	acct := account.ConcreteAccount{
		Address: address,
		Code:    code,
	}.MutableAccount()

	//Setting permission on account to allow contract invocations and queries
	acct.SetPermissions(ContractPerms)
	return acct, nil
}

func (s *stateManager) GetStorage(address account.Address, key binary.Word256) (binary.Word256, error) {
	compKey := address.String() + hex.EncodeToString(key.Bytes())

	if val, ok := s.cache[compKey]; ok {
		return val, nil
	}

	val, err := s.stub.GetState(compKey)
	if err != nil {
		return binary.Word256{}, err
	}

	return binary.LeftPadWord256(val), nil
}

func (s *stateManager) UpdateAccount(updatedAccount account.Account) error {
	return s.stub.PutState(updatedAccount.Address().String(), updatedAccount.Code().Bytes())
}

func (s *stateManager) RemoveAccount(address account.Address) error {
	return s.stub.DelState(address.String())
}

func (s *stateManager) SetStorage(address account.Address, key, value binary.Word256) error {
	var err error

	if err = s.stub.PutState(address.String()+hex.EncodeToString(key.Bytes()), value.Bytes()); err == nil {
		s.cache[address.String()+hex.EncodeToString(key.Bytes())] = value
	}

	return err
}
