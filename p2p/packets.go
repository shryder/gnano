package p2p

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
)

const (
	PACKET_TYPE_INVALID           = 0x0
	PACKET_TYPE_NOT_A_BLOCK       = 0x1
	PACKET_TYPE_KEEPALIVE         = 0x2
	PACKET_TYPE_PUBLISH           = 0x3
	PACKET_TYPE_CONFIRM_REQ       = 0x4
	PACKET_TYPE_CONFIRM_ACK       = 0x5
	PACKET_TYPE_BULK_PULL         = 0x6
	PACKET_TYPE_BULK_PUSH         = 0x7
	PACKET_TYPE_FRONTIER_REQ      = 0x8
	PACKET_TYPE_NODE_ID_HANDSHAKE = 0x0a
	PACKET_TYPE_BULK_PULL_ACCOUNT = 0x0b
	PACKET_TYPE_TELEMETRY_REQ     = 0x0c
	PACKET_TYPE_TELEMETRY_ACK     = 0x0d
)

const (
	BLOCK_TYPE_INVALID     = 0x00
	BLOCK_TYPE_NOT_A_BLOCK = 0x01
	BLOCK_TYPE_SEND        = 0x02
	BLOCK_TYPE_RECEIVE     = 0x03
	BLOCK_TYPE_OPEN        = 0x04
	BLOCK_TYPE_CHANGE      = 0x05
	BLOCK_TYPE_STATE       = 0x06
)

const PROTOCOL_VERSION byte = 18

type HeaderExtension [2]byte
type PacketHeader struct {
	NetworkID       [2]byte
	ProtocolVersion struct {
		Max   byte
		Using byte
		Min   byte
	}
	MessageType byte
	Extension   HeaderExtension
}

type PacketBody struct {
	Buff bytes.Buffer
}

func (packetBody *PacketBody) WriteBE(data interface{}) {
	binary.Write(&packetBody.Buff, binary.BigEndian, data)
}

func (packetBody *PacketBody) WriteLE(data interface{}) {
	binary.Write(&packetBody.Buff, binary.LittleEndian, data)
}

func (srv *P2P) makeHeader(message_type byte, extension uint16) []byte {
	extensions_le := binary.BigEndian.AppendUint16(make([]byte, 0), extension)

	return []byte{
		srv.Config.NetworkId[0],
		srv.Config.NetworkId[1],

		18,
		18,
		18,

		message_type,

		extensions_le[0],
		extensions_le[1],
	}
}

func (srv *P2P) readHeader(reader io.Reader) (PacketHeader, error) {
	var header PacketHeader
	err := binary.Read(reader, binary.BigEndian, &header)
	if err != nil {
		return PacketHeader{}, err
	}

	if header.NetworkID[0] != srv.Config.NetworkId[0] || header.NetworkID[1] != srv.Config.NetworkId[1] {
		return PacketHeader{}, errors.New("Peer is on another network: " + string(header.NetworkID[0:2]))
	}

	return header, nil
}

func (srv *P2P) makePacket(message_type byte, extension uint16, data ...[]byte) []byte {
	packet := srv.makeHeader(message_type, extension)
	for _, field := range data {
		packet = append(packet, field...)
	}

	return packet
}

func (extension *HeaderExtension) Uint() uint16 {
	return binary.LittleEndian.Uint16(extension[:])
}

func (extension *HeaderExtension) Count() uint {
	return uint((extension.Uint() & 0xf000) >> 12)
}

func (extension *HeaderExtension) BlockType() byte {
	return byte((extension.Uint() & 0x0f00) >> 8)
}

func (extension *HeaderExtension) TelemetrySize() byte {
	return byte(extension.Uint() & 0x3ff)
}

func BlockTypeSize(block_type byte) uint {
	switch block_type {
	case BLOCK_TYPE_SEND:
		return 152
	case BLOCK_TYPE_RECEIVE:
		return 136
	case BLOCK_TYPE_OPEN:
		return 168
	case BLOCK_TYPE_CHANGE:
		return 136
	case BLOCK_TYPE_STATE:
		return 216
	}

	// TODO: handle invalid block type
	return 0
}
