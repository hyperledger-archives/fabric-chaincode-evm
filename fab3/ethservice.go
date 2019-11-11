/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fab3

import (
	"bytes"
	"crypto/sha256"
	"encoding/asn1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/ledger"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/core/ledger/util"
	"github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/peer"

	"github.com/hyperledger/fabric-chaincode-evm/evmcc/address"
	"github.com/hyperledger/fabric-chaincode-evm/evmcc/event"
	"github.com/hyperledger/fabric-chaincode-evm/fab3/types"
)

var ZeroAddress = make([]byte, 20)

//go:generate counterfeiter -o mocks/mockchannelclient.go --fake-name MockChannelClient ./ ChannelClient

type ChannelClient interface {
	Query(request channel.Request, options ...channel.RequestOption) (channel.Response, error)
	Execute(request channel.Request, options ...channel.RequestOption) (channel.Response, error)
}

//go:generate counterfeiter -o mocks/mockledgerclient.go --fake-name MockLedgerClient ./ LedgerClient

type LedgerClient interface {
	QueryInfo(options ...ledger.RequestOption) (*fab.BlockchainInfoResponse, error)
	QueryBlock(blockNumber uint64, options ...ledger.RequestOption) (*common.Block, error)
	QueryBlockByTxID(txid fab.TransactionID, options ...ledger.RequestOption) (*common.Block, error)
	QueryBlockByHash(blockHash []byte, options ...ledger.RequestOption) (*common.Block, error)
}

//go:generate counterfeiter -o mocks/mockethservice.go --fake-name MockEthService ./ EthService

// EthService is the rpc server implementation. Each function is an
// implementation of one ethereum json-rpc
// https://github.com/ethereum/wiki/wiki/JSON-RPC
//
// Arguments and return values are formatted as HEX value encoding
// https://github.com/ethereum/wiki/wiki/JSON-RPC#hex-value-encoding
//
// gorilla RPC is the receiver of these functions, they must all take three
// pointers, and return a single error
//
// see godoc for RegisterService(receiver interface{}, name string) error
//
type EthService interface {
	GetCode(r *http.Request, arg *string, reply *string) error
	Call(r *http.Request, args *types.EthArgs, reply *string) error
	SendTransaction(r *http.Request, args *types.EthArgs, reply *string) error
	GetTransactionReceipt(r *http.Request, arg *string, reply *types.TxReceipt) error
	Accounts(r *http.Request, arg *string, reply *[]string) error
	EstimateGas(r *http.Request, args *types.EthArgs, reply *string) error
	GetBalance(r *http.Request, p *[]string, reply *string) error
	GetBlockByNumber(r *http.Request, p *[]interface{}, reply *types.Block) error
	BlockNumber(r *http.Request, _ *interface{}, reply *string) error
	GetTransactionByHash(r *http.Request, txID *string, reply *types.Transaction) error
	GetTransactionCount(r *http.Request, _ *interface{}, reply *string) error
	GetLogs(*http.Request, *types.GetLogsArgs, *[]types.Log) error
	NewFilter(*http.Request, *types.GetLogsArgs, *string) error
	UninstallFilter(*http.Request, *string, *bool) error
}

type ethService struct {
	channelClient ChannelClient
	ledgerClient  LedgerClient
	channelID     string
	ccid          string
	logger        *zap.SugaredLogger
	filterMapLock sync.Mutex
	filterMap     map[uint64]interface{}
	filterSeq     uint64
}

func NewEthService(channelClient ChannelClient, ledgerClient LedgerClient, channelID string, ccid string, logger *zap.SugaredLogger) EthService {
	return &ethService{
		channelClient: channelClient,
		ledgerClient:  ledgerClient,
		channelID:     channelID,
		ccid:          ccid,
		logger:        logger.Named("ethservice"),
		filterMap:     make(map[uint64]interface{}),
	}
}

func (s *ethService) GetCode(r *http.Request, arg *string, reply *string) error {
	strippedAddr := strip0x(*arg)

	response, err := s.query(s.ccid, "getCode", [][]byte{[]byte(strippedAddr)})

	if err != nil {
		return fmt.Errorf("Failed to query the ledger: %s", err)
	}

	*reply = string(response.Payload)

	return nil
}

