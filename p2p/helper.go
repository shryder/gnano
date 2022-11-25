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

func (srv *P2P) MakePacket(message_type byte, extension packets.HeaderExtension, data ...[]byte) (packets.Header, []byte) {
	// Build packet header
	header := packets.Header{
		NetworkID:       [2]byte{srv.Config.NetworkId[0], srv.Config.NetworkId[1]},
		ProtocolVersion: packets.ProtocolVersion{Max: 18, Using: 18, Min: 18},
		MessageType:     packets.MessageType(message_type),
		Extension:       extension,
	}

	// join all data fields
	packet := make([]byte, 0)
	for _, field := range data {
		packet = append(packet, field...)
	}

	return header, packet
}
