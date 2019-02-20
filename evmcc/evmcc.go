/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/burrow/acm"
	"github.com/hyperledger/burrow/binary"
	"github.com/hyperledger/burrow/crypto"
	"github.com/hyperledger/burrow/execution/evm"
	"github.com/hyperledger/burrow/logging"
	"github.com/hyperledger/burrow/permission"
	"github.com/hyperledger/fabric-chaincode-evm/eventmanager"
	"github.com/hyperledger/fabric-chaincode-evm/statemanager"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/msp"
	pb "github.com/hyperledger/fabric/protos/peer"
	"golang.org/x/crypto/sha3"
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
	evmCache := evm.NewState(state)
	eventSink := &eventmanager.EventManager{Stub: stub}
	vm := evm.NewVM(newParams(), callerAddr, nil, evmLogger)

	if calleeAddr == crypto.ZeroAddress {
		logger.Debugf("Deploy contract")

		// Sequence number is used to create the contract address.
		seq := evmCache.GetSequence(callerAddr)

		// Sequence number of 0 means this is the caller's first contract
		// Therefore a new account needs to be created for them to keep track of their sequence.
		if seq == 0 {
			evmCache.CreateAccount(callerAddr)
			if evmErr := evmCache.Error(); evmErr != nil {
				return shim.Error(fmt.Sprintf("failed to create user account: %s ", evmErr))
			}
		}

		// Update contract seq
		// If sequence is not incremented every contract a person deploys with have the same contract address.
		logger.Debugf("Contract sequence number = %d", seq)
		evmCache.IncSequence(callerAddr)
		if evmErr := evmCache.Error(); evmErr != nil {
			return shim.Error(fmt.Sprintf("failed to update user account sequence number: %s ", evmErr))
		}

		contractAddr := crypto.NewContractAddress(callerAddr, seq)
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
	creatorBytes, err := stub.GetCreator()
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to get creator: %s", err))
	}

	si := &msp.SerializedIdentity{}
	if err = proto.Unmarshal(creatorBytes, si); err != nil {
		return shim.Error(fmt.Sprintf("failed to unmarshal serialized identity: %s", err))
	}

	callerAddr, err := identityToAddr(si.IdBytes)
	if err != nil {
		return shim.Error(fmt.Sprintf("fail to convert identity to address: %s", err))
	}

	return shim.Success([]byte(callerAddr.String()))
}

func newParams() evm.Params {
	return evm.Params{
		BlockHeight: 0,
		BlockHash:   binary.Zero256,
		BlockTime:   0,
		GasLimit:    0,
	}
}

func getCallerAddress(stub shim.ChaincodeStubInterface) (crypto.Address, error) {
	creatorBytes, err := stub.GetCreator()
	if err != nil {
		return crypto.ZeroAddress, fmt.Errorf("failed to get creator: %s", err)
	}

	si := &msp.SerializedIdentity{}
	if err = proto.Unmarshal(creatorBytes, si); err != nil {
		return crypto.ZeroAddress, fmt.Errorf("failed to unmarshal serialized identity: %s", err)
	}

	callerAddr, err := identityToAddr(si.IdBytes)
	if err != nil {
		return crypto.ZeroAddress, fmt.Errorf("fail to convert identity to address: %s", err)
	}

	return callerAddr, nil
}

func identityToAddr(id []byte) (crypto.Address, error) {
	bl, _ := pem.Decode(id)
	if bl == nil {
		return crypto.ZeroAddress, fmt.Errorf("no pem data found")
	}

	cert, err := x509.ParseCertificate(bl.Bytes)
	if err != nil {
		return crypto.ZeroAddress, fmt.Errorf("failed to parse certificate: %s", err)
	}

	pubkeyBytes, err := x509.MarshalPKIXPublicKey(cert.PublicKey)
	if err != nil {
		return crypto.ZeroAddress, fmt.Errorf("unable to marshal public key: %s", err)
	}

	return crypto.AddressFromWord256(sha3.Sum256(pubkeyBytes)), nil
}

func main() {
	if err := shim.Start(new(EvmChaincode)); err != nil {
		logger.Infof("Error starting EVM chaincode: %s", err)
	}
}
