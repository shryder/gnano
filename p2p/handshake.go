package p2p

import (
	"bufio"
	"crypto/rand"
	"errors"
	"io"
	"net"

	"github.com/Shryder/gnano/p2p/networking"
	"github.com/Shryder/gnano/types"
	"github.com/shryder/ed25519-blake2b"
)

func (srv *P2P) makeHandshake(conn net.Conn, reader *bufio.Reader) (*networking.PeerNode, error) {
	// Send handshake with random cookie that the peer will have to sign
	cookie := make([]byte, 32)
	rand.Read(cookie)

	conn.Write(srv.MakePacket(10, 0x01_00, cookie))

	header, err := srv.ReadHeader(reader)
	if err != nil {
		return nil, errors.New("Error reading packet header from peer: " + err.Error())
	}

	if header.MessageType != 10 {
		return nil, errors.New("Was expecting a node_id_handshake packet")
	}

	data := make([]byte, 128)
	_, err = io.ReadFull(reader, data)
	if err != nil {
		return nil, errors.New("Error reading data from peer: " + err.Error())
	}

	peer_cookie := data[0:32]
	peer_account := data[32:64]
	peer_signature := data[64:]

	valid_signature := ed25519.Verify(ed25519.PublicKey(peer_account), cookie, peer_signature)
	if !valid_signature {
		return nil, errors.New("Received invalid handshake signature from peer")
	}

	signed_cookie := ed25519.Sign(srv.NodeKeyPair.PrivateKey, peer_cookie)
	signed_handshake_response := srv.MakePacket(10, 0x02_00, srv.NodeKeyPair.PublicKey, signed_cookie)

	conn.Write(signed_handshake_response)

	var node_id types.Address
	copy(node_id[:], peer_account)

	return networking.NewPeerNode(conn, &node_id, false), nil
}
