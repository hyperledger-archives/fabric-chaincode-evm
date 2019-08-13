/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fab3

import (
	"encoding/hex"
	"time"

	"github.com/pkg/errors"

	"github.com/hyperledger/fabric-chaincode-evm/fab3/types"
)

type filterEntry interface {
	LastAccessTime() time.Time
	Filter(s *ethService) ([]interface{}, error)
}

type logsFilter struct {
	logArgs         *types.GetLogsArgs
	latestBlockSeen uint64
	lastAccessTime  time.Time
}

func (f *logsFilter) LastAccessTime() time.Time {
	return f.lastAccessTime
}

func (f *logsFilter) Filter(s *ethService) ([]interface{}, error) {
	s.logger.Debug("lastblockseen before filtering", f.latestBlockSeen)
	currentLatestBlockSeen, err := s.parseBlockNum("latest")
	if err != nil {
		return nil, err
	}
	s.logger.Debug("latest blockseen", currentLatestBlockSeen)

	var l []interface{}
	if f.latestBlockSeen < currentLatestBlockSeen {
		var logs []types.Log
		err = s.GetLogs(nil, f.logArgs, &logs)
		if err != nil {
			return nil, errors.Wrap(err, "GetLogs call failed")
		}
		l = make([]interface{}, 0, len(logs))
		for _, v := range logs {
			l = append(l, v)
		}
	}
	// update the filter
	f.lastAccessTime = time.Now()
	f.latestBlockSeen = currentLatestBlockSeen
	s.logger.Debug("returning %d logs", len(l))
	return l, nil
}

type newBlockFilter struct {
	latestBlockSeen uint64
	lastAccessTime  time.Time
}

func (f *newBlockFilter) LastAccessTime() time.Time {
	return f.lastAccessTime
}

func (f *newBlockFilter) Filter(s *ethService) ([]interface{}, error) {
	s.logger.Debug("lastblockseen before filtering", f.latestBlockSeen)
	blockNumber, err := s.parseBlockNum("latest")
	if err != nil {
		return nil, err
	}
	s.logger.Debug("latest blockseen", blockNumber)
	// iterate over blocks collecting the hashes
	blocksToCollect := blockNumber - f.latestBlockSeen
	// BlockFilter returns array of strings representing block numbers
	var blocks = make([]interface{}, 0, blocksToCollect)
	for i := blockNumber; i > f.latestBlockSeen; i-- {
		block, err := s.ledgerClient.QueryBlock(i)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to query the ledger")
		}

		blkHeader := block.GetHeader()

		blockHash := "0x" + hex.EncodeToString(blockHash(blkHeader))
		blocks = append(blocks, blockHash)
	}
	// update the filter
	f.lastAccessTime = time.Now()
	f.latestBlockSeen = blockNumber
	return blocks, nil
}
