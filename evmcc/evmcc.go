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

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/burrow/account"
	"github.com/hyperledger/burrow/binary"
	"github.com/hyperledger/burrow/execution/evm"
	"github.com/hyperledger/burrow/logging"
	evm_event "github.com/hyperledger/fabric-chaincode-evm/event"
	"github.com/hyperledger/fabric-chaincode-evm/statemanager"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/msp"
	pb "github.com/hyperledger/fabric/protos/peer"
	"golang.org/x/crypto/sha3"
)

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
		return shim.Error(fmt.Sprintf("failed to decode callee address from %s: %s", string(args[0]), err.Error()))
	}

	calleeAddr, err := account.AddressFromBytes(c)
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to get callee address: %s", err.Error()))
	}

	// get caller account from creator public key
	callerAddr, err := getCallerAddress(stub)
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to get caller address: %s", err.Error()))
	}
	callerAcct := account.ConcreteAccount{Address: callerAddr}.MutableAccount()

	// get input bytes from args[1]
	input, err := hex.DecodeString(string(args[1]))
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to decode input bytes: %s", err.Error()))
	}

	var gas uint64 = 10000
	state := statemanager.NewStateManager(stub)
	vm := evm.NewVM(state, newParams(), callerAddr, nil, evmLogger)

	evmgr := evm_event.NewEventManager(stub)
	vm.SetPublisher(evmgr)

	if calleeAddr == account.ZeroAddress {
		logger.Debugf("Deploy contract")

		seqKey := binary.RightPadWord256([]byte("sequence"))
		s, err := state.GetStorage(callerAddr, seqKey)
		if err != nil {
			return shim.Error(fmt.Sprintf("failed to get caller sequence"))
		}

		var seq uint64
		if s == binary.Zero256 {
			seq = 0
		} else {
			seq = binary.Uint64FromWord256(s)
		}

		// Update contract seq
		logger.Debugf("Contract sequence number = %d", seq)
		state.SetStorage(callerAddr, seqKey, binary.Uint64ToWord256(seq+1))

		contractAddr := account.NewContractAddress(callerAddr, seq)
		contractAcct := account.ConcreteAccount{Address: contractAddr}.MutableAccount()

		rtCode, err := vm.Call(callerAcct, contractAcct, input, nil, 0, &gas)
		if err != nil {
			return shim.Error(fmt.Sprintf("failed to deploy code: %s", err.Error()))
		}
		if rtCode == nil {
			return shim.Error(fmt.Sprintf("nil bytecode"))
		}

		contractAcct.SetCode(rtCode)
		if err = state.UpdateAccount(contractAcct); err != nil {
			return shim.Error(fmt.Sprintf("failed to update contract account: %s", err.Error()))
		}

		// return encoded hex bytes for human-readability
		return shim.Success([]byte(hex.EncodeToString(contractAddr.Bytes())))
	} else {
		logger.Debugf("Invoke contract at %x", calleeAddr.Bytes())

		calleeAcct, err := state.GetAccount(calleeAddr)
		if err != nil {
			return shim.Error(fmt.Sprintf("failed to retrieve contract code: %s", err.Error()))
		}

		output, err := vm.Call(callerAcct, account.AsMutableAccount(calleeAcct), calleeAcct.Code().Bytes(), input, 0, &gas)
		if err != nil {
			return shim.Error(fmt.Sprintf("failed to execute contract: %s", err.Error()))
		}

		// Passing the function hash of the method that has triggered the event
		// The function hash is the first 8 bytes of the Input argument
		er := evmgr.Flush(string(args[1][0:8]))
		if er != nil {
			return shim.Error(fmt.Sprintf("error in Flush: %s", er.Error()))
		}

		return shim.Success(output)
	}
}

func (evmcc *EvmChaincode) getCode(stub shim.ChaincodeStubInterface, address []byte) pb.Response {
	c, err := hex.DecodeString(string(address))
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to decode callee address from %s: %s", string(address), err.Error()))
	}

	calleeAddr, err := account.AddressFromBytes(c)
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to get callee address: %s", err.Error()))
	}

	code, err := stub.GetState(calleeAddr.String())
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to get contract account: %s", err.Error()))
	}

	return shim.Success([]byte(hex.EncodeToString(code)))
}

func (evmcc *EvmChaincode) account(stub shim.ChaincodeStubInterface) pb.Response {
	creatorBytes, err := stub.GetCreator()
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to get creator: %s", err.Error()))
	}

	si := &msp.SerializedIdentity{}
	if err = proto.Unmarshal(creatorBytes, si); err != nil {
		return shim.Error(fmt.Sprintf("failed to unmarshal serialized identity: %s", err.Error()))
	}

	callerAddr, err := identityToAddr(si.IdBytes)
	if err != nil {
		return shim.Error(fmt.Sprintf("fail to convert identity to address: %s", err.Error()))
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

func getCallerAddress(stub shim.ChaincodeStubInterface) (account.Address, error) {
	creatorBytes, err := stub.GetCreator()
	if err != nil {
		return account.ZeroAddress, fmt.Errorf("failed to get creator: %s", err)
	}

	si := &msp.SerializedIdentity{}
	if err = proto.Unmarshal(creatorBytes, si); err != nil {
		return account.ZeroAddress, fmt.Errorf("failed to unmarshal serialized identity: %s", err)
	}

	callerAddr, err := identityToAddr(si.IdBytes)
	if err != nil {
		return account.ZeroAddress, fmt.Errorf("fail to convert identity to address: %s", err.Error())
	}

	return callerAddr, nil
}

func identityToAddr(id []byte) (account.Address, error) {
	bl, _ := pem.Decode(id)
	if bl == nil {
		return account.ZeroAddress, fmt.Errorf("no pem data found")
	}

	cert, err := x509.ParseCertificate(bl.Bytes)
	if err != nil {
		return account.ZeroAddress, fmt.Errorf("failed to parse certificate: %s", err)
	}

	pubkeyBytes, err := x509.MarshalPKIXPublicKey(cert.PublicKey)
	if err != nil {
		return account.ZeroAddress, fmt.Errorf("unable to marshal public key: %s", err)
	}

	return account.AddressFromWord256(sha3.Sum256(pubkeyBytes)), nil
}

func main() {
	if err := shim.Start(new(EvmChaincode)); err != nil {
		logger.Infof("Error starting EVM chaincode: %s", err.Error())
	}
}
