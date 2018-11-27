/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main_test

import (
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/hyperledger/burrow/account"
	"github.com/hyperledger/burrow/binary"
	"github.com/hyperledger/burrow/execution/evm/events"
	evm "github.com/hyperledger/fabric-chaincode-evm/evmcc"
	evmcc_mocks "github.com/hyperledger/fabric-chaincode-evm/mocks/evmcc"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/msp"
	"golang.org/x/crypto/sha3"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("evmcc", func() {
	marshalCreator := func(mspId string, certByte []byte) []byte {
		b, err := proto.Marshal(&msp.SerializedIdentity{Mspid: mspId, IdBytes: certByte})
		if err != nil || b == nil {
			panic("Failed to marshal identity ")
		}
		return b
	}

	var (
		evmcc      shim.Chaincode
		stub       *evmcc_mocks.MockStub
		fakeLedger map[string][]byte
	)

	BeforeEach(func() {
		evmcc = &evm.EvmChaincode{}
		stub = &evmcc_mocks.MockStub{}
		fakeLedger = make(map[string][]byte)

		stub.PutStateStub = func(key string, value []byte) error {
			fakeLedger[key] = value
			return nil
		}

		stub.GetStateStub = func(key string) ([]byte, error) {
			return fakeLedger[key], nil
		}

		stub.DelStateStub = func(key string) error {
			delete(fakeLedger, key)
			return nil
		}

	})

	Describe("Init", func() {
		It("returns an OK response", func() {
			res := evmcc.Init(stub)
			Expect(res.Status).To(Equal(int32(shim.OK)))
			Expect(res.Payload).To(Equal([]byte(nil)))
		})
	})

	Describe("Invoke", func() {
		var (
			user0Cert = `-----BEGIN CERTIFICATE-----
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

			creator = marshalCreator("TestOrg", []byte(user0Cert))

			/* Sample App from https://solidity.readthedocs.io/en/develop/introduction-to-smart-contracts.html#storage
			   pragma solidity ^0.4.0;
			   contract SimpleStorage {
			     uint storedData;
			   	function set(uint x) public {
			   	  storedData = x;
			   	}
			   	function get() public constant returns (uint) {
			   	  return storedData;
			   	}
			   }
			*/

			deployCode  = []byte("6060604052341561000f57600080fd5b60d38061001d6000396000f3006060604052600436106049576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806360fe47b114604e5780636d4ce63c14606e575b600080fd5b3415605857600080fd5b606c60048080359060200190919050506094565b005b3415607857600080fd5b607e609e565b6040518082815260200191505060405180910390f35b8060008190555050565b600080549050905600a165627a7a72305820122f55f799d70b5f6dbfd4312efb65cdbfaacddedf7c36249b8b1e915a8dd85b0029")
			runtimeCode = "6060604052600436106049576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806360fe47b114604e5780636d4ce63c14606e575b600080fd5b3415605857600080fd5b606c60048080359060200190919050506094565b005b3415607857600080fd5b607e609e565b6040518082815260200191505060405180910390f35b8060008190555050565b600080549050905600a165627a7a72305820122f55f799d70b5f6dbfd4312efb65cdbfaacddedf7c36249b8b1e915a8dd85b0029"
		)

		BeforeEach(func() {
			// Set contract creator
			stub.GetCreatorReturns(creator, nil)
		})

		It("will create and store the runtime bytecode from the deploy bytecode", func() {
			// zero address, and deploy code is contract creation
			stub.GetArgsReturns([][]byte{[]byte(account.ZeroAddress.String()), deployCode})
			res := evmcc.Invoke(stub)
			Expect(res.Status).To(Equal(int32(shim.OK)))

			// First PutState Call is to store the current sequence number
			Expect(stub.PutStateCallCount()).To(Equal(2))
			key, value := stub.PutStateArgsForCall(1)

			Expect(strings.ToLower(key)).To(Equal(strings.ToLower(string(res.Payload))))
			Expect(hex.EncodeToString(value)).To(Equal(runtimeCode))
		})

		Context("when a contract has already been deployed", func() {
			var (
				contractAddress account.Address
				SET             = "60fe47b1"
				GET             = "6d4ce63c"
			)

			BeforeEach(func() {
				// zero address, and deploy code is contract creation
				stub.GetArgsReturns([][]byte{[]byte(account.ZeroAddress.String()), deployCode})
				res := evmcc.Invoke(stub)
				Expect(res.Status).To(Equal(int32(shim.OK)))
				Expect(stub.PutStateCallCount()).To(Equal(2))

				var err error
				contractAddress, err = account.AddressFromHexString(string(res.Payload))
				Expect(err).ToNot(HaveOccurred())
			})

			It("can run the methods of the contract", func() {
				stub.GetArgsReturns([][]byte{[]byte(contractAddress.String()), []byte(GET)})
				res := evmcc.Invoke(stub)
				Expect(res.Status).To(Equal(int32(shim.OK)))
				Expect(hex.EncodeToString(res.Payload)).To(Equal("0000000000000000000000000000000000000000000000000000000000000000"))

				stub.GetArgsReturns([][]byte{[]byte(contractAddress.String()), []byte(SET + "000000000000000000000000000000000000000000000000000000000000002a")})
				res = evmcc.Invoke(stub)
				Expect(res.Status).To(Equal(int32(shim.OK)))

				stub.GetArgsReturns([][]byte{[]byte(contractAddress.String()), []byte(GET)})
				res = evmcc.Invoke(stub)
				Expect(res.Status).To(Equal(int32(shim.OK)))
				Expect(hex.EncodeToString(res.Payload)).To(Equal("000000000000000000000000000000000000000000000000000000000000002a"))
			})

			Context("when getCode is invoked", func() {
				BeforeEach(func() {
					stub.GetArgsReturns([][]byte{[]byte("getCode"), []byte(contractAddress.String())})
				})
				It("will return the runtime bytecode of the contract", func() {
					res := evmcc.Invoke(stub)
					Expect(res.Status).To(Equal(int32(shim.OK)))
					Expect(string(res.Payload)).To(Equal(runtimeCode))
				})
			})

			Context("when another contract is deployed", func() {
				BeforeEach(func() {
					stub.GetArgsReturns([][]byte{[]byte(account.ZeroAddress.String()), deployCode})
				})
				It("creates a new contract and returns another contract address", func() {
					res := evmcc.Invoke(stub)
					Expect(res.Status).To(Equal(int32(shim.OK)))
					Expect(string(res.Payload)).ToNot(Equal(string(contractAddress.Bytes())))
				})
			})

		})

		Context("when more than 2 args are given", func() {
			BeforeEach(func() {
				stub.GetArgsReturns([][]byte{[]byte("arg1"), []byte("arg2"), []byte("arg3")})
			})

			It("returns an error", func() {
				res := evmcc.Invoke(stub)
				Expect(res.Status).To(Equal(int32(shim.ERROR)))
				Expect(res.Message).To(ContainSubstring("expects 2 args"))
			})
		})

		Context("when less than 2 args are given", func() {
			Context("when only one argument is given", func() {
				Context("when the argument is account", func() {
					var callerAddress account.Address

					BeforeEach(func() {
						stub.GetArgsReturns([][]byte{[]byte("account")})
						stub.GetCreatorReturns(creator, nil)
						si := &msp.SerializedIdentity{IdBytes: []byte(user0Cert)}

						var err error
						callerAddress, err = identityToAddr(si.IdBytes)
						Expect(err).ToNot(HaveOccurred())
					})

					It("will return the caller address of the contract", func() {
						res := evmcc.Invoke(stub)
						Expect(res.Status).To(Equal(int32(shim.OK)))
						Expect(string(res.Payload)).To(Equal(callerAddress.String()))
					})
				})

				Context("when the argument is not account", func() {
					BeforeEach(func() {
						stub.GetArgsReturns([][]byte{[]byte("arg1")})
					})

					It("returns an error", func() {
						res := evmcc.Invoke(stub)
						Expect(res.Status).To(Equal(int32(shim.ERROR)))
						Expect(res.Message).To(ContainSubstring("expects 2 args"))
					})
				})
			})

			Context("when no argument is given", func() {
				BeforeEach(func() {
					stub.GetArgsReturns([][]byte{[]byte("")})
				})

				It("returns an error", func() {
					res := evmcc.Invoke(stub)
					Expect(res.Status).To(Equal(int32(shim.ERROR)))
					Expect(res.Message).To(ContainSubstring("expects 2 args"))
				})
			})
		})

		Describe("Voting DApp", func() {
			var (
				/* Voting App from https://solidity.readthedocs.io/en/develop/solidity-by-example.html#voting
				pragma solidity ^0.4.16;
				/// @title Voting with delegation.
				contract Ballot {
				    // This declares a new complex type which will
				    // be used for variables later.
				    // It will represent a single voter.
				    struct Voter {
				        uint weight; // weight is accumulated by delegation
				        bool voted;  // if true, that person already voted
				        address delegate; // person delegated to
				        uint vote;   // index of the voted proposal
				    }
				    // This is a type for a single proposal.
				    struct Proposal {
				        bytes32 name;   // short name (up to 32 bytes)
				        uint voteCount; // number of accumulated votes
				    }
				    address public chairperson;
				    // This declares a state variable that
				    // stores a `Voter` struct for each possible address.
				    mapping(address => Voter) public voters;
				    // A dynamically-sized array of `Proposal` structs.
				    Proposal[] public proposals;
				    /// Create a new ballot to choose one of `proposalNames`.
				    function Ballot(bytes32[] proposalNames) public {
				        chairperson = msg.sender;
				        voters[chairperson].weight = 1;
				        // For each of the provided proposal names,
				        // create a new proposal object and add it
				        // to the end of the array.
				        for (uint i = 0; i < proposalNames.length; i++) {
				            // `Proposal({...})` creates a temporary
				            // Proposal object and `proposals.push(...)`
				            // appends it to the end of `proposals`.
				            proposals.push(Proposal({
				                name: proposalNames[i],
				                voteCount: 0
				            }));
				        }
				    }
				    // Give `voter` the right to vote on this ballot.
				    // May only be called by `chairperson`.
				    function giveRightToVote(address voter) public {
				        // If the argument of `require` evaluates to `false`,
				        // it terminates and reverts all changes to
				        // the state and to Ether balances. It is often
				        // a good idea to use this if functions are
				        // called incorrectly. But watch out, this
				        // will currently also consume all provided gas
				        // (this is planned to change in the future).
				        require((msg.sender == chairperson) && !voters[voter].voted && (voters[voter].weight == 0));
				        voters[voter].weight = 1;
				    }
				    /// Delegate your vote to the voter `to`.
				    function delegate(address to) public {
				        // assigns reference
				        Voter storage sender = voters[msg.sender];
				        require(!sender.voted);
				        // Self-delegation is not allowed.
				        require(to != msg.sender);
				        // Forward the delegation as long as
				        // `to` also delegated.
				        // In general, such loops are very dangerous,
				        // because if they run too long, they might
				        // need more gas than is available in a block.
				        // In this case, the delegation will not be executed,
				        // but in other situations, such loops might
				        // cause a contract to get "stuck" completely.
				        while (voters[to].delegate != address(0)) {
				            to = voters[to].delegate;
				            // We found a loop in the delegation, not allowed.
				            require(to != msg.sender);
				        }
				        // Since `sender` is a reference, this
				        // modifies `voters[msg.sender].voted`
				        sender.voted = true;
				        sender.delegate = to;
				        Voter storage delegate = voters[to];
				        if (delegate.voted) {
				            // If the delegate already voted,
				            // directly add to the number of votes
				            proposals[delegate.vote].voteCount += sender.weight;
				        } else {
				            // If the delegate did not vote yet,
				            // add to her weight.
				            delegate.weight += sender.weight;
				        }
				    }
				    /// Give your vote (including votes delegated to you)
				    /// to proposal `proposals[proposal].name`.
				    function vote(uint proposal) public {
				        Voter storage sender = voters[msg.sender];
				        require(!sender.voted);
				        sender.voted = true;
				        sender.vote = proposal;
				        // If `proposal` is out of the range of the array,
				        // this will throw automatically and revert all
				        // changes.
				        proposals[proposal].voteCount += sender.weight;
				    }
				    /// @dev Computes the winning proposal taking all
				    /// previous votes into account.
				    function winningProposal() public view
				            returns (uint winningProposal)
				    {
				        uint winningVoteCount = 0;
				        for (uint p = 0; p < proposals.length; p++) {
				            if (proposals[p].voteCount > winningVoteCount) {
				                winningVoteCount = proposals[p].voteCount;
				                winningProposal = p;
				            }
				        }
				    }
				    // Calls winningProposal() function to get the index
				    // of the winner contained in the proposals array and then
				    // returns the name of the winner
				    function winnerName() public view
				            returns (bytes32 winnerName)
				    {
				        winnerName = proposals[winningProposal()].name;
				    }
				}
				*/
				contractByteCode = "6060604052341561000f57600080fd5b604051610b0b380380610b0b833981016040528080518201919050506000336000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555060018060008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060000181905550600090505b815181101561016757600280548060010182816100f7919061016e565b916000526020600020906002020160006040805190810160405280868681518110151561012057fe5b9060200190602002015160001916815260200160008152509091909150600082015181600001906000191690556020820151816001015550505080806001019150506100da565b50506101cf565b81548183558181151161019b5760020281600202836000526020600020918201910161019a91906101a0565b5b505050565b6101cc91905b808211156101c8576000808201600090556001820160009055506002016101a6565b5090565b90565b61092d806101de6000396000f30060606040526004361061008e576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff1680630121b93f14610093578063013cf08b146100b65780632e4176cf146100fc5780635c19a95c14610151578063609ff1bd1461018a5780639e7b8d61146101b3578063a3ec138d146101ec578063e2ba53f01461027e575b600080fd5b341561009e57600080fd5b6100b460048080359060200190919050506102af565b005b34156100c157600080fd5b6100d7600480803590602001909190505061036c565b6040518083600019166000191681526020018281526020019250505060405180910390f35b341561010757600080fd5b61010f61039f565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b341561015c57600080fd5b610188600480803573ffffffffffffffffffffffffffffffffffffffff169060200190919050506103c4565b005b341561019557600080fd5b61019d6106ae565b6040518082815260200191505060405180910390f35b34156101be57600080fd5b6101ea600480803573ffffffffffffffffffffffffffffffffffffffff16906020019091905050610729565b005b34156101f757600080fd5b610223600480803573ffffffffffffffffffffffffffffffffffffffff16906020019091905050610875565b60405180858152602001841515151581526020018373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200182815260200194505050505060405180910390f35b341561028957600080fd5b6102916108d2565b60405180826000191660001916815260200191505060405180910390f35b6000600160003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002090508060010160009054906101000a900460ff1615151561031057600080fd5b60018160010160006101000a81548160ff021916908315150217905550818160020181905550806000015460028381548110151561034a57fe5b9060005260206000209060020201600101600082825401925050819055505050565b60028181548110151561037b57fe5b90600052602060002090600202016000915090508060000154908060010154905082565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b600080600160003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002091508160010160009054906101000a900460ff1615151561042657600080fd5b3373ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff161415151561046157600080fd5b5b600073ffffffffffffffffffffffffffffffffffffffff16600160008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060010160019054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1614151561059f57600160008473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060010160019054906101000a900473ffffffffffffffffffffffffffffffffffffffff1692503373ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff161415151561059a57600080fd5b610462565b60018260010160006101000a81548160ff021916908315150217905550828260010160016101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550600160008473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002090508060010160009054906101000a900460ff16156106925781600001546002826002015481548110151561066f57fe5b9060005260206000209060020201600101600082825401925050819055506106a9565b816000015481600001600082825401925050819055505b505050565b6000806000809150600090505b60028054905081101561072457816002828154811015156106d857fe5b9060005260206000209060020201600101541115610717576002818154811015156106ff57fe5b90600052602060002090600202016001015491508092505b80806001019150506106bb565b505090565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff161480156107d25750600160008273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060010160009054906101000a900460ff16155b801561082057506000600160008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060000154145b151561082b57600080fd5b60018060008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000206000018190555050565b60016020528060005260406000206000915090508060000154908060010160009054906101000a900460ff16908060010160019054906101000a900473ffffffffffffffffffffffffffffffffffffffff16908060020154905084565b600060026108de6106ae565b8154811015156108ea57fe5b9060005260206000209060020201600001549050905600a165627a7a723058209216e84efeb17007ba61a1573380cb306de0e38c64eb02e0f9362367121816080029"
				runtimeByteCode  = "60606040526004361061008e576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff1680630121b93f14610093578063013cf08b146100b65780632e4176cf146100fc5780635c19a95c14610151578063609ff1bd1461018a5780639e7b8d61146101b3578063a3ec138d146101ec578063e2ba53f01461027e575b600080fd5b341561009e57600080fd5b6100b460048080359060200190919050506102af565b005b34156100c157600080fd5b6100d7600480803590602001909190505061036c565b6040518083600019166000191681526020018281526020019250505060405180910390f35b341561010757600080fd5b61010f61039f565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b341561015c57600080fd5b610188600480803573ffffffffffffffffffffffffffffffffffffffff169060200190919050506103c4565b005b341561019557600080fd5b61019d6106ae565b6040518082815260200191505060405180910390f35b34156101be57600080fd5b6101ea600480803573ffffffffffffffffffffffffffffffffffffffff16906020019091905050610729565b005b34156101f757600080fd5b610223600480803573ffffffffffffffffffffffffffffffffffffffff16906020019091905050610875565b60405180858152602001841515151581526020018373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200182815260200194505050505060405180910390f35b341561028957600080fd5b6102916108d2565b60405180826000191660001916815260200191505060405180910390f35b6000600160003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002090508060010160009054906101000a900460ff1615151561031057600080fd5b60018160010160006101000a81548160ff021916908315150217905550818160020181905550806000015460028381548110151561034a57fe5b9060005260206000209060020201600101600082825401925050819055505050565b60028181548110151561037b57fe5b90600052602060002090600202016000915090508060000154908060010154905082565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b600080600160003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002091508160010160009054906101000a900460ff1615151561042657600080fd5b3373ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff161415151561046157600080fd5b5b600073ffffffffffffffffffffffffffffffffffffffff16600160008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060010160019054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1614151561059f57600160008473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060010160019054906101000a900473ffffffffffffffffffffffffffffffffffffffff1692503373ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff161415151561059a57600080fd5b610462565b60018260010160006101000a81548160ff021916908315150217905550828260010160016101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550600160008473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002090508060010160009054906101000a900460ff16156106925781600001546002826002015481548110151561066f57fe5b9060005260206000209060020201600101600082825401925050819055506106a9565b816000015481600001600082825401925050819055505b505050565b6000806000809150600090505b60028054905081101561072457816002828154811015156106d857fe5b9060005260206000209060020201600101541115610717576002818154811015156106ff57fe5b90600052602060002090600202016001015491508092505b80806001019150506106bb565b505090565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff161480156107d25750600160008273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060010160009054906101000a900460ff16155b801561082057506000600160008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060000154145b151561082b57600080fd5b60018060008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000206000018190555050565b60016020528060005260406000206000915090508060000154908060010160009054906101000a900460ff16908060010160019054906101000a900473ffffffffffffffffffffffffffffffffffffffff16908060020154905084565b600060026108de6106ae565b8154811015156108ea57fe5b9060005260206000209060020201600001549050905600a165627a7a723058209216e84efeb17007ba61a1573380cb306de0e38c64eb02e0f9362367121816080029"
				// encoded bytes for ["a", "b"]
				// It consists of four elements which take byte32 each:
				// - 0x20 the location of data
				// - 0x2  the length of array
				// - 0x61 left-aligned 'a'
				// - 0x62 left-aligned 'b'
				constructorArgs = "0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000261000000000000000000000000000000000000000000000000000000000000006200000000000000000000000000000000000000000000000000000000000000"

				// Method hash
				giveRightToVote = "9e7b8d61"
				proposals       = "013cf08b"
				vote            = "0121b93f"
				winnerName      = "e2ba53f0"
				voters          = "a3ec138d"

				user1Cert = `-----BEGIN CERTIFICATE-----
MIICGTCCAcCgAwIBAgIRAOdmptMzz5y0A9GOgFLxRNcwCgYIKoZIzj0EAwIwczEL
MAkGA1UEBhMCVVMxEzARBgNVBAgTCkNhbGlmb3JuaWExFjAUBgNVBAcTDVNhbiBG
cmFuY2lzY28xGTAXBgNVBAoTEG9yZzEuZXhhbXBsZS5jb20xHDAaBgNVBAMTE2Nh
Lm9yZzEuZXhhbXBsZS5jb20wHhcNMTgwMjEyMDY0MDMyWhcNMjgwMjEwMDY0MDMy
WjBbMQswCQYDVQQGEwJVUzETMBEGA1UECBMKQ2FsaWZvcm5pYTEWMBQGA1UEBxMN
U2FuIEZyYW5jaXNjbzEfMB0GA1UEAwwWVXNlcjFAb3JnMS5leGFtcGxlLmNvbTBZ
MBMGByqGSM49AgEGCCqGSM49AwEHA0IABEwsU2N6Kqrtl73S7+7/nD/LTfDFVWO4
q3MTtbckd6MH2zTUj9idLoaQ5VNGJVTRRPs+O6bxlvl0Mitu1rcXFoyjTTBLMA4G
A1UdDwEB/wQEAwIHgDAMBgNVHRMBAf8EAjAAMCsGA1UdIwQkMCKAIKtXuAgSGNzS
0Yz91W08FSieahwkOU7pWJvh86pkNuxSMAoGCCqGSM49BAMCA0cAMEQCIDOGUUvv
SgCqSQONblgBtkKuKgN36VgX+jLhZbaqMNAtAiBXiAHbgYdu3UHBVJwdTYxuFTWJ
Vc4foA7mruwjI8sEng==
-----END CERTIFICATE-----`

				user2Cert = `-----BEGIN CERTIFICATE-----
MIICGDCCAb+gAwIBAgIQMhSPvpu4KGobIvRGEGnZojAKBggqhkjOPQQDAjBzMQsw
CQYDVQQGEwJVUzETMBEGA1UECBMKQ2FsaWZvcm5pYTEWMBQGA1UEBxMNU2FuIEZy
YW5jaXNjbzEZMBcGA1UEChMQb3JnMi5leGFtcGxlLmNvbTEcMBoGA1UEAxMTY2Eu
b3JnMi5leGFtcGxlLmNvbTAeFw0xODAyMTIwNjQwMzJaFw0yODAyMTAwNjQwMzJa
MFsxCzAJBgNVBAYTAlVTMRMwEQYDVQQIEwpDYWxpZm9ybmlhMRYwFAYDVQQHEw1T
YW4gRnJhbmNpc2NvMR8wHQYDVQQDDBZVc2VyMUBvcmcyLmV4YW1wbGUuY29tMFkw
EwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE6mVSTODH+CUZk7/rU+MtycF610ifj0pT
gXGYgJXLLcWbGAC1/ADE8rgq+zihgStD9rnwk0XMitXvqYbIhR0EEqNNMEswDgYD
VR0PAQH/BAQDAgeAMAwGA1UdEwEB/wQCMAAwKwYDVR0jBCQwIoAg1NNSEgEmJaVF
hk5bEaOs6HQS2PFt/VMeXrBUwIGDSogwCgYIKoZIzj0EAwIDRwAwRAIgY6k7AARJ
yJINhf9ub8QcQiMnrTgD4kmhjh5ey8E7fVACIA/i396+beIk0T0c8loygaCiftyG
H8GZeN2ifTyJzzGo
-----END CERTIFICATE-----`

				user1 = marshalCreator("TestOrg", []byte(user1Cert))
				user2 = marshalCreator("TestOrg", []byte(user2Cert))

				deployCode      []byte
				contractAddress account.Address
			)

			BeforeEach(func() {
				deployCode = []byte(contractByteCode + constructorArgs)

				// zero address, and deploy code is contract creation
				stub.GetArgsReturns([][]byte{[]byte(account.ZeroAddress.String()), deployCode})
				res := evmcc.Invoke(stub)
				Expect(res.Status).To(Equal(int32(shim.OK)))

				// Last PutState Call is to store contract runtime bytecode
				key, value := stub.PutStateArgsForCall(stub.PutStateCallCount() - 1)
				Expect(strings.ToLower(key)).To(Equal(strings.ToLower(string(res.Payload))))
				Expect(hex.EncodeToString(value)).To(Equal(runtimeByteCode))

				var err error
				contractAddress, err = account.AddressFromHexString(string(res.Payload))
				Expect(err).ToNot(HaveOccurred())
			})

			It("is able to properly initialize proposals through constructor", func() {
				//invoke proposals(x) to see if constructor args are properly stored
				stub.GetArgsReturns([][]byte{[]byte(contractAddress.String()), []byte(proposals + "0000000000000000000000000000000000000000000000000000000000000000")})
				res := evmcc.Invoke(stub)
				Expect(res.Status).To(Equal(int32(shim.OK)))
				Expect(hex.EncodeToString(res.Payload)).To(Equal("61000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"))

				stub.GetArgsReturns([][]byte{[]byte(contractAddress.String()), []byte(proposals + "0000000000000000000000000000000000000000000000000000000000000001")})
				res = evmcc.Invoke(stub)
				Expect(res.Status).To(Equal(int32(shim.OK)))
				Expect(hex.EncodeToString(res.Payload)).To(Equal("62000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"))
			})

			Context("when user1 is given right to vote", func() {
				var baseCallCount int

				BeforeEach(func() {
					user1Addr, err := identityToAddr([]byte(user1Cert))
					Expect(err).ToNot(HaveOccurred())
					stub.GetArgsReturns([][]byte{[]byte(contractAddress.String()), []byte(giveRightToVote + hex.EncodeToString(user1Addr.Word256().Bytes()))})
					res := evmcc.Invoke(stub)
					Expect(res.Status).To(Equal(int32(shim.OK)))

					baseCallCount = stub.PutStateCallCount()
				})

				Context("when user1 votes for proposal 'a'", func() {
					BeforeEach(func() {
						stub.GetArgsReturns([][]byte{[]byte(contractAddress.String()), []byte(vote + "0000000000000000000000000000000000000000000000000000000000000000")})
						stub.GetCreatorReturns(user1, nil)
						res := evmcc.Invoke(stub)
						Expect(res.Status).To(Equal(int32(shim.OK)))
						Expect(stub.PutStateCallCount()).To(Equal(baseCallCount+3), "`vote` should perform 3 writes: sender.voted, sender.vote, voteCount")
					})

					It("sets the variables of voter 1 (user1) properly", func() {
						user1addr, err := identityToAddr([]byte(user1Cert))
						Expect(err).ToNot(HaveOccurred())
						stub.GetArgsReturns([][]byte{[]byte(contractAddress.String()), []byte(voters + hex.EncodeToString(user1addr.Word256().Bytes()))})
						res := evmcc.Invoke(stub)
						Expect(res.Status).To(Equal(int32(shim.OK)))
						Expect(hex.EncodeToString(res.Payload)).To(Equal("0000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"))
					})

					It("increments vote count of proposal 'a'", func() {
						stub.GetArgsReturns([][]byte{[]byte(contractAddress.String()), []byte(proposals + "0000000000000000000000000000000000000000000000000000000000000000")})
						res := evmcc.Invoke(stub)
						Expect(res.Status).To(Equal(int32(shim.OK)))
						Expect(hex.EncodeToString(res.Payload)).To(Equal("61000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001"))
					})

					It("should make proposal 'a' winner", func() {
						stub.GetArgsReturns([][]byte{[]byte(contractAddress.String()), []byte(winnerName)})
						res := evmcc.Invoke(stub)
						Expect(res.Status).To(Equal(int32(shim.OK)))
						Expect(hex.EncodeToString(res.Payload)).To(Equal("6100000000000000000000000000000000000000000000000000000000000000"))
					})
				})

				Context("when user2 votes for proposal 'a'", func() {
					BeforeEach(func() {
						stub.GetArgsReturns([][]byte{[]byte(contractAddress.String()), []byte(vote + "0000000000000000000000000000000000000000000000000000000000000000")})
						stub.GetCreatorReturns(user2, nil)
						res := evmcc.Invoke(stub)
						baseCallCount = stub.PutStateCallCount()
						Expect(res.Status).To(Equal(int32(shim.OK)))
						Expect(stub.PutStateCallCount()).To(Equal(baseCallCount), "require(!sender.voted) should fail, therefore NO write should be performed")
					})

					It("does not increment vote count of proposal 'a'", func() {
						stub.GetArgsReturns([][]byte{[]byte(contractAddress.String()), []byte(proposals + "0000000000000000000000000000000000000000000000000000000000000000")})
						res := evmcc.Invoke(stub)
						Expect(stub.PutStateCallCount()).To(Equal(baseCallCount), "query should not write to ledger")
						Expect(res.Status).To(Equal(int32(shim.OK)))
						Expect(hex.EncodeToString(res.Payload)).To(Equal("61000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"))
					})
				})
			})
		})

		Context("when a smart contract has events", func() {
			var (
				userCert = `-----BEGIN CERTIFICATE-----
MIICGTCCAcCgAwIBAgIRAOdmptMzz5y0A9GOgFLxRNcwCgYIKoZIzj0EAwIwczEL
MAkGA1UEBhMCVVMxEzARBgNVBAgTCkNhbGlmb3JuaWExFjAUBgNVBAcTDVNhbiBG
cmFuY2lzY28xGTAXBgNVBAoTEG9yZzEuZXhhbXBsZS5jb20xHDAaBgNVBAMTE2Nh
Lm9yZzEuZXhhbXBsZS5jb20wHhcNMTgwMjEyMDY0MDMyWhcNMjgwMjEwMDY0MDMy
WjBbMQswCQYDVQQGEwJVUzETMBEGA1UECBMKQ2FsaWZvcm5pYTEWMBQGA1UEBxMN
U2FuIEZyYW5jaXNjbzEfMB0GA1UEAwwWVXNlcjFAb3JnMS5leGFtcGxlLmNvbTBZ
MBMGByqGSM49AgEGCCqGSM49AwEHA0IABEwsU2N6Kqrtl73S7+7/nD/LTfDFVWO4
q3MTtbckd6MH2zTUj9idLoaQ5VNGJVTRRPs+O6bxlvl0Mitu1rcXFoyjTTBLMA4G
A1UdDwEB/wQEAwIHgDAMBgNVHRMBAf8EAjAAMCsGA1UdIwQkMCKAIKtXuAgSGNzS
0Yz91W08FSieahwkOU7pWJvh86pkNuxSMAoGCCqGSM49BAMCA0cAMEQCIDOGUUvv
SgCqSQONblgBtkKuKgN36VgX+jLhZbaqMNAtAiBXiAHbgYdu3UHBVJwdTYxuFTWJ
Vc4foA7mruwjI8sEng==
-----END CERTIFICATE-----`

				creator = marshalCreator("TestOrg", []byte(userCert))

				/*pragma solidity ^0.4.0;
				  contract Instructor {
				    bytes32 fName;
				    uint age;
				    uint salary;
				    event Setter(bytes32 indexed name, uint age, uint salary);
				    function setInstructor(bytes32 _fName, uint _age, uint _salary) public {
				      fName = _fName;
				      age = _age;
				      salary = _salary;
				      emit Setter(_fName, _age, _salary);
				    }
				    function getInstructor() public constant returns (bytes32, uint, uint) {
				      return (fName, age, salary);
				    }
				  }*/

				deployCode      = []byte("608060405234801561001057600080fd5b50610122806100206000396000f30060806040526004361060485763ffffffff7c010000000000000000000000000000000000000000000000000000000060003504166331fb1dff8114604d5780633c1b81a514606a575b600080fd5b348015605857600080fd5b506068600435602435604435609a565b005b348015607557600080fd5b50607c60e8565b60408051938452602084019290925282820152519081900360600190f35b6000839055600182905560028190556040805183815260208101839052815185927fe920a6ca2d94687457e136223552305dbabca6f28cf9c65d18efc2193a2369b0928290030190a2505050565b6000546001546002549091925600a165627a7a723058201f3b3871bfe7762e6fb776ed8b5d5533e07178b576c630cf89a7e63a7b54b57b0029")
				contractAddress account.Address
				SET             = "31fb1dff" //"setInstructor(bytes32,uint256,uint256)"
				GET             = "3c1b81a5" //"getInstructor()"
				msg             events.EventDataLog
				messagePayloads []events.EventDataLog
			)

			BeforeEach(func() {
				// Set contract creator
				stub.GetCreatorReturns(creator, nil)

				// zero address, and deploy code is contract creation
				stub.GetArgsReturns([][]byte{[]byte(account.ZeroAddress.String()), deployCode})
				res := evmcc.Invoke(stub)
				Expect(res.Status).To(Equal(int32(shim.OK)))
				Expect(stub.PutStateCallCount()).To(Equal(2))

				var err error
				contractAddress, err = account.AddressFromHexString(string(res.Payload))
				Expect(err).ToNot(HaveOccurred())

				topics := []binary.Word256{}

				//First topic refers to the Event: sha3('Setter(bytes32, uint256, uint256)')
				topic, err := hex.DecodeString("e920a6ca2d94687457e136223552305dbabca6f28cf9c65d18efc2193a2369b0")
				Expect(err).ToNot(HaveOccurred())
				topics = append(topics, binary.RightPadWord256(topic))

				// Second topic is the value of the first indexed param of the event. In this case it is the name in bytes. The value is "Sam" in hex
				topic, err = hex.DecodeString("53616d0000000000000000000000000000000000000000000000000000000000")
				Expect(err).ToNot(HaveOccurred())
				topics = append(topics, binary.RightPadWord256(topic))

				//Data contains the non indexed elements of the event concatenated together. Remaining values are age and salary which are hex encoded
				// 0x0000000000000000000000000000000000000000000000000000000000000019 is 25 (age)
				// 0x0000000000000000000000000000000000000000000000000000000000007530 is 30000 (salary)
				data, err := hex.DecodeString("00000000000000000000000000000000000000000000000000000000000000190000000000000000000000000000000000000000000000000000000000007530")

				Expect(err).ToNot(HaveOccurred())

				msg = events.EventDataLog{
					Address: contractAddress,
					Topics:  topics,
					Data:    data,
					Height:  0,
				}

				messagePayloads = []events.EventDataLog{msg}
			})

			Context("if the method called emits event(s)", func() {
				It("sets the chaincode event", func() {
					// The 3 values following SET are the arguments to SET. All 3 are hex encoded
					// First arg is the hex encoded name "Sam"
					// Second arg is the hex encoded value 25
					// Third arg is the hex encoded value 30
					stub.GetArgsReturns([][]byte{[]byte(contractAddress.String()), []byte(SET + "53616d0000000000000000000000000000000000000000000000000000000000" + "0000000000000000000000000000000000000000000000000000000000000019" + "0000000000000000000000000000000000000000000000000000000000007530")})

					res := evmcc.Invoke(stub)
					Expect(res.Status).To(Equal(int32(shim.OK)))

					expectedPayload, ok := json.Marshal(messagePayloads)
					Expect(ok).ToNot(HaveOccurred())

					Expect(stub.SetEventCallCount()).To(Equal(1))
					setEventName, setEventPayload := stub.SetEventArgsForCall(0)
					Expect(setEventName).To(Equal(SET))
					Expect(setEventPayload).To(Equal([]byte(expectedPayload)))

					var unmarshaledPayloads []events.EventDataLog
					err := json.Unmarshal(setEventPayload, &unmarshaledPayloads)
					Expect(err).ToNot(HaveOccurred())
					Expect(unmarshaledPayloads).To(Equal(messagePayloads))
				})
			})

			Context("if the method called does not emit any events", func() {
				It("doesn't set any chaincode event", func() {
					stub.GetArgsReturns([][]byte{[]byte(contractAddress.String()), []byte(GET)})
					res := evmcc.Invoke(stub)
					Expect(res.Status).To(Equal(int32(shim.OK)))
					Expect(stub.SetEventCallCount()).To(Equal(0))
				})
			})
		})
	})
})

// TODO: This is copied from evmcc. Consider moving this to an util pkg
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