func (s *ethService) Call(r *http.Request, args *types.EthArgs, reply *string) error {
	response, err := s.query(s.ccid, strip0x(args.To), [][]byte{[]byte(strip0x(args.Data))})

	if err != nil {
		return fmt.Errorf("Failed to query the ledger: %s", err)
	}

	// Clients expect the prefix to present in responses
	*reply = "0x" + hex.EncodeToString(response.Payload)

	return nil
}

func (s *ethService) SendTransaction(r *http.Request, args *types.EthArgs, reply *string) error {
	if args.To == "" {
		args.To = hex.EncodeToString(ZeroAddress)
	}

	response, err := s.channelClient.Execute(channel.Request{
		ChaincodeID: s.ccid,
		Fcn:         strip0x(args.To),
		Args:        [][]byte{[]byte(strip0x(args.Data))},
	})

	if err != nil {
		return fmt.Errorf("Failed to execute transaction: %s", err)
	}
	*reply = string(response.TransactionID)
	return nil
}

func (s *ethService) GetTransactionReceipt(r *http.Request, txID *string, reply *types.TxReceipt) error {
	logger := s.logger.With("method", "GetTransactionReceipt")
	strippedTxID := strip0x(*txID)

	block, err := s.ledgerClient.QueryBlockByTxID(fab.TransactionID(strippedTxID))
	if err != nil {
		return fmt.Errorf("Failed to query the ledger: %s", err)
	}

	blkHeader := block.GetHeader()
	transactionsFilter := util.TxValidationFlags(block.GetMetadata().GetMetadata()[common.BlockMetadataIndex_TRANSACTIONS_FILTER])
	receipt := types.TxReceipt{
		TransactionHash:   "0x" + strippedTxID,
		BlockHash:         "0x" + hex.EncodeToString(blockHash(blkHeader)),
		BlockNumber:       "0x" + strconv.FormatUint(blkHeader.GetNumber(), 16),
		GasUsed:           0,
		CumulativeGasUsed: 0,
	}

	index, txPayload, err := findTransaction(strippedTxID, block.GetData().GetData())
	if err != nil {
		return fmt.Errorf("Failed parsing the transactions in the block: %s", err)
	}

	receipt.TransactionIndex = index
	indexU, _ := strconv.ParseUint(strip0x(index), 16, 64)
	// for ethereum transactions, 0 is invalid, 1 is valid
	txnValidValue := 0
	if transactionsFilter.IsValid(int(indexU)) {
		txnValidValue = 1
	}

	receipt.Status = "0x" + strconv.Itoa(txnValidValue)

	to, _, from, respPayload, err := getTransactionInformation(txPayload)
	if err != nil {
		return fmt.Errorf("Failed getting transaction infomration: %s", err)
	}

	receipt.From = from
	if to != "" {
		callee, err := hex.DecodeString(to)
		if err != nil {
			return fmt.Errorf("Failed to decode to address: %s", err)
		}

		if bytes.Equal(callee, ZeroAddress) {
			receipt.ContractAddress = "0x" + string(respPayload.GetResponse().GetPayload())
		} else {
			receipt.To = "0x" + to
		}
	}

	txLogs, err := fabricEventToEVMLogs(logger, respPayload.Events, receipt.BlockNumber, receipt.TransactionHash,
		receipt.TransactionIndex, receipt.BlockHash, types.AddressFilter{}, types.TopicsFilter{})
	if err != nil {
		return errors.Wrap(err, "failed to get EVM Logs out of fabric event")
	}
	receipt.Logs = txLogs

	*reply = receipt
	return nil
}

func (s *ethService) Accounts(r *http.Request, arg *string, reply *[]string) error {
	response, err := s.query(s.ccid, "account", [][]byte{})
	if err != nil {
		return fmt.Errorf("Failed to query the ledger: %s", err)
	}

	*reply = []string{"0x" + strings.ToLower(string(response.Payload))}

	return nil
}

