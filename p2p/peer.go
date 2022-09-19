package p2p

import (
	"net"

	"github.com/Shryder/gnano/types"
)

type PeerNode struct {
	Conn   net.Conn
	NodeID *types.Address
}
