/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package event

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hyperledger/burrow/execution/evm/events"
	"github.com/hyperledger/fabric/core/chaincode/shim"
)

type EventManager struct {
	stub       shim.ChaincodeStubInterface
	EventCache []events.EventDataLog
}

func NewEventManager(stub shim.ChaincodeStubInterface) *EventManager {
	return &EventManager{
		stub:       stub,
		EventCache: []events.EventDataLog{},
	}
}

func (evmgr *EventManager) Flush(eventName string) error {
	if len(evmgr.EventCache) == 0 {
		return nil
	}
	payload, err := json.Marshal(evmgr.EventCache)
	if err != nil {
		return fmt.Errorf("Failed to marshal event messages: %s", err.Error())
	}
	return evmgr.stub.SetEvent(eventName, payload)
}

func (evmgr *EventManager) Publish(ctx context.Context, message interface{}, tags map[string]interface{}) error {
	evID, ok := tags["EventID"].(string)
	if !ok {
		return fmt.Errorf("type mismatch: expected string but received %T", tags["EventID"])
	}

	msg, ok := message.(*events.EventDataLog)
	if !ok {
		return fmt.Errorf("type mismatch: expected *events.EventDataLog but received %T", message)
	}

	//Burrow EVM emits other events related to state (such as account call) as well, but we are only interested in log events
	if evID[0:3] == "Log" {
		evmgr.EventCache = append(evmgr.EventCache, *msg)
	}
	return nil
}