// EstimateGas accepts the same arguments as Call but all arguments are
// optional.  This implementation ignores all arguments and returns a zero
// estimate.
//
// The intention is to estimate how much gas is necessary to allow a transaction
// to complete.
//
// EVM-chaincode does not require gas to run transactions. The chaincode will
// give enough gas per transaction.
func (s *ethService) EstimateGas(r *http.Request, _ *types.EthArgs, reply *string) error {
	s.logger.Debug("EstimateGas called")
	*reply = "0x0"
	return nil
}

// GetBalance takes an address and a block, but this implementation
// does not check or use either parameter.
//
// Always returns zero.
func (s *ethService) GetBalance(r *http.Request, p *[]string, reply *string) error {
	s.logger.Debug("GetBalance called")
	*reply = "0x0"
	return nil
}

// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_getblockbynumber
func (s *ethService) GetBlockByNumber(r *http.Request, p *[]interface{}, reply *types.Block) error {
	s.logger.Debug("Received a request for GetBlockByNumber")
	params := *p
	s.logger.Debug("Params are : ", params)

	// handle params
	// must have two params
	numParams := len(params)
	if numParams != 2 {
		return fmt.Errorf("need 2 params, got %q", numParams)
	}
	// first arg is string of block to get
	number, ok := params[0].(string)
	if !ok {
		s.logger.Debugf("Incorrect argument received: %#v", params[0])
		return fmt.Errorf("Incorrect first parameter sent, must be string")
	}

	// second arg is bool for full txn or hash txn
	fullTransactions, ok := params[1].(bool)
	if !ok {
		return fmt.Errorf("Incorrect second parameter sent, must be boolean")
	}

	parsedNumber, err := s.parseBlockNum(strip0x(number))
	if err != nil {
		return err
	}

	block, err := s.ledgerClient.QueryBlock(parsedNumber)
	if err != nil {
		return fmt.Errorf("Failed to query the ledger: %v", err)
	}

	blkHeader := block.GetHeader()

	blockHash := "0x" + hex.EncodeToString(blockHash(blkHeader))
	blockNumber := "0x" + strconv.FormatUint(parsedNumber, 16)

	// each data is a txn
	data := block.GetData().GetData()
	transactionsFilter := util.TxValidationFlags(block.GetMetadata().GetMetadata()[common.BlockMetadataIndex_TRANSACTIONS_FILTER])
	txns := make([]interface{}, 0, len(data))

	// drill into the block to find the transaction ids it contains
	for index, transactionData := range data {
		if transactionData == nil {
			continue
		}

		if !transactionsFilter.IsValid(index) {
			continue
		}

		payload, chdr, err := getChannelHeaderandPayloadFromTransactionData(transactionData)
		if err != nil {
			return err
		}

		if fullTransactions {
			txn := types.Transaction{
				BlockHash:        blockHash,
				BlockNumber:      blockNumber,
				TransactionIndex: "0x" + strconv.FormatUint(uint64(index), 16),
				Hash:             "0x" + chdr.TxId,
			}
			to, input, from, _, err := getTransactionInformation(payload)
			if err != nil {
				return err
			}

			txn.To = "0x" + to
			txn.Input = "0x" + input
			txn.From = from
			txns = append(txns, txn)
		} else {
			txns = append(txns, "0x"+chdr.TxId)
		}
	}

	blk := types.Block{
		BlockData: types.BlockData{
			Number:     blockNumber,
			Hash:       blockHash,
			ParentHash: "0x" + hex.EncodeToString(blkHeader.GetPreviousHash()),
		},
		Transactions: txns,
	}
	s.logger.Debug("asked for block", number, "found block", blk)

	*reply = blk
	return nil
}

func (s *ethService) BlockNumber(r *http.Request, _ *interface{}, reply *string) error {
	blockNumber, err := s.parseBlockNum("latest")
	if err != nil {
		return fmt.Errorf("failed to get latest block number: %s", err)
	}
	*reply = "0x" + strconv.FormatUint(uint64(blockNumber), 16)

	return nil
}

