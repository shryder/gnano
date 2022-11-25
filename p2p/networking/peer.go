package networking

import (
	"net"
	"sync"

	"github.com/Shryder/gnano/types"
)

type PeerNode struct {
	Alias      string
	Conn       net.Conn
	writeMutex sync.Mutex

	BootstrapConnection bool

	NodeID *types.Address
}

func NewPeerNode(conn net.Conn, nodeId *types.Address, bootstrap_connection bool) *PeerNode {
	alias := conn.RemoteAddr().String()
	if nodeId != nil {
		alias += "(" + nodeId.ToNodeAddress() + ")"
	} else {
		alias += "(bootstrap)"
	}

	return &PeerNode{
		Alias:               alias,
		Conn:                conn,
		NodeID:              nodeId,
		BootstrapConnection: bootstrap_connection,

		writeMutex: sync.Mutex{},
	}
}

func (peer *PeerNode) Write(p []byte) error {
	peer.writeMutex.Lock()
	_, err := peer.Conn.Write(p)
	peer.writeMutex.Unlock()

	return err
}
