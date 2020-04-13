/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

/*
Package types contains the types used to interact with the json-rpc
interface. It exists for users of fab3 types to use them without importing the
fabric protobuf definitions.
*/
package types

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

/*
Input types used as arguments to ethservice methods.
*/

const (
	// HexEncodedAddressLegnth is 20 bytes, which is 40 chars when hex-encoded
	HexEncodedAddressLegnth = 40
	// HexEncodedTopicLegnth is 32 bytes, which is 64 hex chars when hex-encoded
	HexEncodedTopicLegnth = 64
)

type EthArgs struct {
	To       string `json:"to"`
	From     string `json:"from"`
	Gas      string `json:"gas"`
	GasPrice string `json:"gasPrice"`
	Value    string `json:"value"`
	Data     string `json:"data"`
	Nonce    string `json:"nonce"`
}

type GetLogsArgs struct {
	FromBlock string        `json:"fromBlock,omitempty"`
	ToBlock   string        `json:"toBlock,omitempty"`
	Address   AddressFilter `json:"address,omitempty"`
	Topics    TopicsFilter  `json:"topics,omitempty"`
	BlockHash string        `json:"blockHash,omitempty"`
}

type AddressFilter []string // 20 Byte Addresses, OR'd together

type TopicFilter []string // 32 Byte Topics, OR'd together

type TopicsFilter []TopicFilter // TopicFilter, AND'd together

func (gla *GetLogsArgs) UnmarshalJSON(data []byte) error {
	type inputGetLogsArgs struct {
		FromBlock string      `json:"fromBlock"`
		ToBlock   string      `json:"toBlock"`
		Address   interface{} `json:"address"` // string or array of strings.
		Topics    interface{} `json:"topics"`  // array of strings, or array of array of strings
		BlockHash string      `json:"blockHash"`
	}
	var input inputGetLogsArgs
	if err := json.Unmarshal(data, &input); err != nil {
		return err
	}

	gla.FromBlock = strip0x(input.FromBlock)
	gla.ToBlock = strip0x(input.ToBlock)
	gla.BlockHash = strip0x(input.BlockHash)
	if gla.BlockHash != "" && (gla.FromBlock != "" || gla.ToBlock != "") {
		return errors.New("cannot provide BlockHash and (FromBlock or ToBlock), they are exclusive options")
	}

	if input.Address != nil {
		var af AddressFilter
		// DATA|Array, 20 Bytes - (optional) Contract address or a list of
		// addresses from which logs should originate.
		switch address := input.Address.(type) {
		case string:
			a, err := NewAddressFilter(address)
			if err != nil {
				return err
			}
			af = append(af, a...)
		case []interface{}:
			for i, address := range address {
				if singleAddress, ok := address.(string); ok {
					a, err := NewAddressFilter(singleAddress)
					if err != nil {
						return errors.Wrapf(err, "invalid address at position %d", i)
					}
					af = append(af, a...)
				}
			}
		default:
			return fmt.Errorf("badly formatted address field")
		}

		gla.Address = af
	}

	if input.Topics != nil {
		var tf TopicsFilter
		// handle the topics parsing
		if topics, ok := input.Topics.([]interface{}); !ok {
			return fmt.Errorf("topics must be slice")
		} else {
			for i, topic := range topics {
				if singleTopic, ok := topic.(string); ok {
					f, err := NewTopicFilter(singleTopic)
					if err != nil {
						return errors.Wrapf(err, "invalid topic at position %d", i)
					}
					tf = append(tf, f)
				} else if multipleTopic, ok := topic.([]interface{}); ok {
					var mtf TopicFilter
					for _, singleTopic := range multipleTopic {
						if stringTopic, ok := singleTopic.(string); ok {
							f, err := NewTopicFilter(stringTopic)
							if err != nil {
								return errors.Wrapf(err, "invalid topic at position %d", i)
							}
							mtf = append(mtf, f...)
						} else if singleTopic == nil {
							f := TopicFilter{""}
							mtf = append(mtf, f...)
						} else {
							return fmt.Errorf("all topics must be strings")
						}
					}
					tf = append(tf, mtf)
				} else if topic == nil {
					f := TopicFilter{""}
					tf = append(tf, f)
				} else {
					return fmt.Errorf("incorrect topics format %q", topic)
				}
			}
		}

		gla.Topics = tf
	}

	return nil
}

// NewAddressFilter takes a string and checks that is the correct length to
// represent a topic and strips the 0x
func NewAddressFilter(s string) (AddressFilter, error) {
	s = strip0x(s)
	if len(s) != HexEncodedAddressLegnth {
		return nil, fmt.Errorf("address in wrong format, need 40 chars prefixed with '0x', got %d chars for %q", len(s), s)
	}
	return AddressFilter{strings.ToLower(s)}, nil
}

// NewTopicFilter takes a string and checks that is the correct length to
// represent a topic and strips the 0x
func NewTopicFilter(s string) (TopicFilter, error) {
	s = strip0x(s)
	if len(s) != HexEncodedTopicLegnth {
		return nil, fmt.Errorf("topic in wrong format, need 64 chars prefixed with '0x', got %d for %q", len(s), s)
	}
	return TopicFilter{s}, nil
}