// GetTransactionByHash takes a TransactionID as a string and returns the
// details of the transaction.
//
// The implementation of this function follows the EVM ChainCode implementation
// of Invoke.
//
// Since this takes only one string, we can have gorilla verify that it has
// received only a single string, and skip our own verification.
//
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_gettransactionbyhash
func (s *ethService) GetTransactionByHash(r *http.Request, txID *string, reply *types.Transaction) error {
	strippedTxId := strip0x(*txID)

	if strippedTxId == "" {
		return fmt.Errorf("txID was empty")
	}
	s.logger.Debug("GetTransactionByHash", strippedTxId) // logging input to function

	txn := types.Transaction{
		Hash: "0x" + strippedTxId,
	}

	block, err := s.ledgerClient.QueryBlockByTxID(fab.TransactionID(strippedTxId))
	if err != nil {
		return fmt.Errorf("Failed to query the ledger: %s", err)
	}
	blkHeader := block.GetHeader()
	txn.BlockHash = "0x" + hex.EncodeToString(blockHash(blkHeader))
	txn.BlockNumber = "0x" + strconv.FormatUint(blkHeader.GetNumber(), 16)

	index, txPayload, err := findTransaction(strippedTxId, block.GetData().GetData())
	if err != nil {
		return fmt.Errorf("Failed to parse through transactions in the block: %s", err)
	}

	txn.TransactionIndex = index

	to, input, from, _, err := getTransactionInformation(txPayload)
	if err != nil {
		return err
	}

	if to != "" {
		txn.To = "0x" + to
	}

	if input != "" {
		txn.Input = "0x" + input
	}
	txn.From = from

	*reply = txn
	return nil
}

// GetTransactionCount will always return 0 regardless of the provided parameters.
//
// This implmentation does not match the Ethereum JSON RPC API definition of eth_getTransactionCount.
// This has been stubbed out in order to have Truffle IDE support.
func (s *ethService) GetTransactionCount(r *http.Request, _ *interface{}, reply *string) error {
	*reply = "0x0"
	return nil
}

