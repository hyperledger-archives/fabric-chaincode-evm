/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/hyperledger/burrow/acm"
	"github.com/hyperledger/burrow/crypto"
	"github.com/hyperledger/burrow/execution/evm"
	"github.com/hyperledger/burrow/logging"
	"github.com/hyperledger/burrow/permission"
	"github.com/hyperledger/fabric-chaincode-evm/address"
	"github.com/hyperledger/fabric-chaincode-evm/eventmanager"
	"github.com/hyperledger/fabric-chaincode-evm/statemanager"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

//Permissions for all accounts (users & contracts) to send CallTx or SendTx to a contract
const ContractPermFlags = permission.Call | permission.Send

var ContractPerms = permission.AccountPermissions{
	Base: permission.BasePermissions{
		Perms:  ContractPermFlags,
		SetBit: ContractPermFlags,
	},
}

var logger = flogging.MustGetLogger("evmcc")
var evmLogger = logging.NewNoopLogger()

type EvmChaincode struct{}

func (evmcc *EvmChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
	logger.Debugf("Init evmcc, it's no-op")
	return shim.Success(nil)
}

func (evmcc *EvmChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	// We always expect 2 args: 'callee address, input data' or ' getCode ,  contract address'
	args := stub.GetArgs()

	if len(args) == 1 {
		if string(args[0]) == "account" {
			return evmcc.account(stub)
		}
	}

	if len(args) != 2 {
		return shim.Error(fmt.Sprintf("expects 2 args, got %d : %s", len(args), string(args[0])))
	}

	if string(args[0]) == "getCode" {
		return evmcc.getCode(stub, args[1])
	}

	c, err := hex.DecodeString(string(args[0]))
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to decode callee address from %s: %s", string(args[0]), err))
	}

	calleeAddr, err := crypto.AddressFromBytes(c)
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to get callee address: %s", err))
	}

	// get caller account from creator public key
	callerAddr, err := getCallerAddress(stub)
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to get caller address: %s", err))
	}

	// get input bytes from args[1]
	input, err := hex.DecodeString(string(args[1]))
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to decode input bytes: %s", err))
	}

	var gas uint64 = 10000
	state := statemanager.NewStateManager(stub)
	evmCache := evm.NewState(state, func(height uint64) []byte {
		// This function is to be used to return the block hash
		// Currently EVMCC does not support the BLOCKHASH opcode.
		// This function is only used for that opcode and will not
		// affect execution if BLOCKHASH is not called.
		panic("Block Hash shouldn't be called")
	})
	eventSink := &eventmanager.EventManager{Stub: stub}
	nonce := crypto.Nonce(callerAddr, []byte(stub.GetTxID()))
	vm := evm.NewVM(newParams(), callerAddr, nonce, evmLogger)

	if calleeAddr == crypto.ZeroAddress {
		logger.Debugf("Deploy contract")

		logger.Debugf("Contract nonce number = %d", nonce)
		contractAddr := crypto.NewContractAddress(callerAddr, nonce)
		// Contract account needs to be created before setting code to it
		evmCache.CreateAccount(contractAddr)
		if evmErr := evmCache.Error(); evmErr != nil {
			return shim.Error(fmt.Sprintf("failed to create the contract account: %s ", evmErr))
		}

		evmCache.SetPermission(contractAddr, ContractPermFlags, true)
		if evmErr := evmCache.Error(); evmErr != nil {
			return shim.Error(fmt.Sprintf("failed to set contract account permissions: %s ", evmErr))
		}

		rtCode, evmErr := vm.Call(evmCache, eventSink, callerAddr, contractAddr, input, input, 0, &gas)
		if evmErr != nil {
			return shim.Error(fmt.Sprintf("failed to deploy code: %s", evmErr))
		}
		if rtCode == nil {
			return shim.Error(fmt.Sprintf("nil bytecode"))
		}

		evmCache.InitCode(contractAddr, rtCode)
		if evmErr := evmCache.Error(); evmErr != nil {
			return shim.Error(fmt.Sprintf("failed to update contract account: %s", evmErr))
		}

		// Passing the first 8 bytes contract address just created
		err := eventSink.Flush(string(contractAddr.Bytes()[0:8]))
		if err != nil {
			return shim.Error(fmt.Sprintf("error in Flush: %s", err))
		}

		if evmErr := evmCache.Sync(); evmErr != nil {
			return shim.Error(fmt.Sprintf("failed to sync: %s", evmErr))
		}
		// return encoded hex bytes for human-readability
		return shim.Success([]byte(hex.EncodeToString(contractAddr.Bytes())))
	} else {
		logger.Debugf("Invoke contract at %x", calleeAddr.Bytes())

		calleeCode := evmCache.GetCode(calleeAddr)
		if evmErr := evmCache.Error(); evmErr != nil {
			return shim.Error(fmt.Sprintf("failed to retrieve contract code: %s", evmErr))
		}

		output, evmErr := vm.Call(evmCache, eventSink, callerAddr, calleeAddr, calleeCode, input, 0, &gas)
		if evmErr != nil {
			return shim.Error(fmt.Sprintf("failed to execute contract: %s", evmErr))
		}

		// Passing the function hash of the method that has triggered the event
		// The function hash is the first 8 bytes of the Input argument
		err := eventSink.Flush(string(args[1][0:8]))
		if err != nil {
			return shim.Error(fmt.Sprintf("error in Flush: %s", err))
		}

		// Sync is required for evm to send writes to the statemanager.
		if evmErr := evmCache.Sync(); evmErr != nil {
			return shim.Error(fmt.Sprintf("failed to sync: %s", evmErr))
		}

		return shim.Success(output)
	}
}

func (evmcc *EvmChaincode) getCode(stub shim.ChaincodeStubInterface, address []byte) pb.Response {
	c, err := hex.DecodeString(string(address))
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to decode callee address from %s: %s", string(address), err))
	}

	calleeAddr, err := crypto.AddressFromBytes(c)
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to get callee address: %s", err))
	}

	acctBytes, err := stub.GetState(strings.ToLower(calleeAddr.String()))
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to get contract account: %s", err))
	}

	if len(acctBytes) == 0 {
		return shim.Success(acctBytes)
	}

	acct, err := acm.Decode(acctBytes)
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to decode contract account: %s", err))
	}

	return shim.Success([]byte(hex.EncodeToString(acct.Code.Bytes())))
}

func (evmcc *EvmChaincode) account(stub shim.ChaincodeStubInterface) pb.Response {
	callerAddr, err := getCallerAddress(stub)
	if err != nil {
		return shim.Error(fmt.Sprintf("fail to convert identity to address: %s", err))
	}
	return shim.Success([]byte(callerAddr.String()))
}

func newParams() evm.Params {
	return evm.Params{
		BlockHeight: 0,
		BlockTime:   0,
		GasLimit:    0,
	}
}

func getCallerAddress(stub shim.ChaincodeStubInterface) (crypto.Address, error) {
	creatorBytes, err := stub.GetCreator()
	if err != nil {
		return crypto.ZeroAddress, fmt.Errorf("failed to get creator: %s", err)
	}

	callerAddr, err := address.IdentityToAddr(creatorBytes)
	if err != nil {
		return crypto.ZeroAddress, fmt.Errorf("fail to convert identity to address: %s", err)
	}

	return crypto.AddressFromBytes(callerAddr)
}

func main() {
	if err := shim.Start(new(EvmChaincode)); err != nil {
		logger.Infof("Error starting EVM chaincode: %s", err)
	}
}
