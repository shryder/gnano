package packets

import (
	"encoding/binary"
	"log"
)

type BlockType byte
type HeaderExtension [2]byte
type Header struct {
	NetworkID       [2]byte
	ProtocolVersion struct {
		Max   byte
		Using byte
		Min   byte
	}
	MessageType byte
	Extension   HeaderExtension
}

func (extension *HeaderExtension) Uint() uint16 {
	return binary.LittleEndian.Uint16(extension[:])
}

func (extension *HeaderExtension) Count() uint {
	return uint((extension.Uint() & 0xf000) >> 12)
}

func (extension *HeaderExtension) BlockType() BlockType {
	return BlockType((extension.Uint() & 0x0f00) >> 8)
}

func (extension *HeaderExtension) TelemetrySize() uint {
	return uint(extension.Uint() & 0x3ff)
}

func (extension *HeaderExtension) ExtendedParamsPresent() bool {
	return uint(extension.Uint()&0x0001) == 1
}

func (extension *HeaderExtension) IsQuery() bool {
	return extension.Uint()&1 == 1
}

func (extension *HeaderExtension) IsResponse() bool {
	return extension.Uint()&2 == 1
}

func (blockType BlockType) Size() uint {
	switch byte(blockType) {
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

func (header *Header) PacketSize() uint {
	switch header.MessageType {
	case PACKET_TYPE_BULK_PUSH:
		return 0
	case PACKET_TYPE_TELEMETRY_REQ:
		return 0
	case PACKET_TYPE_FRONTIER_REQ:
		return 32 + 4 + 4
	case PACKET_TYPE_BULK_PULL_ACCOUNT:
		return 32 + 16 + 1
	case PACKET_TYPE_KEEPALIVE:
		return 8 * (16 + 2)
	case PACKET_TYPE_NODE_ID_HANDSHAKE:
		{
			size := uint(0)

			if header.Extension.IsQuery() {
				// Cookie only
				size += 32
			}

			if header.Extension.IsResponse() {
				// Account (32) + signed cookie (64)
				size += 32 + 64
			}

			return size
		}
	case PACKET_TYPE_CONFIRM_ACK:
		{
			// First 104 bytes contain common data shared by block votes and vote-by-hash votes
			// 32 (account) + 64 (signature) + 8 (timestamp_and_vote_duration)
			size := uint(104)

			if header.Extension.BlockType() == BLOCK_TYPE_NOT_A_BLOCK {
				// Only block hashes available
				size += uint(header.Extension.Count()) * 32
			} else {
				// Single entire block included in the packet
				size += header.Extension.BlockType().Size()
			}

			return size
		}
	case PACKET_TYPE_CONFIRM_REQ:
		{
			if header.Extension.BlockType() == BLOCK_TYPE_NOT_A_BLOCK {
				return uint(64 * header.Extension.Count())
			}

			return header.Extension.BlockType().Size()
		}
	case PACKET_TYPE_PUBLISH:
		return header.Extension.BlockType().Size()
	case PACKET_TYPE_TELEMETRY_ACK:
		return uint(header.Extension.TelemetrySize())

	}

	log.Println("Encountered invalid packet type:", header.MessageType)
	return 0
}