func NewTopicsFilter(tf ...TopicFilter) TopicsFilter {
	return tf
}

/*
Output types used as return values from ethservice methods.
*/

type TxReceipt struct {
	TransactionHash   string `json:"transactionHash"`
	TransactionIndex  string `json:"transactionIndex"`
	BlockHash         string `json:"blockHash"`
	BlockNumber       string `json:"blockNumber"`
	ContractAddress   string `json:"contractAddress,omitempty"`
	GasUsed           int    `json:"gasUsed"`
	CumulativeGasUsed int    `json:"cumulativeGasUsed"`
	To                string `json:"to"`
	Logs              []Log  `json:"logs"`
	Status            string `json:"status"`
	From              string `json:"from"`
}

type Log struct {
	Address     string   `json:"address"`
	Topics      []string `json:"topics"`
	Data        string   `json:"data,omitempty"`
	BlockNumber string   `json:"blockNumber"`
	TxHash      string   `json:"transactionHash"`
	TxIndex     string   `json:"transactionIndex"`
	BlockHash   string   `json:"blockHash"`
	Index       string   `json:"logIndex"`
}

// MarshalJSON will json marshal the transaction receipt and set Logs to an empty []Log if none returned
func (txReceipt *TxReceipt) MarshalJSON() ([]byte, error) {
	receipt := *txReceipt

	if receipt.Logs == nil {
		receipt.Logs = []Log{}
	}

	return json.Marshal(receipt)
}

// Transaction represents an ethereum evm transaction.
//
// https://github.com/ethereum/wiki/wiki/JSON-RPC#returns-28
type Transaction struct { // object, or null when no transaction was found:
	BlockHash   string `json:"blockHash"`   // DATA, 32 Bytes - hash of the block where this transaction was in. null when its pending.
	BlockNumber string `json:"blockNumber"` // QUANTITY - block number where this transaction was in. null when its pending.
	To          string `json:"to"`          // DATA, 20 Bytes - address of the receiver. null when its a contract creation transaction.
	// From is generated by EVM Chaincode. Until account generation
	// stabilizes, we are not returning a value.
	//
	// From can be gotten from the Signature on the Transaction Envelope
	//
	From             string `json:"from"`             // DATA, 20 Bytes - address of the sender.
	Input            string `json:"input"`            // DATA - the data send along with the transaction.
	TransactionIndex string `json:"transactionIndex"` // QUANTITY - integer of the transactions index position in the block. null when its pending.
	Hash             string `json:"hash"`             // DATA, 32 Bytes - hash of the transaction.
	GasPrice         string `json:"gasPrice"`         // QUANTITY - gas price provided by the sender in Wei.
	Value            string `json:"value"`            // QUANTITY - value transferred in Wei.
}

// MarshalJSON will json marshal the tx object as well as setting the Gas Price and Value fields as Ox0
func (tx *Transaction) MarshalJSON() ([]byte, error) {
	type Alias Transaction
	temp := (Alias)(*tx)
	temp.GasPrice = "0x0"
	temp.Value = "0x0"

	return json.Marshal(temp)
}

// Block is an eth return struct
// defined https://github.com/ethereum/wiki/wiki/JSON-RPC#returns-26
type Block struct {
	BlockData
	Transactions []interface{} `json:"transactions"` // transactions: Array - Array of transaction objects, or 32 Bytes transaction hashes depending on the last given parameter.
}

// BlockData contains the fields that are constant in a Block object
type BlockData struct {
	Number     string `json:"number"`     // number: QUANTITY - the block number. null when its pending block.
	Hash       string `json:"hash"`       // hash: DATA, 32 Bytes - hash of the block. null when its pending block.
	ParentHash string `json:"parentHash"` // parentHash: DATA, 32 Bytes - hash of the parent block.
	GasLimit   string `json:"gasLimit"`   // gasLimit: QUANTITY - the maximum gas allowed in this block.
}

// MarshalJSON marshals the data differently based on whether
// transactions are full or just tx hashes
func (originalBlk *Block) MarshalJSON() ([]byte, error) {
	blk := *originalBlk
	blk.GasLimit = "0x0"

	if len(blk.Transactions) == 0 {
		return json.Marshal(blk)
	}

	if _, ok := blk.Transactions[0].(Transaction); !ok {
		return json.Marshal(blk)
	}

	txns := make([]Transaction, len(blk.Transactions))
	for i, txn := range blk.Transactions {
		txns[i] = txn.(Transaction)
	}

	temp := struct {
		BlockData
		Transactions []Transaction `json:"transactions"`
	}{
		BlockData:    blk.BlockData,
		Transactions: txns,
	}

	return json.Marshal(temp)
}

func strip0x(s string) string {
	//Not checking for malformed addresses just stripping `0x` prefix where applicable
	return strings.TrimPrefix(s, "0x")
}