// GetLogs returns matching logs in range FromBlock to ToBlock. If BlockHash is specified, the
// single matching block is searched for logs.
func (s *ethService) GetLogs(r *http.Request, args *types.GetLogsArgs, logs *[]types.Log) error {
	logger := s.logger.With("method", "GetLogs")
	logger.Debug("parameters", args)

	var from, to uint64
	var err error
	if args.BlockHash != "" {
		hash, err := hex.DecodeString(args.BlockHash)
		if err != nil {
			return errors.Wrap(err, "failed to parse the block hash")
		}
		block, err := s.ledgerClient.QueryBlockByHash(hash)
		if err != nil {
			return errors.Wrap(err, "failed to find block by block hash")
		}
		blockNumber := block.Header.Number
		from = blockNumber
		to = blockNumber
	} else {
		// set defaults *after* checking for input conflicts and validating
		if args.FromBlock == "" {
			args.FromBlock = "latest"
		}
		if args.ToBlock == "" {
			args.ToBlock = "latest"
		}

		from, err = s.parseBlockNum(args.FromBlock)
		if err != nil {
			return errors.Wrap(err, "failed to parse the block number")
		}
		// check if both from and to are the same to avoid doing two
		// queries to the fabric network.
		if args.FromBlock == args.ToBlock {
			to = from
		} else {
			to, err = s.parseBlockNum(args.ToBlock)
			if err != nil {
				return errors.Wrap(err, "failed to parse the block number")
			}
		}
		if from > to {
			return fmt.Errorf("fromBlock number greater than toBlock number")
		}
	}

	var txLogs []types.Log

	logger.Debugw("checking blocks for logs", "from", from, "to", to)
	for blockNumber := from; blockNumber <= to; blockNumber++ {
		logger = logger.With("block-number", blockNumber)
		logger.Debug("handling single block")
		block, err := s.ledgerClient.QueryBlock(blockNumber)
		if err != nil {
			return errors.Wrap(err, "failed to query the ledger")
		}
		blockHeader := block.GetHeader()
		blockHash := "0x" + hex.EncodeToString(blockHash(blockHeader))
		blockData := block.GetData().GetData()
		transactionsFilter := util.TxValidationFlags(block.GetMetadata().GetMetadata()[common.BlockMetadataIndex_TRANSACTIONS_FILTER])
		logger.Debug("handling ", len(blockData), " transactions in block")
		for transactionIndex, transactionData := range blockData {
			logger = logger.With("transaction-index", transactionIndex)
			// check validity of transaction
			if !transactionsFilter.IsValid(transactionIndex) || (transactionData == nil) {
				continue
			}

			// start processing the transaction
			payload, chdr, err := getChannelHeaderandPayloadFromTransactionData(transactionData)
			if err != nil {
				return errors.Wrap(err, "failed to unmarshal the transaction")
			}
			// only process transactions
			if chdr.Type != int32(common.HeaderType_ENDORSER_TRANSACTION) {
				logger.Debug("skipping non-ENDORSER_TRANSACTION")
				continue
			}

			transactionHash := "0x" + chdr.TxId
			logger.Debug("transaction ", transactionIndex, " has hash ", transactionHash)

			var respPayload *peer.ChaincodeAction
			_, _, _, respPayload, err = getTransactionInformation(payload)
			if err != nil {
				return errors.Wrap(err, "failed to unmarshal the transaction details")
			}

			blkNumber := "0x" + strconv.FormatUint(blockNumber, 16)
			transactionIndexStr := "0x" + strconv.FormatUint(uint64(transactionIndex), 16)
			logs, err := fabricEventToEVMLogs(logger, respPayload.Events, blkNumber, transactionHash,
				transactionIndexStr, blockHash, args.Address, args.Topics)
			if err != nil {
				return errors.Wrap(err, "failed to get EVM Logs out of fabric event")
			}
			txLogs = append(txLogs, logs...)
		}
	}

	logger.Debug("returning logs", txLogs)
	*logs = txLogs

	return nil
}

func (s *ethService) NewFilter(_ *http.Request, filter *types.GetLogsArgs, result *string) error {
	s.filterMapLock.Lock()
	s.filterSeq++
	index := s.filterSeq
	s.filterMap[index] = filter
	s.filterMapLock.Unlock()
	*result = "0x" + strconv.FormatUint(index, 16)
	return nil
}

func (s *ethService) UninstallFilter(_ *http.Request, filterID *string, removed *bool) error {
	id, err := strconv.ParseUint(strip0x(*filterID), 16, 64)
	if err != nil {
		return errors.Wrap(err, "failed to parse filter id")
	}

	s.filterMapLock.Lock()
	defer s.filterMapLock.Unlock()

	if _, ok := s.filterMap[id]; ok {
		delete(s.filterMap, id)
		*removed = true
	}

	return nil
}

func (s *ethService) query(ccid, function string, queryArgs [][]byte) (channel.Response, error) {
	return s.channelClient.Query(channel.Request{
		ChaincodeID: ccid,
		Fcn:         function,
		Args:        queryArgs,
	})
}

// https://github.com/ethereum/wiki/wiki/JSON-RPC#the-default-block-parameter
func (s *ethService) parseBlockNum(input string) (uint64, error) {
	// check if it's one of the named-blocks
	switch input {
	case "latest":
		// latest
		// qscc GetChainInfo, for a BlockchainInfo
		// from that take the height
		// using the height, call GetBlockByNumber

		blockchainInfo, err := s.ledgerClient.QueryInfo()
		if err != nil {
			s.logger.Debug(err)
			return 0, fmt.Errorf("failed to query the ledger: %v", err)
		}
		// height is the block being worked on now, we want the previous block
		topBlockNumber := blockchainInfo.BCI.GetHeight() - 1
		return topBlockNumber, nil
	case "earliest":
		return 0, nil
	case "pending":
		return 0, fmt.Errorf("unsupported: fabric does not have the concept of in-progress blocks being visible")
	default:
		return strconv.ParseUint(input, 16, 64)
	}
}

