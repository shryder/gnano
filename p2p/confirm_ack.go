package p2p

import (
	"io"

	"github.com/Shryder/gnano/p2p/packets"
)

func (srv *P2P) SendConfirmAck(reader io.Reader, header *packets.Header, peer *PeerNode) error {
	// TODO
	return nil
}
