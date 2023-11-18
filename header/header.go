package header

import (
	"encoding/binary"
	"errors"
	"fmt"
)

type HeaderVersion uint8
type MessageType uint8
type MessageSubType uint8

// Header Length
const (
	Version HeaderVersion = 1
	Len                   = 16
)

// A custom header allows for varying payloads to be used and encoded/decoded properly
// The type and subtype fields can be used to know exactly what type of protobuf message is expected,
// preventing the use and complexities that come with using oneof
// However, this strategy is best used with a clever composition of protobuf messages.
// An example of this is using a reusable nested field that can be valid for multiple message types to reduce complexity
// and message types, which each need to be generated by the protoc compiler and take up code space

// Message Types
// // Data signifies data packet, no protobuf message
// // Control signifies protobuf message
// // Test signifies protobuf test messages
//
//go:generate stringer -type=MessageType
const (
	Punch     MessageType = 0
	Handshake MessageType = 1
	Data      MessageType = 2
	Keepalive MessageType = 3
	ClosePeer MessageType = 4
	Test      MessageType = 5
)

// Control Subtypes
//
//go:generate stringer -type=MessageSubType
const (
	None      MessageSubType = 0
	Initiator MessageSubType = 1
	Responder MessageSubType = 2
	Request   MessageSubType = 3
	Reply     MessageSubType = 4
)

// Test SubTypes
// Signifies type of protobuf test message
const (
	TestRequest MessageSubType = 0
	TestReply   MessageSubType = 1
)

type Header struct {
	Version    HeaderVersion
	Type       MessageType
	SubType    MessageSubType
	ID         uint32
	MsgCounter uint64
	Unused     uint8
}

var (
	MessageTypeMap = map[MessageType]string{
		Punch:     "punch",
		Handshake: "handshake",
		Data:      "data",
		Keepalive: "keepalive",
		ClosePeer: "closepeer",
		Test:      "test",
	}

	SubTypeMapNone = map[MessageSubType]string{None: "none"}
	SubTypeMap     = map[MessageType]*map[MessageSubType]string{
		Punch: &SubTypeMapNone,
		Data:  &SubTypeMapNone,
		Handshake: {
			Initiator: "initiator",
			Responder: "responder",
		},
		Keepalive: {
			Request: "keepalive request",
			Reply:   "keepalive reply",
		},
		ClosePeer: &SubTypeMapNone,
		Test: {
			TestRequest: "test request",
			TestReply:   "test reply",
		},
	}
)

func (h *Header) Encode(b []byte, t MessageType, st MessageSubType, id uint32, counter uint64) ([]byte, error) {
	if h == nil {
		return nil, errors.New("header is nil")
	} else if cap(b) < Len {
		return nil, errors.New("provided byte array too small to encode header")
	}
	h.Version = Version
	h.Type = t
	h.SubType = st
	h.ID = id
	h.MsgCounter = counter
	h.Unused = 0

	return encodeBytes(b, h), nil
}

func encodeBytes(b []byte, h *Header) []byte {
	b = b[:Len]
	b[0] = byte(h.Version)
	b[1] = byte(h.Type)
	b[2] = byte(h.SubType)
	binary.BigEndian.PutUint32(b[3:7], h.ID)
	binary.BigEndian.PutUint64(b[7:15], h.MsgCounter)
	b[15] = h.Unused
	return b
}

func (h *Header) Parse(b []byte) error {
	if len(b) < Len {
		return errors.New("header length is too short")
	}
	h.Version = HeaderVersion(b[0])
	h.Type = MessageType(b[1])
	h.SubType = MessageSubType(b[2])
	h.ID = binary.BigEndian.Uint32(b[3:7])
	h.MsgCounter = binary.BigEndian.Uint64(b[7:15])
	h.Unused = b[15]

	if h.Version != Version {
		return errors.New("header version mismatch")
	} else if h.Unused != 0 {
		return errors.New("header unused field has non-nil value")
	}

	return nil
}

func (h *Header) String() string {
	if h == nil {
		return "nil"
	}

	return fmt.Sprintf("header: {version: %d type: %s subtype: %s index: %d msgcounter: %d unused: #%x", h.Version, h.TypeName(), h.SubTypeName(), h.Index(), h.Counter(), h.Unused)
}

func (h *Header) TypeName() string {
	if tn, found := MessageTypeMap[h.Type]; found {
		return tn
	}
	return "unknown type"
}

func (h *Header) SubTypeName() string {
	if n, ok := SubTypeMap[h.Type]; ok {
		if x, ok := (*n)[h.SubType]; ok {
			return x
		}
	}
	return "unknown subtype"
}

func (h *Header) Index() uint32 {
	if h == nil {
		return 0
	}
	return h.ID
}

func (h *Header) Counter() uint64 {
	if h == nil {
		return 0
	}
	return h.MsgCounter
}
