package p2p

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/Shryder/gnano/p2p/networking"
	"github.com/Shryder/gnano/p2p/packets"
)

func (srv *P2P) HandleKeepAlive(reader packets.PacketReader, header *packets.Header, peer *networking.PeerNode) error {
	message := make([]byte, 8*(16+2))
	_, err := io.ReadFull(reader, message)
	if err != nil {
		return errors.New("Error reading message from peer: " + err.Error())
	}

	peer_count := int(len(message) / 18)
	var peers []string

	for i := 0; i < peer_count*18; i += 18 {
		ip := net.IP(message[i : i+16])
		if ip.DefaultMask() != nil {
			port := binary.LittleEndian.Uint16(message[i+16 : i+16+2])
			peers = append(peers, ip.String()+":"+fmt.Sprintf("%d", port))
		}
	}

	srv.Database.Backend.AddNodeIPs(peers)

	// Write back the keepalive
	return srv.WriteToPeer(peer, packets.PACKET_TYPE_KEEPALIVE, packets.HeaderExtension{}, message)
}
