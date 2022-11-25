package p2p

import (
	"github.com/Shryder/gnano/p2p/networking"
	"github.com/Shryder/gnano/p2p/packets"
	"github.com/Shryder/gnano/types"
)

func (srv *P2P) SendBulkPull(peer *networking.PeerNode, start types.Hash, end types.Hash) error {
	return srv.WriteToPeer(
		peer,
		packets.PACKET_TYPE_BULK_PULL,
		packets.HeaderExtension{},

		start[:],
		end[:],
	)
}
