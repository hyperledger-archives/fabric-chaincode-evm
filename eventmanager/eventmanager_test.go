/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package eventmanager_test

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	"github.com/hyperledger/burrow/binary"
	"github.com/hyperledger/burrow/crypto"
	"github.com/hyperledger/burrow/execution/exec"

	"github.com/hyperledger/fabric-chaincode-evm/event"
	"github.com/hyperledger/fabric-chaincode-evm/eventmanager"
	mocks "github.com/hyperledger/fabric-chaincode-evm/mocks/evmcc"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Event", func() {
	var (
		eventManager *eventmanager.EventManager
		mockStub     *mocks.MockStub
		addr         crypto.Address
		topics       []binary.Word256
		data         binary.HexBytes

		expectedTopics []string
		expectedData   string
	)

	BeforeEach(func() {
		mockStub = &mocks.MockStub{}
		eventManager = &eventmanager.EventManager{Stub: mockStub, EventCache: []event.Event{}}

		var err error
		addr, err = crypto.AddressFromBytes([]byte("000000000000addressz"))
		Expect(err).ToNot(HaveOccurred())

		topic1 := binary.RightPadWord256([]byte("topic-1z"))
		topic2 := binary.RightPadWord256([]byte("topic-2z"))
		topics = []binary.Word256{topic1, topic2}
		expectedTopics = []string{
			hex.EncodeToString(topic1.Bytes()),
			hex.EncodeToString(topic2.Bytes()),
		}

		data = []byte("sample-data")
		expectedData = hex.EncodeToString(data.Bytes())
	})

	Describe("Log", func() {
		var message *exec.LogEvent

		BeforeEach(func() {
			message = &exec.LogEvent{Address: addr, Data: data, Topics: topics}
		})

		It("appends the new message info into the eventCache", func() {
			originalLength := len(eventManager.EventCache)
			err := eventManager.Log(message)
			Expect(err).ToNot(HaveOccurred())
			newLength := len(eventManager.EventCache)
			Expect(newLength).To(Equal(originalLength + 1))
			Expect(eventManager.EventCache[newLength-1]).To(Equal(event.Event{
				Address: strings.ToLower(addr.String()),
				Data:    expectedData,
				Topics:  expectedTopics}))
		})
	})

	Describe("Call", func() {
		It("is a noop", func() {
			originalLength := len(eventManager.EventCache)

			err := eventManager.Call(&exec.CallEvent{}, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(eventManager.EventCache).To(HaveLen(originalLength))
		})
	})

	Describe("Flush", func() {
		var (
			message1 *exec.LogEvent
			message2 *exec.LogEvent
		)

		BeforeEach(func() {
			message1 = &exec.LogEvent{
				Address: addr,
			}
			message2 = &exec.LogEvent{
				Address: addr,
			}
		})

		Context("when a single event is emitted", func() {
			It("sets a new event with a single messageInfo object payload", func() {
				err := eventManager.Log(message1)
				Expect(err).ToNot(HaveOccurred())
				err = eventManager.Flush("Chaincode event")
				Expect(err).ToNot(HaveOccurred())

				messagePayloads := []event.Event{{Address: strings.ToLower(addr.String())}}
				expectedPayload, err := json.Marshal(messagePayloads)
				Expect(err).ToNot(HaveOccurred())

				Expect(mockStub.SetEventCallCount()).To(Equal(1))
				setEventName, setEventPayload := mockStub.SetEventArgsForCall(0)
				Expect(setEventName).To(Equal("Chaincode event"))
				Expect(setEventPayload).To(Equal(expectedPayload))
			})
		})

		Context("when multiple events are emitted", func() {
			It("sets a new event with a payload consisting of messageInfo objects marshaled together", func() {
				err := eventManager.Log(message1)
				Expect(err).ToNot(HaveOccurred())
				err = eventManager.Log(message2)
				Expect(err).ToNot(HaveOccurred())
				err = eventManager.Flush("Chaincode event")
				Expect(err).ToNot(HaveOccurred())

				messagePayloads := []event.Event{
					{Address: strings.ToLower(addr.String())},
					{Address: strings.ToLower(addr.String())},
				}
				expectedPayload, err := json.Marshal(messagePayloads)
				Expect(err).ToNot(HaveOccurred())

				Expect(mockStub.SetEventCallCount()).To(Equal(1))
				setEventName, setEventPayload := mockStub.SetEventArgsForCall(0)
				Expect(setEventName).To(Equal("Chaincode event"))
				Expect(setEventPayload).To(Equal(expectedPayload))
			})
		})

		Context("when the event name is invalid (empty string)", func() {
			BeforeEach(func() {
				mockStub.SetEventReturns(errors.New("error: nil event name"))
			})

			It("returns an error", func() {
				err := eventManager.Log(message1)
				Expect(err).ToNot(HaveOccurred())
				err1 := eventManager.Log(message2)
				Expect(err1).ToNot(HaveOccurred())
				er := eventManager.Flush("")
				Expect(er).To(HaveOccurred())
			})
		})
	})
})
