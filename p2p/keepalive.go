package p2p

import (
	"encoding/binary"
	"errors"
	"io"
	"log"
	"net"
)

func (srv *P2P) HandleKeepAlive(reader io.Reader, header *PacketHeader, peer *PeerNode) error {
	message := make([]byte, 8*(16+2))
	_, err := io.ReadFull(reader, message)
	if err != nil {
		return errors.New("Error reading from peer: " + err.Error())
	}

	type SuggestedPeer struct {
		IP   net.IP
		Port uint16
	}

	peer_count := int(len(message) / 18)
	var peers []SuggestedPeer

	for i := 0; i < peer_count*18; i += 18 {
		peers = append(peers, SuggestedPeer{
			IP:   net.IP(message[i : i+16]),
			Port: binary.LittleEndian.Uint16(message[i+16 : i+16+2]),
		})
	}

	log.Println("Suggested Peers (", len(peers), "):", peers)

	return nil
}
