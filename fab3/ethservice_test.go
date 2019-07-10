/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fab3_test

import (
	"crypto/sha256"
	"encoding/asn1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/gogo/protobuf/proto"
	"github.com/hyperledger/burrow/crypto"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/ledger"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/msp"

	"github.com/hyperledger/fabric-chaincode-evm/event"
	"github.com/hyperledger/fabric-chaincode-evm/fab3"
	"github.com/hyperledger/fabric-chaincode-evm/fab3/types"

	fab3_mocks "github.com/hyperledger/fabric-chaincode-evm/mocks/fab3"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	evmcc = "evmcc"
	cert  = `-----BEGIN CERTIFICATE-----
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
	// Address associated with the above cert
	addrFromCert = "0xb3778bcee2b9c349702e5832928730d2aed0ac07"
)
var _ = Describe("Ethservice", func() {
	var (
		ethservice fab3.EthService

		mockChClient     *fab3_mocks.MockChannelClient
		mockLedgerClient *fab3_mocks.MockLedgerClient
		channelID        string
	)
	core := zapcore.NewCore(zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()), zapcore.AddSync(GinkgoWriter), zap.DebugLevel)
	rawLogger := zap.New(core)
	zap.ReplaceGlobals(rawLogger)
	logger := rawLogger.Sugar()

	BeforeEach(func() {
		mockChClient = &fab3_mocks.MockChannelClient{}
		mockLedgerClient = &fab3_mocks.MockLedgerClient{}
		channelID = "test-channel"

		ethservice = fab3.NewEthService(mockChClient, mockLedgerClient, channelID, evmcc, logger)
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
			sampleArgs      *types.EthArgs
		)

		BeforeEach(func() {
			sampleResponse := []byte("sample response")
			encodedResponse = make([]byte, hex.EncodedLen(len(sampleResponse)))
			hex.Encode(encodedResponse, sampleResponse)
			mockChClient.QueryReturns(channel.Response{
				Payload: sampleResponse,
			}, nil)

			sampleArgs = &types.EthArgs{
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

				err := ethservice.Call(&http.Request{}, &types.EthArgs{}, &reply)
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
			sampleArgs     *types.EthArgs
		)

		BeforeEach(func() {
			sampleResponse = channel.Response{
				Payload:       []byte("sample-response"),
				TransactionID: "1",
			}
			mockChClient.ExecuteReturns(sampleResponse, nil)

			sampleArgs = &types.EthArgs{
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

				zeroAddress := hex.EncodeToString(fab3.ZeroAddress)
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

				err := ethservice.SendTransaction(&http.Request{}, &types.EthArgs{}, &reply)
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
			otherTransaction    *peer.ProcessedTransaction
			sampleBlock         *common.Block
			sampleTransactionID string
			sampleAddress       string
		)

		BeforeEach(func() {
			sampleAddress = "82373458164820947891"
			sampleTransactionID = "1234567123"

			var err error
			sampleTransaction, err = GetSampleTransaction([][]byte{[]byte(sampleAddress), []byte("sample arg 2")}, []byte("sample-response"), []byte{}, sampleTransactionID)
			Expect(err).ToNot(HaveOccurred())

			otherTransaction, err = GetSampleTransaction([][]byte{[]byte("1234567"), []byte("sample arg 3")}, []byte("sample-response 2"), []byte{}, "5678")

			sampleBlock = GetSampleBlockWithTransaction(31, []byte("12345abcd"), otherTransaction, sampleTransaction)
			Expect(err).ToNot(HaveOccurred())

			mockLedgerClient.QueryBlockByTxIDReturns(sampleBlock, nil)
		})

		It("returns the transaction receipt associated to that transaction address", func() {
			var reply types.TxReceipt

			err := ethservice.GetTransactionReceipt(&http.Request{}, &sampleTransactionID, &reply)
			Expect(err).ToNot(HaveOccurred())

			Expect(mockLedgerClient.QueryBlockByTxIDCallCount()).To(Equal(1))
			txID, reqOpts := mockLedgerClient.QueryBlockByTxIDArgsForCall(0)
			Expect(txID).To(Equal(fab.TransactionID(sampleTransactionID)))
			Expect(reqOpts).To(HaveLen(0))

			Expect(reply).To(Equal(types.TxReceipt{
				TransactionHash:   "0x" + sampleTransactionID,
				TransactionIndex:  "0x1",
				BlockHash:         "0x" + hex.EncodeToString(blockHash(sampleBlock.GetHeader())),
				BlockNumber:       "0x1f",
				GasUsed:           0,
				CumulativeGasUsed: 0,
				To:                "0x" + sampleAddress,
				Status:            "0x1",
				From:              addrFromCert,
			}))
		})

		Context("when the transaction has associated events", func() {
			var (
				msg, msg2    event.Event
				eventPayload []byte
				eventBytes   []byte
			)

			BeforeEach(func() {
				var err error
				addr, err := crypto.AddressFromBytes([]byte(sampleAddress))
				Expect(err).ToNot(HaveOccurred())

				msg = event.Event{
					Address: strings.ToLower(addr.String()),
					Topics:  []string{"sample-topic-1", "sample-topic2"},
					Data:    "sample-data",
				}
				// A log with no data
				msg2 = event.Event{
					Address: strings.ToLower(addr.String()),
					Topics:  []string{"sample-topic-1", "sample-topic2"},
				}

				events := []event.Event{msg, msg2}
				eventPayload, err = json.Marshal(events)
				Expect(err).ToNot(HaveOccurred())

				chaincodeEvent := peer.ChaincodeEvent{
					ChaincodeId: "qscc",
					TxId:        sampleTransactionID,
					EventName:   "Chaincode event",
					Payload:     eventPayload,
				}

				eventBytes, err = proto.Marshal(&chaincodeEvent)
				Expect(err).ToNot(HaveOccurred())

				tx, err := GetSampleTransaction([][]byte{[]byte(sampleAddress), []byte("sample arg 2")}, []byte("sample-response"), eventBytes, sampleTransactionID)
				*sampleTransaction = *tx
				Expect(err).ToNot(HaveOccurred())

				*sampleBlock = *GetSampleBlockWithTransaction(31, []byte("12345abcd"), sampleTransaction, otherTransaction)
			})

			It("returns the transaction receipt associated to that transaction address", func() {
				var reply types.TxReceipt

				err := ethservice.GetTransactionReceipt(&http.Request{}, &sampleTransactionID, &reply)
				Expect(err).ToNot(HaveOccurred())
				Expect(mockLedgerClient.QueryBlockByTxIDCallCount()).To(Equal(1))
				txID, reqOpts := mockLedgerClient.QueryBlockByTxIDArgsForCall(0)
				Expect(txID).To(Equal(fab.TransactionID(sampleTransactionID)))
				Expect(reqOpts).To(HaveLen(0))

				topics := []string{}
				for _, topic := range msg.Topics {
					topics = append(topics, "0x"+topic)
				}

				expectedLog := types.Log{
					Address:     "0x" + hex.EncodeToString([]byte(sampleAddress)),
					Topics:      topics,
					Data:        "0x" + msg.Data,
					BlockNumber: "0x1f",
					TxHash:      "0x" + sampleTransactionID,
					TxIndex:     "0x0",
					BlockHash:   "0x" + hex.EncodeToString(blockHash(sampleBlock.GetHeader())),
					Index:       "0x0",
				}
				expectedLog2 := types.Log{
					Address:     "0x" + hex.EncodeToString([]byte(sampleAddress)),
					Topics:      topics,
					BlockNumber: "0x1f",
					TxHash:      "0x" + sampleTransactionID,
					TxIndex:     "0x0",
					BlockHash:   "0x" + hex.EncodeToString(blockHash(sampleBlock.GetHeader())),
					Index:       "0x1",
				}

				var expectedLogs []types.Log
				expectedLogs = make([]types.Log, 0)
				expectedLogs = append(expectedLogs, expectedLog)
				expectedLogs = append(expectedLogs, expectedLog2)
				Expect(reply).To(Equal(types.TxReceipt{
					TransactionHash:   "0x" + sampleTransactionID,
					TransactionIndex:  "0x0",
					BlockHash:         "0x" + hex.EncodeToString(blockHash(sampleBlock.GetHeader())),
					BlockNumber:       "0x1f",
					GasUsed:           0,
					CumulativeGasUsed: 0,
					To:                "0x" + sampleAddress,
					Logs:              expectedLogs,
					Status:            "0x1",
					From:              addrFromCert,
				}))
			})

		})

		Context("when the transaction is creation of a smart contract", func() {
			var contractAddress []byte
			BeforeEach(func() {
				contractAddress = []byte("123456789abcdef1234")
				zeroAddress := make([]byte, hex.EncodedLen(len(fab3.ZeroAddress)))
				hex.Encode(zeroAddress, fab3.ZeroAddress)

				tx, err := GetSampleTransaction([][]byte{zeroAddress, []byte("sample arg 2")}, contractAddress, []byte{}, sampleTransactionID)
				*sampleTransaction = *tx
				Expect(err).ToNot(HaveOccurred())

				*sampleBlock = *GetSampleBlockWithTransaction(31, []byte("12345abcd"), sampleTransaction, otherTransaction)
			})

			It("returns the contract address in the transaction receipt", func() {
				var reply types.TxReceipt

				err := ethservice.GetTransactionReceipt(&http.Request{}, &sampleTransactionID, &reply)
				Expect(err).ToNot(HaveOccurred())

				Expect(mockLedgerClient.QueryBlockByTxIDCallCount()).To(Equal(1))
				txID, reqOpts := mockLedgerClient.QueryBlockByTxIDArgsForCall(0)
				Expect(txID).To(Equal(fab.TransactionID(sampleTransactionID)))
				Expect(reqOpts).To(HaveLen(0))

				Expect(reply).To(Equal(types.TxReceipt{
					TransactionHash:   "0x" + sampleTransactionID,
					TransactionIndex:  "0x0",
					BlockHash:         "0x" + hex.EncodeToString(blockHash(sampleBlock.GetHeader())),
					BlockNumber:       "0x1f",
					ContractAddress:   "0x" + string(contractAddress),
					GasUsed:           0,
					CumulativeGasUsed: 0,
					Logs:              nil,
					Status:            "0x1",
					From:              addrFromCert,
				}))
			})

			Context("when transaction ID has `0x` prefix", func() {
				BeforeEach(func() {
					sampleTransactionID = "0x" + sampleTransactionID
				})
				It("strips the prefix before querying the ledger", func() {
					var reply types.TxReceipt

					err := ethservice.GetTransactionReceipt(&http.Request{}, &sampleTransactionID, &reply)
					Expect(err).ToNot(HaveOccurred())

					Expect(mockLedgerClient.QueryBlockByTxIDCallCount()).To(Equal(1))
					txID, reqOpts := mockLedgerClient.QueryBlockByTxIDArgsForCall(0)
					Expect(txID).To(Equal(fab.TransactionID(sampleTransactionID[2:])))
					Expect(reqOpts).To(HaveLen(0))

					Expect(reply).To(Equal(types.TxReceipt{
						TransactionHash:   sampleTransactionID,
						TransactionIndex:  "0x0",
						BlockHash:         "0x" + hex.EncodeToString(blockHash(sampleBlock.GetHeader())),
						BlockNumber:       "0x1f",
						ContractAddress:   "0x" + string(contractAddress),
						GasUsed:           0,
						CumulativeGasUsed: 0,
						Logs:              nil,
						Status:            "0x1",
						From:              addrFromCert,
					}))
				})
			})
		})

		Context("when requested transaction is not an evm smart contract transaction", func() {
			var (
				tooFewArgsTransaction, tooManyArgsTransaction, getCodeTransaction *peer.ProcessedTransaction
				txnID1, txnID2, txnID3                                            string
			)
			BeforeEach(func() {
				var err error

				txnID1 = "1234567123"
				txnID2 = "2234567123"
				txnID3 = "3234567123"

				tooFewArgsTransaction, err = GetSampleTransaction([][]byte{[]byte("82373458")}, []byte("sample-response"), []byte{}, txnID1)
				Expect(err).ToNot(HaveOccurred())

				tooManyArgsTransaction, err = GetSampleTransaction([][]byte{[]byte("82373458"), []byte("sample-arg2"), []byte("sample-arg3")}, []byte("sample-response"), []byte{}, txnID2)
				Expect(err).ToNot(HaveOccurred())

				getCodeTransaction, err = GetSampleTransaction([][]byte{[]byte("getCode"), []byte("sample-arg")}, []byte("sample-response 2"), []byte{}, txnID3)
				Expect(err).ToNot(HaveOccurred())

				sampleBlock = GetSampleBlockWithTransaction(31, []byte("12345abcd"), tooFewArgsTransaction, tooManyArgsTransaction, getCodeTransaction)
				Expect(err).ToNot(HaveOccurred())

				mockLedgerClient.QueryBlockByTxIDReturns(sampleBlock, nil)
			})

			It("does not provide to field when the requested tx has less than 2 args", func() {
				var reply types.TxReceipt
				err := ethservice.GetTransactionReceipt(&http.Request{}, &txnID1, &reply)
				Expect(err).ToNot(HaveOccurred())

				Expect(reply).To(Equal(types.TxReceipt{
					TransactionHash:   "0x" + txnID1,
					TransactionIndex:  "0x0",
					BlockHash:         "0x" + hex.EncodeToString(blockHash(sampleBlock.GetHeader())),
					BlockNumber:       "0x1f",
					GasUsed:           0,
					CumulativeGasUsed: 0,
					Status:            "0x1",
				}))
			})

			It("does not provide to field when the requested tx has more than 2 args", func() {
				var reply types.TxReceipt
				err := ethservice.GetTransactionReceipt(&http.Request{}, &txnID2, &reply)
				Expect(err).ToNot(HaveOccurred())

				Expect(reply).To(Equal(types.TxReceipt{
					TransactionHash:   "0x" + txnID2,
					TransactionIndex:  "0x1",
					BlockHash:         "0x" + hex.EncodeToString(blockHash(sampleBlock.GetHeader())),
					BlockNumber:       "0x1f",
					GasUsed:           0,
					CumulativeGasUsed: 0,
					Status:            "0x1",
				}))
			})

			It("does not provide to field when the requested tx is a getCode", func() {
				var reply types.TxReceipt
				err := ethservice.GetTransactionReceipt(&http.Request{}, &txnID3, &reply)
				Expect(err).ToNot(HaveOccurred())

				Expect(reply).To(Equal(types.TxReceipt{
					TransactionHash:   "0x" + txnID3,
					TransactionIndex:  "0x2",
					BlockHash:         "0x" + hex.EncodeToString(blockHash(sampleBlock.GetHeader())),
					BlockNumber:       "0x1f",
					GasUsed:           0,
					CumulativeGasUsed: 0,
					Status:            "0x1",
				}))
			})
		})

		Context("when the ledger errors when processing a query for the block", func() {
			BeforeEach(func() {
				mockLedgerClient.QueryBlockByTxIDReturns(nil, errors.New("boom!"))
			})

			It("returns a corresponding error", func() {
				var reply types.TxReceipt

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
			err := ethservice.EstimateGas(&http.Request{}, &types.EthArgs{}, &reply)
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
			var reply types.Block

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

			It("returns an error, when the second arg is not a boolean", func() {
				arg := make([]interface{}, 2)
				arg[0] = "latest"
				arg[1] = "durf"
				err := ethservice.GetBlockByNumber(&http.Request{}, &arg, &reply)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when there are good parameters", func() {
			var (
				reply                types.Block
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

				Context("when a block is requested by number", func() {
					var uintBlockNumber uint64

					BeforeEach(func() {
						requestedBlockNumber = "abc0"

						var err error
						uintBlockNumber, err = strconv.ParseUint(requestedBlockNumber, 16, 64)
						Expect(err).ToNot(HaveOccurred())
					})

					It("requests a block by number", func() {
						sampleBlock := GetSampleBlock(uintBlockNumber)
						mockLedgerClient.QueryBlockReturns(sampleBlock, nil)

						err := ethservice.GetBlockByNumber(&http.Request{}, &args, &reply)
						Expect(err).ToNot(HaveOccurred())

						Expect(reply.Number).To(Equal("0x"+requestedBlockNumber), "block number")
						Expect(reply.Hash).To(Equal("0x"+hex.EncodeToString(blockHash(sampleBlock.Header))), "block data hash")
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
						fullTransactions = false
					})

					Context("when asked for the latest block", func() {
						BeforeEach(func() {
							requestedBlockNumber = "latest"

							var err error
							uintBlockNumber, err = strconv.ParseUint("abc0", 16, 64)
							Expect(err).ToNot(HaveOccurred())

							mockLedgerClient.QueryInfoReturns(&fab.BlockchainInfoResponse{BCI: &common.BlockchainInfo{Height: uintBlockNumber + 1}}, nil)
						})

						It("returns the most recently formed block", func() {
							sampleBlock := GetSampleBlock(uintBlockNumber)
							mockLedgerClient.QueryBlockReturns(sampleBlock, nil)

							err := ethservice.GetBlockByNumber(&http.Request{}, &args, &reply)
							Expect(err).ToNot(HaveOccurred())
							Expect(reply.Number).To(Equal("0xabc0"), "block number")
							Expect(reply.Hash).To(Equal("0x"+hex.EncodeToString(blockHash(sampleBlock.Header))), "block data hash")
							Expect(reply.ParentHash).To(Equal("0x"+hex.EncodeToString(sampleBlock.Header.PreviousHash)), "block parent hash")

							Expect(mockLedgerClient.QueryBlockCallCount()).To(Equal(1))
							requestBlockNumber, _ := mockLedgerClient.QueryBlockArgsForCall(0)
							Expect(requestBlockNumber).To(Equal(uintBlockNumber))

							txns := reply.Transactions
							Expect(txns).To(HaveLen(2))
							Expect(txns[0]).To(BeEquivalentTo("0x5678"))
							Expect(txns[1]).To(BeEquivalentTo("0x1234"))
						})
					})

					Context("when asked for the earliest block", func() {
						BeforeEach(func() {
							requestedBlockNumber = "earliest"
						})

						It("returns the first block", func() {
							sampleBlock := GetSampleBlock(0)
							mockLedgerClient.QueryBlockReturns(sampleBlock, nil)

							err := ethservice.GetBlockByNumber(&http.Request{}, &args, &reply)
							Expect(err).ToNot(HaveOccurred())
							Expect(reply.Number).To(Equal("0x0"), "block number")
							Expect(reply.Hash).To(Equal("0x"+hex.EncodeToString(blockHash(sampleBlock.Header))), "block data hash")
							Expect(reply.ParentHash).To(Equal("0x"+hex.EncodeToString(sampleBlock.Header.PreviousHash)), "block parent hash")

							Expect(mockLedgerClient.QueryBlockCallCount()).To(Equal(1))
							requestBlockNumber, _ := mockLedgerClient.QueryBlockArgsForCall(0)
							Expect(requestBlockNumber).To(Equal(uint64(0)))

							txns := reply.Transactions
							Expect(txns).To(HaveLen(2))
							Expect(txns[0]).To(BeEquivalentTo("0x5678"))
							Expect(txns[1]).To(BeEquivalentTo("0x1234"))
						})
					})

					It("returns an error when asked for pending blocks", func() {
						args[0] = "pending"
						err := ethservice.GetBlockByNumber(&http.Request{}, &args, &reply)
						Expect(err).To(HaveOccurred())
					})

				})
			})

			Context("when asking for full transactions", func() {
				var uintBlockNumber uint64
				BeforeEach(func() {
					requestedBlockNumber = "abc0"
					fullTransactions = true

					var err error
					uintBlockNumber, err = strconv.ParseUint("abc0", 16, 64)
					Expect(err).ToNot(HaveOccurred())

					mockLedgerClient.QueryInfoReturns(&fab.BlockchainInfoResponse{BCI: &common.BlockchainInfo{Height: uintBlockNumber + 1}}, nil)
				})

				It("returns a block with transactions with detail", func() {
					sampleBlock := GetSampleBlock(uintBlockNumber)
					mockLedgerClient.QueryBlockReturns(sampleBlock, nil)

					err := ethservice.GetBlockByNumber(&http.Request{}, &args, &reply)
					Expect(err).ToNot(HaveOccurred())

					blockNumber := "0x" + requestedBlockNumber
					Expect(reply.Number).To(Equal(blockNumber), "block number")

					blockHash := "0x" + hex.EncodeToString(blockHash(sampleBlock.Header))
					Expect(reply.Hash).To(Equal(blockHash), "block data hash")
					Expect(reply.ParentHash).To(Equal("0x"+hex.EncodeToString(sampleBlock.Header.PreviousHash)), "block parent hash")

					txns := reply.Transactions
					Expect(txns).To(HaveLen(2))

					t0, ok := txns[0].(types.Transaction)
					Expect(ok).To(BeTrue())
					Expect(t0.BlockHash).To(Equal(blockHash))
					Expect(t0.BlockNumber).To(Equal(blockNumber))
					Expect(t0.To).To(Equal("0x12345678"))
					Expect(t0.Input).To(Equal("0xsample arg 1"))
					Expect(t0.TransactionIndex).To(Equal("0x0"))
					Expect(t0.Hash).To(Equal("0x5678"))

					t1, ok := txns[1].(types.Transaction)
					Expect(ok).To(BeTrue())
					Expect(t1.BlockHash).To(Equal(blockHash))
					Expect(t1.BlockNumber).To(Equal(blockNumber))
					Expect(t1.To).To(Equal("0x98765432"))
					Expect(t1.Input).To(Equal("0xsample arg 2"))
					Expect(t1.TransactionIndex).To(Equal("0x1"))
					Expect(t1.Hash).To(Equal("0x1234"))
				})
			})
		})
	})

	Describe("BlockNumber", func() {
		var reply string

		It("returns the latest block number", func() {
			uintBlockNumber, err := strconv.ParseUint("abc0", 16, 64)
			Expect(err).ToNot(HaveOccurred())
			mockLedgerClient.QueryInfoReturns(&fab.BlockchainInfoResponse{BCI: &common.BlockchainInfo{Height: uintBlockNumber + 1}}, nil)

			err = ethservice.BlockNumber(&http.Request{}, nil, &reply)
			Expect(err).ToNot(HaveOccurred())
			Expect(reply).To(Equal("0xabc0"), "block number")
		})

		It("returns a no blockchain info error when the ledger info results in an error", func() {
			mockLedgerClient.QueryInfoReturns(nil, fmt.Errorf("no block info"))

			err := ethservice.BlockNumber(&http.Request{}, nil, &reply)
			Expect(err).To(MatchError(ContainSubstring("no block info")))
		})
	})

	Describe("GetTransactionByHash", func() {
		var reply types.Transaction

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
			block := GetSampleBlock(1)
			mockLedgerClient.QueryBlockByTxIDReturns(block, nil)
			err := ethservice.GetTransactionByHash(&http.Request{}, &txID, &reply)
			Expect(err).ToNot(HaveOccurred())
			Expect(reply.Hash).To(Equal(txID), "txn id hash that was passed in")
			Expect(reply.BlockHash).To(Equal("0x"+hex.EncodeToString(blockHash(block.Header))), "block data hash")
			Expect(reply.BlockNumber).To(Equal("0x1"), "blocknumber")
			Expect(reply.TransactionIndex).To(Equal("0x1"), "txn Index")
			Expect(reply.To).To(Equal("0x98765432"))
			Expect(reply.Input).To(Equal("0xsample arg 2"))
		})

		Context("when requested transaction is not an evm smart contract transaction", func() {
			var (
				tooFewArgsTransaction, tooManyArgsTransaction, getCodeTransaction *peer.ProcessedTransaction
				sampleBlock                                                       *common.Block
				txnID1, txnID2, txnID3                                            string
			)
			BeforeEach(func() {
				var err error

				txnID1 = "1234567123"
				txnID2 = "2234567123"
				txnID3 = "3234567123"

				tooFewArgsTransaction, err = GetSampleTransaction([][]byte{[]byte("82373458")}, []byte("sample-response"), []byte{}, txnID1)
				Expect(err).ToNot(HaveOccurred())

				tooManyArgsTransaction, err = GetSampleTransaction([][]byte{[]byte("82373458"), []byte("sample-arg2"), []byte("sample-arg3")}, []byte("sample-response"), []byte{}, txnID2)
				Expect(err).ToNot(HaveOccurred())

				getCodeTransaction, err = GetSampleTransaction([][]byte{[]byte("getCode"), []byte("sample-arg")}, []byte("sample-response 2"), []byte{}, txnID3)
				Expect(err).ToNot(HaveOccurred())

				sampleBlock = GetSampleBlockWithTransaction(31, []byte("12345abcd"), tooFewArgsTransaction, tooManyArgsTransaction, getCodeTransaction)
				Expect(err).ToNot(HaveOccurred())

				mockLedgerClient.QueryBlockByTxIDReturns(sampleBlock, nil)
			})

			It("does not provide to or input field when the requested tx has less than 2 args", func() {
				var reply types.Transaction
				err := ethservice.GetTransactionByHash(&http.Request{}, &txnID1, &reply)
				Expect(err).ToNot(HaveOccurred())

				Expect(reply).To(Equal(types.Transaction{
					Hash:             "0x" + txnID1,
					TransactionIndex: "0x0",
					BlockHash:        "0x" + hex.EncodeToString(blockHash(sampleBlock.GetHeader())),
					BlockNumber:      "0x1f",
				}))
			})

			It("does not provide to field when the requested transaction has more than 2 args", func() {
				var reply types.Transaction
				err := ethservice.GetTransactionByHash(&http.Request{}, &txnID2, &reply)
				Expect(err).ToNot(HaveOccurred())

				Expect(reply).To(Equal(types.Transaction{
					Hash:             "0x" + txnID2,
					TransactionIndex: "0x1",
					BlockHash:        "0x" + hex.EncodeToString(blockHash(sampleBlock.GetHeader())),
					BlockNumber:      "0x1f",
				}))
			})

			It("does not provide to field when the requested transaction is a getCode", func() {
				var reply types.Transaction
				err := ethservice.GetTransactionByHash(&http.Request{}, &txnID3, &reply)
				Expect(err).ToNot(HaveOccurred())

				Expect(reply).To(Equal(types.Transaction{
					Hash:             "0x" + txnID3,
					TransactionIndex: "0x2",
					BlockHash:        "0x" + hex.EncodeToString(blockHash(sampleBlock.GetHeader())),
					BlockNumber:      "0x1f",
				}))
			})
		})
	})

	Describe("GetLogs", func() {
		var logsArgs *types.GetLogsArgs
		var reply *[]types.Log
		BeforeEach(func() {
			logsArgs = &types.GetLogsArgs{}
			reply = &[]types.Log{}
		})
		Context("errors appropriately", func() {
			It("fails when the ledger is down", func() {
				mockLedgerClient.QueryInfoReturns(nil, fmt.Errorf("it's broke"))
				Expect(ethservice.GetLogs(&http.Request{}, logsArgs, reply)).ToNot(Succeed())
				mockLedgerClient.QueryInfoReturns(&fab.BlockchainInfoResponse{BCI: &common.BlockchainInfo{Height: 1}}, nil)
				mockLedgerClient.QueryBlockReturns(nil, fmt.Errorf("yup still broke"))
				Expect(ethservice.GetLogs(&http.Request{}, logsArgs, reply)).ToNot(Succeed())
			})
			It("does not allow FromBlock to be greater than ToBlock", func() {
				logsArgs = &types.GetLogsArgs{FromBlock: "1", ToBlock: "0"}
				Expect(ethservice.GetLogs(&http.Request{}, logsArgs, reply)).ToNot(Succeed())
			})
			It("does not accept pending as a block parameter", func() {
				mockLedgerClient.QueryInfoReturns(&fab.BlockchainInfoResponse{BCI: &common.BlockchainInfo{Height: 1}}, nil)
				sampleBlock := GetSampleBlock(1)
				mockLedgerClient.QueryBlockReturns(sampleBlock, nil)

				logsArgs = &types.GetLogsArgs{FromBlock: "pending"}
				Expect(ethservice.GetLogs(&http.Request{}, logsArgs, reply)).ToNot(Succeed())
				logsArgs = &types.GetLogsArgs{ToBlock: "pending"}
				Expect(ethservice.GetLogs(&http.Request{}, logsArgs, reply)).ToNot(Succeed())
			})
		})
		Context("when valid arguments are given; with Block Number 2 available as the latest block", func() {
			BeforeEach(func() {
				mockLedgerClient.QueryInfoReturns(&fab.BlockchainInfoResponse{BCI: &common.BlockchainInfo{Height: 3}}, nil)
				sampleBlock1 := GetSampleBlock(1)
				sampleBlock2 := GetSampleBlock(2)
				qbs := func(b uint64, _ ...ledger.RequestOption) (*common.Block, error) {
					logger.Debug("mockblock", b)
					if b == 1 {
						return sampleBlock1, nil
					} else if b == 2 {
						return sampleBlock2, nil
					} else {
						return nil, fmt.Errorf("no block available for block number %d", b)
					}
				}
				mockLedgerClient.QueryBlockStub = qbs
				mockLedgerClient.QueryBlockByHashReturns(sampleBlock2, nil)
				logsArgs = &types.GetLogsArgs{}
				reply = &[]types.Log{}
			})

			It("accepts an empty input struct by defaulting", func() {
				Expect(ethservice.GetLogs(&http.Request{}, logsArgs, reply)).To(Succeed())
				Expect(len(*reply)).To(Equal(2))
				reply1 := reply
				By("running the same test with explicit args")
				logsArgs = &types.GetLogsArgs{FromBlock: "latest", ToBlock: "latest"}
				Expect(ethservice.GetLogs(&http.Request{}, logsArgs, reply)).To(Succeed())
				Expect(len(*reply)).To(Equal(2))
				reply2 := reply
				By("comparing the results of the two invocations")
				Expect(*reply1).To(Equal(*reply2))
			})

			// Block references as input
			It("returns the latest block when explicitly asking for latest", func() {
				logsArgs = &types.GetLogsArgs{FromBlock: "latest", ToBlock: "latest"}
				Expect(ethservice.GetLogs(&http.Request{}, logsArgs, reply)).To(Succeed())
				Expect(len(*reply)).To(Equal(2))
			})

			It("returns events from the requested block when asking by blockhash", func() {
				logsArgs = &types.GetLogsArgs{BlockHash: "deff"}
				Expect(ethservice.GetLogs(&http.Request{}, logsArgs, reply)).To(Succeed())
				Expect(len(*reply)).To(Equal(2))
			})

			It("returns logs when both FromBlock and ToBlock are specified", func() {
				logsArgs = &types.GetLogsArgs{FromBlock: "1", ToBlock: "2"}
				Expect(ethservice.GetLogs(&http.Request{}, logsArgs, reply)).To(Succeed())
				Expect(len(*reply)).To(Equal(4))
			})

			// Addresses as input
			It("returns no matches when given an address filter that does not match anything", func() {
				logsArgs = &types.GetLogsArgs{Address: types.AddressFilter{""}}
				Expect(ethservice.GetLogs(&http.Request{}, logsArgs, reply)).To(Succeed())
				Expect(len(*reply)).To(Equal(0), "Nothing to match")
			})

			It("returns logs associated when given a single address as a filter", func() {
				addr, err := crypto.AddressFromBytes([]byte("82373458164820947891"))
				Expect(err).ToNot(HaveOccurred())
				af, err := types.NewAddressFilter("0x" + addr.String())
				Expect(err).ToNot(HaveOccurred())
				logsArgs = &types.GetLogsArgs{Address: af}
				Expect(ethservice.GetLogs(&http.Request{}, logsArgs, reply)).To(Succeed())
				Expect(len(*reply)).To(Equal(1), "Only one of events to match")
			})

			It("returns the events associated when given an array of multiple addresses as a filter", func() {
				addr, err := crypto.AddressFromBytes([]byte("82373458164820947891"))
				Expect(err).ToNot(HaveOccurred())
				af, err := types.NewAddressFilter("0x" + addr.String())
				Expect(err).ToNot(HaveOccurred())
				addr, err = crypto.AddressFromBytes([]byte("82373458164820947892"))
				Expect(err).ToNot(HaveOccurred())
				af2, err := types.NewAddressFilter("0x" + addr.String())
				Expect(err).ToNot(HaveOccurred())
				logsArgs = &types.GetLogsArgs{Address: append(af, af2...)}
				Expect(ethservice.GetLogs(&http.Request{}, logsArgs, reply)).To(Succeed())
				Expect(len(*reply)).To(Equal(2))
			})

			// topics as input
			It("returns all events when given a number of empty topics less than the number of topics in the event", func() {
				logsArgs = &types.GetLogsArgs{Topics: types.TopicsFilter{nil}}
				Expect(ethservice.GetLogs(&http.Request{}, logsArgs, reply)).To(Succeed())
				Expect(len(*reply)).To(Equal(2), "all in block to match")
			})

			It("returns all events when given a number of empty topics equal to number of topics in the event", func() {
				logsArgs = &types.GetLogsArgs{Topics: types.TopicsFilter{nil, nil}}
				Expect(ethservice.GetLogs(&http.Request{}, logsArgs, reply)).To(Succeed())
				Expect(len(*reply)).To(Equal(2), "all in block to match")
			})

			It("returns no events when given a number of topics more than the number of topics in the event", func() {
				logsArgs = &types.GetLogsArgs{Topics: types.TopicsFilter{nil, nil, nil}}
				Expect(ethservice.GetLogs(&http.Request{}, logsArgs, reply)).To(Succeed())
				Expect(len(*reply)).To(Equal(0), "none in block to match")
			})

			It("returns the events associated when given a single topic to match", func() {
				tf, err := types.NewTopicFilter("0x" + formatTopic("sample-topic-1"))
				Expect(err).ToNot(HaveOccurred())
				logsArgs = &types.GetLogsArgs{Topics: types.NewTopicsFilter(tf)}
				Expect(ethservice.GetLogs(&http.Request{}, logsArgs, reply)).To(Succeed())
				Expect(len(*reply)).To(Equal(1), "Only one of events to match")
			})

			It("returns the events associated when given nil as the first topic, and a matcher for the second topic", func() {
				tf, err := types.NewTopicFilter("0x" + formatTopic("sample-topic-2"))
				Expect(err).ToNot(HaveOccurred())
				logsArgs = &types.GetLogsArgs{Topics: types.TopicsFilter{nil, tf}}
				Expect(ethservice.GetLogs(&http.Request{}, logsArgs, reply)).To(Succeed())
				Expect(len(*reply)).To(Equal(1), "only one in block to match")
			})

			It("returns the events associated when given an array of and'd topics", func() {
				tf1, err := types.NewTopicFilter("0x" + formatTopic("sample-topic-1"))
				Expect(err).ToNot(HaveOccurred())
				tf2, err := types.NewTopicFilter("0x" + formatTopic("sample-topic-2"))
				Expect(err).ToNot(HaveOccurred())
				logsArgs = &types.GetLogsArgs{Topics: types.NewTopicsFilter(tf1, tf2)}
				Expect(ethservice.GetLogs(&http.Request{}, logsArgs, reply)).To(Succeed())
				Expect(len(*reply)).To(Equal(1), "Only one of events to match")
			})

			It("returns the events associated when given an array of multiple or'd topics", func() {
				tf1, err := types.NewTopicFilter("0x" + formatTopic("sample-topic-1"))
				Expect(err).ToNot(HaveOccurred())
				tf2, err := types.NewTopicFilter("0x" + formatTopic("sample-topic-3"))
				Expect(err).ToNot(HaveOccurred())
				tf := append(tf1, tf2...)
				logsArgs = &types.GetLogsArgs{Topics: types.NewTopicsFilter(tf)}
				Expect(ethservice.GetLogs(&http.Request{}, logsArgs, reply)).To(Succeed())
				Expect(len(*reply)).To(Equal(2), "all events in block to match")
			})

			// combination of multiple fields as input
			It("returns logs associated when given a single address and single topic as a filter", func() {
				addr, err := crypto.AddressFromBytes([]byte("82373458164820947891"))
				Expect(err).ToNot(HaveOccurred())
				af, err := types.NewAddressFilter("0x" + addr.String())
				Expect(err).ToNot(HaveOccurred())

				tf, err := types.NewTopicFilter("0x" + formatTopic("sample-topic-1"))
				Expect(err).ToNot(HaveOccurred())

				logsArgs = &types.GetLogsArgs{Address: af, Topics: types.TopicsFilter{tf}}
				Expect(ethservice.GetLogs(&http.Request{}, logsArgs, reply)).To(Succeed())
				Expect(len(*reply)).To(Equal(1), "Only one of events to match")
			})

			It("returns no logs associated when given a single address and single topic, with none that match the same object as a filter", func() {
				addr, err := crypto.AddressFromBytes([]byte("82373458164820947891"))
				Expect(err).ToNot(HaveOccurred())
				af, err := types.NewAddressFilter("0x" + addr.String())
				Expect(err).ToNot(HaveOccurred())

				tf, err := types.NewTopicFilter("0x" + formatTopic("sample-topic-3"))
				Expect(err).ToNot(HaveOccurred())

				logsArgs = &types.GetLogsArgs{Address: af, Topics: types.TopicsFilter{tf}}
				Expect(ethservice.GetLogs(&http.Request{}, logsArgs, reply)).To(Succeed())
				Expect(len(*reply)).To(Equal(0), "none of the events to match")
			})
		})
	})

	Describe("GetTransactionCount", func() {
		It("always returns 0x0", func() {
			var reply string
			err := ethservice.GetTransactionCount(&http.Request{}, nil, &reply)
			Expect(err).ToNot(HaveOccurred())
			Expect(reply).To(Equal("0x0"))
		})
	})
})

func formatTopic(s string) string {
	if len(s) > 64 {
		s = s[:64]
	}
	return fmt.Sprintf("%64v", s)
}

func GetSampleBlock(blockNumber uint64) *common.Block {
	addr, err := crypto.AddressFromBytes([]byte("82373458164820947891"))
	Expect(err).ToNot(HaveOccurred())

	msg := event.Event{
		Address: strings.ToLower(addr.String()),
		Topics:  []string{formatTopic("sample-topic-1"), formatTopic("sample-topic-2")},
		Data:    "sample-data",
	}
	events := []event.Event{msg}
	eventPayload, err := json.Marshal(events)
	Expect(err).ToNot(HaveOccurred())

	chaincodeEvent := peer.ChaincodeEvent{
		ChaincodeId: "evmcc",
		TxId:        "1234",
		EventName:   "Chaincode event",
		Payload:     eventPayload,
	}

	eventBytes, err := proto.Marshal(&chaincodeEvent)
	Expect(err).ToNot(HaveOccurred())

	addr, err = crypto.AddressFromBytes([]byte("82373458164820947892"))
	Expect(err).ToNot(HaveOccurred())

	msg = event.Event{
		Address: strings.ToLower(addr.String()),
		Topics:  []string{formatTopic("sample-topic-3"), formatTopic("sample-topic-4")},
		Data:    "sample-data",
	}
	events = []event.Event{msg}
	eventPayload, err = json.Marshal(events)
	Expect(err).ToNot(HaveOccurred())

	chaincodeEvent = peer.ChaincodeEvent{
		ChaincodeId: "evmcc",
		TxId:        "1234",
		EventName:   "Chaincode event",
		Payload:     eventPayload,
	}

	eventBytes2, err := proto.Marshal(&chaincodeEvent)
	Expect(err).ToNot(HaveOccurred())

	tx, err := GetSampleTransaction([][]byte{[]byte("12345678"), []byte("sample arg 1")}, []byte("sample-response1"), eventBytes, "5678")
	Expect(err).ToNot(HaveOccurred())
	txn1, err := proto.Marshal(tx.TransactionEnvelope)
	Expect(err).ToNot(HaveOccurred())

	tx, err = GetSampleTransaction([][]byte{[]byte("98765432"), []byte("sample arg 2")}, []byte("sample-response2"), eventBytes2, "1234")
	txn2, err := proto.Marshal(tx.TransactionEnvelope)
	Expect(err).ToNot(HaveOccurred())

	phash := []byte("abc\x00")
	dhash := []byte("def\xFF")
	return &common.Block{
		Header: &common.BlockHeader{Number: blockNumber,
			PreviousHash: phash,
			DataHash:     dhash},
		Data: &common.BlockData{Data: [][]byte{txn1, txn2}},
		// each block data needs each of the metadata
		Metadata: &common.BlockMetadata{Metadata: [][]byte{{0, 0}, {0, 0}, {0, 0}, {0, 0}}},
	}
}

func GetSampleBlockWithTransaction(blockNumber uint64, blkHash []byte, txns ...*peer.ProcessedTransaction) *common.Block {

	blockData := [][]byte{}
	blockMetadata := [][]byte{{}, {}, {}, {}}
	transactionsFilter := []byte{}

	for _, tx := range txns {
		txn, err := proto.Marshal(tx.TransactionEnvelope)
		Expect(err).ToNot(HaveOccurred())

		blockData = append(blockData, txn)
		transactionsFilter = append(transactionsFilter, '0')
	}

	blockMetadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER] = transactionsFilter

	phash := []byte("abc\x00")
	return &common.Block{
		Header: &common.BlockHeader{Number: blockNumber,
			PreviousHash: phash,
			DataHash:     blkHash},
		Data:     &common.BlockData{Data: blockData},
		Metadata: &common.BlockMetadata{Metadata: blockMetadata},
	}
}

func GetSampleTransaction(inputArgs [][]byte, txResponse, eventBytes []byte, txId string) (*peer.ProcessedTransaction, error) {

	respPayload := &peer.ChaincodeAction{
		Events: eventBytes,
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

	chdr := &common.ChannelHeader{Type: int32(common.HeaderType_ENDORSER_TRANSACTION), TxId: txId}
	chdrBytes, err := proto.Marshal(chdr)
	if err != nil {
		return &peer.ProcessedTransaction{}, err
	}

	creator, err := proto.Marshal(&msp.SerializedIdentity{IdBytes: []byte(cert)})
	if err != nil {
		return nil, err
	}

	sigHdr, err := proto.Marshal(&common.SignatureHeader{Creator: creator})
	if err != nil {
		return nil, err
	}

	payload := &common.Payload{
		Header: &common.Header{
			ChannelHeader:   chdrBytes,
			SignatureHeader: sigHdr,
		},
		Data: actionsPayload,
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		return &peer.ProcessedTransaction{}, err
	}

	tx := &peer.ProcessedTransaction{
		TransactionEnvelope: &common.Envelope{
			Payload:   payloadBytes,
			Signature: []byte(cert),
		},
	}

	return tx, nil
}

type asn1Header struct {
	Number       int64
	PreviousHash []byte
	DataHash     []byte
}

// Bytes returns the ASN.1 marshaled representation of the block header.
func blockHash(b *common.BlockHeader) []byte {
	asn1Header := asn1Header{
		PreviousHash: b.PreviousHash,
		DataHash:     b.DataHash,
	}
	if b.Number > uint64(math.MaxInt64) {
		panic(fmt.Errorf("Golang does not currently support encoding uint64 to asn1"))
	} else {
		asn1Header.Number = int64(b.Number)
	}
	result, err := asn1.Marshal(asn1Header)
	if err != nil {
		// Errors should only arise for types which cannot be encoded, since the
		// BlockHeader type is known a-priori to contain only encodable types, an
		// error here is fatal and should not be propogated
		panic(err)
	}

	//util.ComputeSHA256(result)
	h := sha256.New()
	h.Write(result)
	return h.Sum(nil)
}
