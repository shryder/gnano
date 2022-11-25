package p2p

import (
	"github.com/Shryder/gnano/p2p/networking"
	"github.com/Shryder/gnano/p2p/packets"
	"github.com/Shryder/gnano/types"
)

const (
	BULK_PULL_ACCOUNT_HASH_AND_AMOUNT         = 0
	BULK_PULL_ACCOUNT_PENDING_ADDRESS_ONLY    = 1
	BULK_PULL_ACCOUNT_HASH_AMOUNT_AND_ADDRESS = 2
)

func (srv *P2P) SendBulkPullAccount(peer *networking.PeerNode, account types.Address, amount types.Amount, flag byte) error {
	return srv.WriteToPeer(peer,
		packets.PACKET_TYPE_BULK_PULL_ACCOUNT,
		packets.HeaderExtension{},

		account[:],
		amount.Bytes()[:],

		[]byte{flag},
	)
}