func strip0x(addr string) string {
	//Not checking for malformed addresses just stripping `0x` prefix where applicable
	return strings.TrimPrefix(addr, "0x")
}

func getPayloads(txActions *peer.TransactionAction) (*peer.ChaincodeProposalPayload, *peer.ChaincodeAction, error) {
	// TODO: pass in the tx type (in what follows we're assuming the type is ENDORSER_TRANSACTION)
	ccPayload := &peer.ChaincodeActionPayload{}
	err := proto.Unmarshal(txActions.Payload, ccPayload)
	if err != nil {
		return nil, nil, err
	}

	if ccPayload.Action == nil || ccPayload.Action.ProposalResponsePayload == nil {
		return nil, nil, fmt.Errorf("no payload in ChaincodeActionPayload")
	}

	ccProposalPayload := &peer.ChaincodeProposalPayload{}
	err = proto.Unmarshal(ccPayload.ChaincodeProposalPayload, ccProposalPayload)
	if err != nil {
		return nil, nil, err
	}

	pRespPayload := &peer.ProposalResponsePayload{}
	err = proto.Unmarshal(ccPayload.Action.ProposalResponsePayload, pRespPayload)
	if err != nil {
		return nil, nil, err
	}

	if pRespPayload.Extension == nil {
		return nil, nil, fmt.Errorf("response payload is missing extension")
	}

	respPayload := &peer.ChaincodeAction{}
	err = proto.Unmarshal(pRespPayload.Extension, respPayload)
	if err != nil {
		return ccProposalPayload, nil, err
	}
	return ccProposalPayload, respPayload, nil
}

// getTransactionInformation takes a payload
// It returns if available the To, Input, From, the Response Payload of the transaction in the payload, otherwise it returns an error
func getTransactionInformation(payload *common.Payload) (string, string, string, *peer.ChaincodeAction, error) {
	txActions := &peer.Transaction{}
	err := proto.Unmarshal(payload.GetData(), txActions)
	if err != nil {
		return "", "", "", nil, err
	}

	ccPropPayload, respPayload, err := getPayloads(txActions.GetActions()[0])
	if err != nil {
		return "", "", "", nil, fmt.Errorf("Failed to unmarshal transaction: %s", err)
	}

	invokeSpec := &peer.ChaincodeInvocationSpec{}
	err = proto.Unmarshal(ccPropPayload.GetInput(), invokeSpec)
	if err != nil {
		return "", "", "", nil, fmt.Errorf("Failed to unmarshal transaction: %s", err)
	}

	// callee, input data is standard case, also handle getcode & account cases
	args := invokeSpec.GetChaincodeSpec().GetInput().Args

	if len(args) != 2 || string(args[0]) == "getCode" {
		// no more data available to fill the transaction
		return "", "", "", respPayload, nil
	}

	sigHdr := &common.SignatureHeader{}
	if err := proto.Unmarshal(payload.GetHeader().GetSignatureHeader(), sigHdr); err != nil {
		return "", "", "", nil, fmt.Errorf("Failed unmarshaling signature header: %s", err)
	}

	from, err := address.IdentityToAddr(sigHdr.GetCreator())
	if err != nil {
		return "", "", "", nil, fmt.Errorf("Failed generating from address: %s", err)
	}

	// At this point, this is either an EVM Contract Deploy,
	// or an EVM Contract Invoke. We don't care about the
	// specific case, fill in the fields directly.

	// First arg is to and second arg is the input data
	return string(args[0]), string(args[1]), "0x" + hex.EncodeToString(from), respPayload, nil
}

// findTransaction takes in the txId and  block data from block.GetData().GetData() where block is of type *common.Block
// It returns the index of the transaction, transaction payload, otherwise it returns an error
func findTransaction(txID string, blockData [][]byte) (string, *common.Payload, error) {
	for index, transactionData := range blockData {
		if transactionData == nil { // can a data be empty? Is this an error?
			continue
		}

		payload, chdr, err := getChannelHeaderandPayloadFromTransactionData(transactionData)
		if err != nil {
			return "", nil, err
		}

		// early exit to try next transaction
		if txID != chdr.TxId {
			// transaction does not match, go to next
			continue
		}

		return "0x" + strconv.FormatUint(uint64(index), 16), payload, nil
	}

	return "", nil, nil
}

