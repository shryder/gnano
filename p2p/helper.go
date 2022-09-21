package p2p

import (
	"encoding/binary"
	"errors"
	"io"

	"github.com/Shryder/gnano/p2p/packets"
)

func (srv *P2P) ReadHeader(reader io.Reader) (packets.Header, error) {
	var header packets.Header
	err := binary.Read(reader, binary.BigEndian, &header)
	if err != nil {
		return packets.Header{}, err
	}

	if header.NetworkID[0] != srv.Config.NetworkId[0] || header.NetworkID[1] != srv.Config.NetworkId[1] {
		return packets.Header{}, errors.New("Peer is on another network: " + string(header.NetworkID[0:2]))
	}

	return header, nil
}

func (srv *P2P) MakePacket(message_type byte, extension uint16, data ...[]byte) []byte {
	// Build packet header
	extensions_le := binary.BigEndian.AppendUint16(make([]byte, 0), extension)
	packet := []byte{
		srv.Config.NetworkId[0],
		srv.Config.NetworkId[1],

		18,
		18,
		18,

		message_type,

		extensions_le[0],
		extensions_le[1],
	}

	// Append all data fields
	for _, field := range data {
		packet = append(packet, field...)
	}

	return packet
}
