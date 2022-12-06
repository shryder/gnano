package packets

import (
	"encoding/binary"
	"fmt"
	"log"
)

type BlockType byte
type HeaderExtension [2]byte
type MessageType byte
type ProtocolVersion struct {
	Max   byte
	Using byte
	Min   byte
}

type Header struct {
	NetworkID       [2]byte
	ProtocolVersion ProtocolVersion
	MessageType     MessageType
	Extension       HeaderExtension
}

func (messageType MessageType) ToString() string {
	packet_names := []string{"INVALID_0", "NOT_A_BLOCK", "KEEP_ALIVE", "PUBLISH", "CONFIRM_REQ", "CONFIRM_ACK", "BULK_PULL", "BULK_PUSH", "FRONTIER_REQ", "INVALID_9", "NODE_ID_HANDSHAKE", "BULK_PULL_ACCOUNT", "TELEMETRY_REQ", "TELEMETRY_ACK"}

	message_type_int := uint(messageType)
	if message_type_int >= uint(len(packet_names)) {
		return fmt.Sprintf("INVALID_%d", message_type_int)
	}

	return packet_names[message_type_int]
}

func (header *Header) Serialize() []byte {
	extensions_le := binary.LittleEndian.AppendUint16(make([]byte, 0), header.Extension.Uint())

	return []byte{
		header.NetworkID[0],
		header.NetworkID[1],

		header.ProtocolVersion.Max,
		header.ProtocolVersion.Using,
		header.ProtocolVersion.Min,

		byte(header.MessageType),

		extensions_le[0],
		extensions_le[1],
	}
}

func (extension *HeaderExtension) Uint() uint16 {
	return binary.LittleEndian.Uint16(extension[:])
}

func (extension *HeaderExtension) BlockType() BlockType {
	return BlockType((extension.Uint() & 0x0f00) >> 8)
}

func (extension *HeaderExtension) SetBlockType(blockType BlockType) {
	u16 := extension.Uint()
	u16 &= 0x0f00
	u16 |= uint16(uint16(blockType) << 8)

	binary.LittleEndian.PutUint16(extension[:], u16)
}

func (extension *HeaderExtension) Count() uint {
	return uint((extension.Uint() & 0xf000) >> 12)
}

func (extension *HeaderExtension) SetCount(count uint16) {
	count = count << 12

	u16 := extension.Uint()
	u16 &= 0x0fff
	u16 |= count

	binary.LittleEndian.PutUint16(extension[:], u16)
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

func (extension *HeaderExtension) SetQuery(is_query bool) {
	u16 := extension.Uint()
	u16 &= 0xfffe
	u16 |= 0x0001

	binary.LittleEndian.PutUint16(extension[:], u16)
}

func (extension *HeaderExtension) IsResponse() bool {
	return extension.Uint()&2 == 1
}

func (extension *HeaderExtension) SetResponse(is_response bool) {
	u16 := extension.Uint()
	u16 &= 0xfffd
	u16 |= 0x0002

	binary.LittleEndian.PutUint16(extension[:], u16)
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