func fabricEventToEVMLogs(logger *zap.SugaredLogger, events []byte, blocknumber, txhash, txindex, blockhash string,
	af types.AddressFilter, tf types.TopicsFilter) ([]types.Log, error) {
	if len(events) == 0 {
		return nil, nil
	}

	chaincodeEvent := &peer.ChaincodeEvent{}
	err := proto.Unmarshal(events, chaincodeEvent)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode chaincode event")
	}

	var eventMsgs []event.Event
	err = json.Unmarshal(chaincodeEvent.Payload, &eventMsgs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal chaincode event payload")
	}

	var txLogs []types.Log
LOG_EVENT:
	for i, logEvent := range eventMsgs {
		if len(af) != 0 {
			foundMatch := false
			// if no address, empty range, skipped, present but empty address field results in no match
			for _, address := range af {
				logger.Debugw("trying address match", "matcherAddress", address, "eventAddress", logEvent.Address)
				if logEvent.Address == address {
					foundMatch = true
					break
				}
			}
			if !foundMatch {
				continue LOG_EVENT // no match, move to next logEvent
			}
		}

		// If we have more matchers than things to match against, we cannot match.
		if len(tf) > len(logEvent.Topics) {
			continue LOG_EVENT
		}

		logger.Debug("checking for topics")
		// Check match for each topic. This implementation matches behavior we have observed from other implementations.
		//
		// [] "anything"
		// [A] "A in first position (and anything after)"
		// [null, B] "anything in first position AND B in second position (and anything after)"
		// [A, B] "A in first position AND B in second position (and anything after)"
		// [[A, B], [A, B]] "(A OR B) in first position AND (A OR B) in second position (and anything after)"
		//
		// null matchers can be used to force an event to have at least that many topics
		// [] "anything"
		// [null] "anything with at least one topic"
		// [null, null] "anything with two or more topics"
		for i, topicFilter := range tf {
			// if filter is empty it matches automatically.
			if len(topicFilter) == 0 {
				continue
			}

			eventTopic := logEvent.Topics[i]
			foundMatch := false
			for _, topic := range topicFilter {
				logger.Debugw("matching Topic ", "matcherTopic", topic, "eventTopic", eventTopic)
				if topic == eventTopic || topic == "" {
					foundMatch = true
					break
				}
			}
			if !foundMatch {
				// if we didn't find a match, no use in checking any of the other topics
				continue LOG_EVENT
			}
		}
		logger.Debug("finished checking for topics")

		var topics []string
		for _, topic := range logEvent.Topics {
			topics = append(topics, "0x"+topic)
		}
		log := types.Log{
			Address:     "0x" + logEvent.Address,
			Topics:      topics,
			BlockNumber: blocknumber,
			TxHash:      txhash,
			TxIndex:     txindex,
			BlockHash:   blockhash,
			Index:       "0x" + strconv.FormatUint(uint64(i), 16),
		}

		if logEvent.Data != "" {
			log.Data = "0x" + logEvent.Data
		}

		txLogs = append(txLogs, log)
	}
	return txLogs, nil
}

func getChannelHeaderandPayloadFromTransactionData(transactionData []byte) (*common.Payload, *common.ChannelHeader, error) {
	env := &common.Envelope{}
	if err := proto.Unmarshal(transactionData, env); err != nil {
		return nil, nil, err
	}

	payload := &common.Payload{}
	if err := proto.Unmarshal(env.GetPayload(), payload); err != nil {
		return nil, nil, err
	}
	chdr := &common.ChannelHeader{}
	if err := proto.Unmarshal(payload.GetHeader().GetChannelHeader(), chdr); err != nil {
		return nil, nil, err
	}

	return payload, chdr, nil
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

	h := sha256.New()
	h.Write(result)
	return h.Sum(nil)
}
