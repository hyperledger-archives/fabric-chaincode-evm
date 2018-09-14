/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabproxy_test

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/hyperledger/fabric-chaincode-evm/fabproxy"
	"github.com/hyperledger/fabric-chaincode-evm/mocks"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/peer"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var evmcc = "evmcc"
var _ = Describe("Ethservice", func() {
	var (
		ethservice fabproxy.EthService

		mockChClient     *mocks.MockChannelClient
		mockLedgerClient *mocks.MockLedgerClient
		channelID        string
	)

	BeforeEach(func() {
		mockChClient = &mocks.MockChannelClient{}
		mockLedgerClient = &mocks.MockLedgerClient{}
		channelID = "test-channel"

		ethservice = fabproxy.NewEthService(mockChClient, mockLedgerClient, channelID, evmcc)
	})

	Describe("GetCode", func() {
		var (
			sampleCode    []byte
			sampleAddress string
		)

		BeforeEach(func() {
			sampleCode = []byte("sample-code")
			mockChClient.QueryReturns(channel.Response{
				Payload: sampleCode,
			}, nil)

			sampleAddress = "1234567123"
		})

		It("returns the code associated to that address", func() {
			var reply string

			err := ethservice.GetCode(&http.Request{}, &sampleAddress, &reply)
			Expect(err).ToNot(HaveOccurred())

			Expect(mockChClient.QueryCallCount()).To(Equal(1))
			chReq, reqOpts := mockChClient.QueryArgsForCall(0)
			Expect(chReq).To(Equal(channel.Request{
				ChaincodeID: evmcc,
				Fcn:         "getCode",
				Args:        [][]byte{[]byte(sampleAddress)},
			}))

			Expect(reqOpts).To(HaveLen(0))

			Expect(reply).To(Equal(string(sampleCode)))
		})

		Context("when the address has `0x` prefix", func() {
			BeforeEach(func() {
				sampleAddress = "0x123456"
			})
			It("returns the code associated with that address", func() {
				var reply string

				err := ethservice.GetCode(&http.Request{}, &sampleAddress, &reply)
				Expect(err).ToNot(HaveOccurred())

				Expect(mockChClient.QueryCallCount()).To(Equal(1))
				chReq, reqOpts := mockChClient.QueryArgsForCall(0)
				Expect(chReq).To(Equal(channel.Request{
					ChaincodeID: evmcc,
					Fcn:         "getCode",
					Args:        [][]byte{[]byte(sampleAddress[2:])},
				}))

				Expect(reqOpts).To(HaveLen(0))

				Expect(reply).To(Equal(string(sampleCode)))
			})
		})

		Context("when the ledger errors when processing a query", func() {
			BeforeEach(func() {
				mockChClient.QueryReturns(channel.Response{}, errors.New("boom!"))
			})

			It("returns a corresponding error", func() {
				var reply string

				err := ethservice.GetCode(&http.Request{}, &sampleAddress, &reply)
				Expect(err).To(MatchError(ContainSubstring("Failed to query the ledger")))

				Expect(reply).To(BeEmpty())
			})
		})
	})

	Describe("Call", func() {
		var (
			encodedResponse []byte
			sampleArgs      *fabproxy.EthArgs
		)

		BeforeEach(func() {
			sampleResponse := []byte("sample response")
			encodedResponse = make([]byte, hex.EncodedLen(len(sampleResponse)))
			hex.Encode(encodedResponse, sampleResponse)
			mockChClient.QueryReturns(channel.Response{
				Payload: sampleResponse,
			}, nil)

			sampleArgs = &fabproxy.EthArgs{
				To:   "1234567123",
				Data: "sample-data",
			}
		})

		It("returns the value of the simulation of executing a smart contract with a `0x` prefix", func() {

			var reply string

			err := ethservice.Call(&http.Request{}, sampleArgs, &reply)
			Expect(err).ToNot(HaveOccurred())

			Expect(mockChClient.QueryCallCount()).To(Equal(1))
			chReq, reqOpts := mockChClient.QueryArgsForCall(0)
			Expect(chReq).To(Equal(channel.Request{
				ChaincodeID: evmcc,
				Fcn:         sampleArgs.To,
				Args:        [][]byte{[]byte(sampleArgs.Data)},
			}))

			Expect(reqOpts).To(HaveLen(0))

			Expect(reply).To(Equal("0x" + string(encodedResponse)))
		})

		Context("when the ledger errors when processing a query", func() {
			BeforeEach(func() {
				mockChClient.QueryReturns(channel.Response{}, errors.New("boom!"))
			})

			It("returns a corresponding error", func() {
				var reply string

				err := ethservice.Call(&http.Request{}, &fabproxy.EthArgs{}, &reply)
				Expect(err).To(MatchError(ContainSubstring("Failed to query the ledger")))
				Expect(reply).To(BeEmpty())
			})
		})

		Context("when the address has a `0x` prefix", func() {
			BeforeEach(func() {
				sampleArgs.To = "0x" + sampleArgs.To
			})
			It("strips the prefix from the query", func() {
				var reply string

				err := ethservice.Call(&http.Request{}, sampleArgs, &reply)
				Expect(err).ToNot(HaveOccurred())

				Expect(mockChClient.QueryCallCount()).To(Equal(1))
				chReq, reqOpts := mockChClient.QueryArgsForCall(0)
				Expect(chReq).To(Equal(channel.Request{
					ChaincodeID: evmcc,
					Fcn:         sampleArgs.To[2:],
					Args:        [][]byte{[]byte(sampleArgs.Data)},
				}))

				Expect(reqOpts).To(HaveLen(0))

				Expect(reply).To(Equal("0x" + string(encodedResponse)))
			})
		})

		Context("when the data has a `0x` prefix", func() {
			BeforeEach(func() {
				sampleArgs.Data = "0x" + sampleArgs.Data
			})

			It("strips the prefix from the query", func() {
				var reply string

				err := ethservice.Call(&http.Request{}, sampleArgs, &reply)
				Expect(err).ToNot(HaveOccurred())

				Expect(mockChClient.QueryCallCount()).To(Equal(1))
				chReq, reqOpts := mockChClient.QueryArgsForCall(0)
				Expect(chReq).To(Equal(channel.Request{
					ChaincodeID: evmcc,
					Fcn:         sampleArgs.To,
					Args:        [][]byte{[]byte(sampleArgs.Data[2:])},
				}))

				Expect(reqOpts).To(HaveLen(0))

				Expect(reply).To(Equal("0x" + string(encodedResponse)))
			})
		})
	})

	Describe("SendTransaction", func() {
		var (
			sampleResponse channel.Response
			sampleArgs     *fabproxy.EthArgs
		)

		BeforeEach(func() {
			sampleResponse = channel.Response{
				Payload:       []byte("sample-response"),
				TransactionID: "1",
			}
			mockChClient.ExecuteReturns(sampleResponse, nil)

			sampleArgs = &fabproxy.EthArgs{
				To:   "1234567123",
				Data: "sample-data",
			}
		})

		It("returns the transaction id", func() {
			var reply string
			err := ethservice.SendTransaction(&http.Request{}, sampleArgs, &reply)
			Expect(err).ToNot(HaveOccurred())

			Expect(mockChClient.ExecuteCallCount()).To(Equal(1))
			chReq, reqOpts := mockChClient.ExecuteArgsForCall(0)
			Expect(chReq).To(Equal(channel.Request{
				ChaincodeID: evmcc,
				Fcn:         sampleArgs.To,
				Args:        [][]byte{[]byte(sampleArgs.Data)},
			}))

			Expect(reqOpts).To(HaveLen(0))

			Expect(reply).To(Equal(string(sampleResponse.TransactionID)))
		})

		Context("when the transaction is a contract deployment", func() {
			BeforeEach(func() {
				sampleArgs.To = ""
			})

			It("returns the transaction id", func() {
				var reply string
				err := ethservice.SendTransaction(&http.Request{}, sampleArgs, &reply)
				Expect(err).ToNot(HaveOccurred())

				zeroAddress := hex.EncodeToString(fabproxy.ZeroAddress)
				Expect(mockChClient.ExecuteCallCount()).To(Equal(1))
				chReq, reqOpts := mockChClient.ExecuteArgsForCall(0)
				Expect(chReq).To(Equal(channel.Request{
					ChaincodeID: evmcc,
					Fcn:         zeroAddress,
					Args:        [][]byte{[]byte(sampleArgs.Data)},
				}))

				Expect(reqOpts).To(HaveLen(0))

				Expect(reply).To(Equal(string(sampleResponse.TransactionID)))
			})
		})

		Context("when the ledger errors when processing a query", func() {
			BeforeEach(func() {
				mockChClient.ExecuteReturns(channel.Response{}, errors.New("boom!"))
			})

			It("returns a corresponding error", func() {
				var reply string

				err := ethservice.SendTransaction(&http.Request{}, &fabproxy.EthArgs{}, &reply)
				Expect(err).To(MatchError(ContainSubstring("Failed to execute transaction")))
				Expect(reply).To(BeEmpty())
			})
		})

		Context("when the address has a `0x` prefix", func() {
			BeforeEach(func() {
				sampleArgs.To = "0x" + sampleArgs.To
			})

			It("strips the prefix before calling the evmscc", func() {
				var reply string
				err := ethservice.SendTransaction(&http.Request{}, sampleArgs, &reply)
				Expect(err).ToNot(HaveOccurred())

				Expect(mockChClient.ExecuteCallCount()).To(Equal(1))
				chReq, reqOpts := mockChClient.ExecuteArgsForCall(0)
				Expect(chReq).To(Equal(channel.Request{
					ChaincodeID: evmcc,
					Fcn:         sampleArgs.To[2:],
					Args:        [][]byte{[]byte(sampleArgs.Data)},
				}))

				Expect(reqOpts).To(HaveLen(0))

				Expect(reply).To(Equal(string(sampleResponse.TransactionID)))
			})
		})

		Context("when the data has a `0x` prefix", func() {
			BeforeEach(func() {
				sampleArgs.Data = "0x" + sampleArgs.Data
			})

			It("strips the prefix before calling the evmscc", func() {
				var reply string
				err := ethservice.SendTransaction(&http.Request{}, sampleArgs, &reply)
				Expect(err).ToNot(HaveOccurred())

				Expect(mockChClient.ExecuteCallCount()).To(Equal(1))
				chReq, reqOpts := mockChClient.ExecuteArgsForCall(0)
				Expect(chReq).To(Equal(channel.Request{
					ChaincodeID: evmcc,
					Fcn:         sampleArgs.To,
					Args:        [][]byte{[]byte(sampleArgs.Data[2:])},
				}))

				Expect(reqOpts).To(HaveLen(0))

				Expect(reply).To(Equal(string(sampleResponse.TransactionID)))
			})
		})
	})

	Describe("GetTransactionReceipt", func() {
		var (
			sampleTransaction   *peer.ProcessedTransaction
			sampleBlock         *common.Block
			sampleTransactionID string
		)

		BeforeEach(func() {
			var err error
			sampleTransaction, err = GetSampleTransaction([][]byte{[]byte("82373458"), []byte("sample arg 2")}, []byte("sample-response"), "1234")
			Expect(err).ToNot(HaveOccurred())

			sampleBlock, err = GetSampleBlock(31, []byte("12345abcd"))
			Expect(err).ToNot(HaveOccurred())

			mockLedgerClient.QueryBlockByTxIDReturns(sampleBlock, nil)
			mockLedgerClient.QueryTransactionReturns(sampleTransaction, nil)
			sampleTransactionID = "1234567123"
		})

		It("returns the transaction receipt associated to that transaction address", func() {
			var reply fabproxy.TxReceipt

			err := ethservice.GetTransactionReceipt(&http.Request{}, &sampleTransactionID, &reply)
			Expect(err).ToNot(HaveOccurred())

			Expect(mockLedgerClient.QueryTransactionCallCount()).To(Equal(1))
			txID, reqOpts := mockLedgerClient.QueryTransactionArgsForCall(0)
			Expect(txID).To(Equal(fab.TransactionID(sampleTransactionID)))
			Expect(reqOpts).To(HaveLen(0))

			Expect(mockLedgerClient.QueryBlockByTxIDCallCount()).To(Equal(1))
			txID, reqOpts = mockLedgerClient.QueryBlockByTxIDArgsForCall(0)
			Expect(txID).To(Equal(fab.TransactionID(sampleTransactionID)))
			Expect(reqOpts).To(HaveLen(0))

			Expect(reply).To(Equal(fabproxy.TxReceipt{
				TransactionHash:   sampleTransactionID,
				BlockHash:         hex.EncodeToString(sampleBlock.GetHeader().GetDataHash()),
				BlockNumber:       "0x1f",
				GasUsed:           0,
				CumulativeGasUsed: 0,
			}))
		})

		Context("when the transaction is creation of a smart contract", func() {
			var contractAddress []byte
			BeforeEach(func() {
				contractAddress = []byte("0x123456789abcdef1234")
				zeroAddress := make([]byte, hex.EncodedLen(len(fabproxy.ZeroAddress)))
				hex.Encode(zeroAddress, fabproxy.ZeroAddress)

				tx, err := GetSampleTransaction([][]byte{zeroAddress, []byte("sample arg 2")}, contractAddress, "1234")
				*sampleTransaction = *tx
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns the contract address in the transaction receipt", func() {
				var reply fabproxy.TxReceipt

				err := ethservice.GetTransactionReceipt(&http.Request{}, &sampleTransactionID, &reply)
				Expect(err).ToNot(HaveOccurred())

				Expect(mockLedgerClient.QueryTransactionCallCount()).To(Equal(1))
				txID, reqOpts := mockLedgerClient.QueryTransactionArgsForCall(0)
				Expect(txID).To(Equal(fab.TransactionID(sampleTransactionID)))
				Expect(reqOpts).To(HaveLen(0))

				Expect(mockLedgerClient.QueryBlockByTxIDCallCount()).To(Equal(1))
				txID, reqOpts = mockLedgerClient.QueryBlockByTxIDArgsForCall(0)
				Expect(txID).To(Equal(fab.TransactionID(sampleTransactionID)))
				Expect(reqOpts).To(HaveLen(0))

				Expect(reply).To(Equal(fabproxy.TxReceipt{
					TransactionHash:   sampleTransactionID,
					BlockHash:         hex.EncodeToString(sampleBlock.GetHeader().GetDataHash()),
					BlockNumber:       "0x1f",
					ContractAddress:   string(contractAddress),
					GasUsed:           0,
					CumulativeGasUsed: 0,
				}))
			})

			Context("when transaction ID has `0x` prefix", func() {
				BeforeEach(func() {
					sampleTransactionID = "0x" + sampleTransactionID
				})
				It("strips the prefix before querying the ledger", func() {
					var reply fabproxy.TxReceipt

					err := ethservice.GetTransactionReceipt(&http.Request{}, &sampleTransactionID, &reply)
					Expect(err).ToNot(HaveOccurred())

					Expect(mockLedgerClient.QueryTransactionCallCount()).To(Equal(1))
					txID, reqOpts := mockLedgerClient.QueryTransactionArgsForCall(0)
					Expect(txID).To(Equal(fab.TransactionID(sampleTransactionID[2:])))
					Expect(reqOpts).To(HaveLen(0))

					Expect(mockLedgerClient.QueryBlockByTxIDCallCount()).To(Equal(1))
					txID, reqOpts = mockLedgerClient.QueryBlockByTxIDArgsForCall(0)
					Expect(txID).To(Equal(fab.TransactionID(sampleTransactionID[2:])))
					Expect(reqOpts).To(HaveLen(0))

					Expect(reply).To(Equal(fabproxy.TxReceipt{
						TransactionHash:   sampleTransactionID,
						BlockHash:         hex.EncodeToString(sampleBlock.GetHeader().GetDataHash()),
						BlockNumber:       "0x1f",
						ContractAddress:   string(contractAddress),
						GasUsed:           0,
						CumulativeGasUsed: 0,
					}))
				})
			})
		})

		Context("when the ledger errors when processing a transaction query for the transaction", func() {
			BeforeEach(func() {
				mockLedgerClient.QueryTransactionReturns(nil, errors.New("boom!"))
			})

			It("returns a corresponding error", func() {
				var reply fabproxy.TxReceipt

				err := ethservice.GetTransactionReceipt(&http.Request{}, &sampleTransactionID, &reply)
				Expect(err).To(MatchError(ContainSubstring("Failed to query the ledger")))
				Expect(reply).To(BeZero())
			})
		})

		Context("when the ledger errors when processing a query for the block", func() {
			BeforeEach(func() {
				mockLedgerClient.QueryBlockByTxIDReturns(nil, errors.New("boom!"))
			})

			It("returns a corresponding error", func() {
				var reply fabproxy.TxReceipt

				err := ethservice.GetTransactionReceipt(&http.Request{}, &sampleTransactionID, &reply)
				Expect(err).To(MatchError(ContainSubstring("Failed to query the ledger")))
				Expect(reply).To(BeZero())
			})
		})
	})

	Describe("Accounts", func() {
		var (
			sampleAccount string
			arg           string
		)

		BeforeEach(func() {
			sampleAccount = "123456ABCD"
			mockChClient.QueryReturns(channel.Response{
				Payload: []byte(sampleAccount),
			}, nil)

		})

		It("requests the user address from the evmscc based on the user cert", func() {
			var reply []string

			err := ethservice.Accounts(&http.Request{}, &arg, &reply)
			Expect(err).ToNot(HaveOccurred())

			Expect(mockChClient.QueryCallCount()).To(Equal(1))
			chReq, reqOpts := mockChClient.QueryArgsForCall(0)
			Expect(chReq).To(Equal(channel.Request{
				ChaincodeID: evmcc,
				Fcn:         "account",
				Args:        [][]byte{},
			}))

			Expect(reqOpts).To(HaveLen(0))
			expectedResponse := []string{"0x" + strings.ToLower(sampleAccount)}
			Expect(reply).To(Equal(expectedResponse))
		})

		Context("when the ledger errors when processing a query", func() {
			BeforeEach(func() {
				mockChClient.QueryReturns(channel.Response{}, errors.New("boom!"))
			})

			It("returns a corresponding error", func() {
				var reply []string
				err := ethservice.Accounts(&http.Request{}, &arg, &reply)
				Expect(err).To(MatchError(ContainSubstring("Failed to query the ledger")))
				Expect(reply).To(BeEmpty())
			})
		})
	})

	Describe("EstimateGas", func() {
		It("always returns zero", func() {
			var reply string
			err := ethservice.EstimateGas(&http.Request{}, &fabproxy.EthArgs{}, &reply)
			Expect(err).ToNot(HaveOccurred())
			Expect(reply).To(Equal("0x0"))
		})
	})

	Describe("GetBalance", func() {
		It("always returns zero", func() {
			arg := make([]string, 2)
			var reply string
			err := ethservice.GetBalance(&http.Request{}, &arg, &reply)
			Expect(err).ToNot(HaveOccurred())
			Expect(reply).To(Equal("0x0"))
		})
	})

	Describe("GetBlockByNumber", func() {
		Context("when provided with bad parameters", func() {
			var reply fabproxy.Block

			It("returns an error when arg length is not 2", func() {
				var arg []interface{}
				err := ethservice.GetBlockByNumber(&http.Request{}, &arg, &reply)
				Expect(err).To(HaveOccurred())
			})

			It("returns an error when the first arg is not a string", func() {
				arg := make([]interface{}, 2)
				arg[0] = false
				err := ethservice.GetBlockByNumber(&http.Request{}, &arg, &reply)
				Expect(err).To(HaveOccurred())
			})
			It("returns an error when first arg is not a named block or numbered block", func() {
				arg := make([]interface{}, 2)
				arg[0] = "hurf%&"
				err := ethservice.GetBlockByNumber(&http.Request{}, &arg, &reply)
				Expect(err).To(HaveOccurred())
			})

			It("returns an error, when the second arg is not a booleand", func() {
				arg := make([]interface{}, 2)
				arg[0] = "latest"
				arg[1] = "durf"
				err := ethservice.GetBlockByNumber(&http.Request{}, &arg, &reply)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when there are good parameters", func() {
			var (
				reply                fabproxy.Block
				args                 []interface{}
				fullTransactions     bool
				requestedBlockNumber string
			)

			JustBeforeEach(func() {
				args = make([]interface{}, 2)
				args[0] = requestedBlockNumber
				args[1] = fullTransactions
			})

			Context("when asking for partial transactions", func() {
				BeforeEach(func() {
					fullTransactions = false
				})

				Context("returns an error when querying the ledger info results in an error", func() {
					BeforeEach(func() {
						requestedBlockNumber = "latest"
						mockLedgerClient.QueryInfoReturns(nil, fmt.Errorf("no block info"))
					})

					It("returns a no blockchain info error when requesting a named block", func() {
						err := ethservice.GetBlockByNumber(&http.Request{}, &args, &reply)
						Expect(err).To(MatchError(ContainSubstring("no block info")))
					})
				})

				Context("when querying the ledger for a block results in an error", func() {
					BeforeEach(func() {
						requestedBlockNumber = "0xa"
						mockLedgerClient.QueryBlockReturns(nil, fmt.Errorf("no block"))
					})

					It("returns the error", func() {
						err := ethservice.GetBlockByNumber(&http.Request{}, &args, &reply)
						Expect(err).To(HaveOccurred())
					})

					Context("when querying for a named block", func() {
						BeforeEach(func() {
							requestedBlockNumber = "latest"
							mockLedgerClient.QueryInfoReturns(&fab.BlockchainInfoResponse{BCI: &common.BlockchainInfo{Height: 1}}, nil)
						})

						It("returns an error", func() {
							err := ethservice.GetBlockByNumber(&http.Request{}, &args, &reply)
							Expect(err).To(HaveOccurred())
						})
					})
				})

				It("returns an error when asked for pending blocks", func() {
					args[0] = "pending"
					err := ethservice.GetBlockByNumber(&http.Request{}, &args, &reply)
					Expect(err).To(HaveOccurred())
				})

				Context("when a block is requested by number", func() {
					var uintBlockNumber uint64

					BeforeEach(func() {
						requestedBlockNumber = "abc0"

						var err error
						uintBlockNumber, err = strconv.ParseUint(requestedBlockNumber, 16, 64)
						Expect(err).ToNot(HaveOccurred())
					})

					It("requests a block by number", func() {
						sampleBlock := GetSampleBlockWithTransactions(uintBlockNumber)
						mockLedgerClient.QueryBlockReturns(sampleBlock, nil)

						err := ethservice.GetBlockByNumber(&http.Request{}, &args, &reply)
						Expect(err).ToNot(HaveOccurred())

						Expect(reply.Number).To(Equal("0x"+requestedBlockNumber), "block number")
						Expect(reply.Hash).To(Equal("0x"+hex.EncodeToString(sampleBlock.Header.DataHash)), "block data hash")
						Expect(reply.ParentHash).To(Equal("0x"+hex.EncodeToString(sampleBlock.Header.PreviousHash)), "block parent hash")
						txns := reply.Transactions
						Expect(txns).To(HaveLen(2))
						Expect(txns[0]).To(BeEquivalentTo("0x5678"))
						Expect(txns[1]).To(BeEquivalentTo("0x1234"))
					})
				})

				Context("when the block is requested by name", func() {
					var uintBlockNumber uint64

					BeforeEach(func() {
						requestedBlockNumber = "latest"

						var err error
						uintBlockNumber, err = strconv.ParseUint("abc0", 16, 64)
						Expect(err).ToNot(HaveOccurred())

						mockLedgerClient.QueryInfoReturns(&fab.BlockchainInfoResponse{BCI: &common.BlockchainInfo{Height: uintBlockNumber + 1}}, nil)
					})

					It("returns the block", func() {
						sampleBlock := GetSampleBlockWithTransactions(uintBlockNumber)
						mockLedgerClient.QueryBlockReturns(sampleBlock, nil)

						err := ethservice.GetBlockByNumber(&http.Request{}, &args, &reply)
						Expect(err).ToNot(HaveOccurred())
						Expect(reply.Number).To(Equal("0xabc0"), "block number")
						Expect(reply.Hash).To(Equal("0x"+hex.EncodeToString(sampleBlock.Header.DataHash)), "block data hash")
						Expect(reply.ParentHash).To(Equal("0x"+hex.EncodeToString(sampleBlock.Header.PreviousHash)), "block parent hash")
						txns := reply.Transactions
						Expect(txns).To(HaveLen(2))
						Expect(txns[0]).To(BeEquivalentTo("0x5678"))
						Expect(txns[1]).To(BeEquivalentTo("0x1234"))

					})
				})
			})

			Context("when asking for full transactions", func() {
				BeforeEach(func() {
					fullTransactions = true
				})

				It("returns an unimplemented error", func() {
					err := ethservice.GetBlockByNumber(&http.Request{}, &args, &reply)
					Expect(err).To(HaveOccurred())
				})
			})
		})
	})

	Describe("GetTransactionByHash", func() {
		var reply fabproxy.Transaction

		It("returns an error when given an empty string for transaction hash", func() {
			txID := ""
			err := ethservice.GetTransactionByHash(&http.Request{}, &txID, &reply)
			Expect(err).To(HaveOccurred())
		})

		Context("if the ledger returns an error", func() {
			BeforeEach(func() {
				mockLedgerClient.QueryBlockByTxIDReturns(nil, fmt.Errorf("bad ledger lookup"))
			})
			It("returns an error ", func() {
				txID := "0x1234"
				err := ethservice.GetTransactionByHash(&http.Request{}, &txID, &reply)
				Expect(err).To(HaveOccurred())
			})
		})

		It("gets a transaction", func() {
			txID := "0x1234"
			block := GetSampleBlockWithTransactions(1)
			mockLedgerClient.QueryBlockByTxIDReturns(block, nil)
			err := ethservice.GetTransactionByHash(&http.Request{}, &txID, &reply)
			Expect(err).ToNot(HaveOccurred())
			Expect(reply.Hash).To(Equal(txID), "txn id hash that was passed in")
			Expect(reply.BlockHash).To(Equal("0x"+hex.EncodeToString(block.Header.DataHash)), "block data hash")
			Expect(reply.BlockNumber).To(Equal("0x1"), "blocknumber")
			Expect(reply.TransactionIndex).To(Equal("0x1"), "txn Index")
			Expect(reply.To).To(Equal("0x98765432"))
			Expect(reply.Input).To(Equal("0xsample arg 2"))
		})
	})
})

func GetSampleBlock(blkNumber uint64, blkHash []byte) (*common.Block, error) {
	return &common.Block{
		Header: &common.BlockHeader{Number: blkNumber, DataHash: blkHash},
	}, nil
}

func GetSampleBlockWithTransactions(blockNumber uint64) *common.Block {
	tx, err := GetSampleTransaction([][]byte{[]byte("12345678"), []byte("sample arg 1")}, []byte("sample-response1"), "5678")
	Expect(err).ToNot(HaveOccurred())
	txn1, err := proto.Marshal(tx.TransactionEnvelope)
	Expect(err).ToNot(HaveOccurred())

	tx, err = GetSampleTransaction([][]byte{[]byte("98765432"), []byte("sample arg 2")}, []byte("sample-response2"), "1234")
	txn2, err := proto.Marshal(tx.TransactionEnvelope)
	Expect(err).ToNot(HaveOccurred())

	phash := []byte("abc\x00")
	dhash := []byte("def\xFF")
	return &common.Block{
		Header: &common.BlockHeader{Number: blockNumber,
			PreviousHash: phash,
			DataHash:     dhash},
		Data: &common.BlockData{Data: [][]byte{txn1, txn2}},
	}
}

func GetSampleTransaction(inputArgs [][]byte, txResponse []byte, txId string) (*peer.ProcessedTransaction, error) {

	respPayload := &peer.ChaincodeAction{
		Response: &peer.Response{
			Payload: txResponse,
		},
	}

	ext, err := proto.Marshal(respPayload)
	if err != nil {
		return &peer.ProcessedTransaction{}, err
	}

	pRespPayload := &peer.ProposalResponsePayload{
		Extension: ext,
	}

	ccProposalPayload, err := proto.Marshal(pRespPayload)
	if err != nil {
		return &peer.ProcessedTransaction{}, err
	}

	invokeSpec := &peer.ChaincodeInvocationSpec{
		ChaincodeSpec: &peer.ChaincodeSpec{
			ChaincodeId: &peer.ChaincodeID{
				Name: evmcc,
			},
			Input: &peer.ChaincodeInput{
				Args: inputArgs,
			},
		},
	}

	invokeSpecBytes, err := proto.Marshal(invokeSpec)
	if err != nil {
		return &peer.ProcessedTransaction{}, err
	}

	ccPropPayload, err := proto.Marshal(&peer.ChaincodeProposalPayload{
		Input: invokeSpecBytes,
	})
	if err != nil {
		return &peer.ProcessedTransaction{}, err
	}

	ccPayload := &peer.ChaincodeActionPayload{
		Action: &peer.ChaincodeEndorsedAction{
			ProposalResponsePayload: ccProposalPayload,
		},
		ChaincodeProposalPayload: ccPropPayload,
	}

	actionPayload, err := proto.Marshal(ccPayload)
	if err != nil {
		return &peer.ProcessedTransaction{}, err
	}

	txAction := &peer.TransactionAction{
		Payload: actionPayload,
	}

	txActions := &peer.Transaction{
		Actions: []*peer.TransactionAction{txAction},
	}

	actionsPayload, err := proto.Marshal(txActions)
	if err != nil {
		return &peer.ProcessedTransaction{}, err
	}

	chdr := &common.ChannelHeader{TxId: txId}
	chdrBytes, err := proto.Marshal(chdr)
	if err != nil {
		return &peer.ProcessedTransaction{}, err
	}

	payload := &common.Payload{
		Header: &common.Header{
			ChannelHeader: chdrBytes,
		},
		Data: actionsPayload,
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		return &peer.ProcessedTransaction{}, err
	}

	tx := &peer.ProcessedTransaction{
		TransactionEnvelope: &common.Envelope{
			Payload: payloadBytes,
		},
	}

	return tx, nil
}
