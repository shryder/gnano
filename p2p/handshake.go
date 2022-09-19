package p2p

import (
	"bufio"
	"crypto/rand"
	"errors"
	"io"
	"log"
	"net"

	"github.com/Shryder/gnano/types"
	"github.com/shryder/ed25519-blake2b"
)

func (srv *P2P) loadLocalNodeID() (*ed25519.PublicKey, *ed25519.PrivateKey, error) {
	node_public_key, node_private_key, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, nil, err
	}

	return &node_public_key, &node_private_key, nil
}

func (srv *P2P) makeHandshake(conn net.Conn, reader *bufio.Reader) (*PeerNode, error) {
	node_public_key, node_private_key, err := srv.loadLocalNodeID()
	if err != nil {
		return nil, err
	}

	srv.NodeKeyPair = &NodeKeyPair{
		PrivateKey: node_private_key,
		PublicKey:  node_public_key,
	}

	// Send handshake with random cookie that the peer will have to sign
	cookie := make([]byte, 32)
	rand.Read(cookie)

	conn.Write(srv.makePacket(10, 0x01_00, cookie))

	header, err := srv.readHeader(reader)
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

	log.Println("Received header:", header)

	peer_cookie := data[0:32]
	peer_account := data[32:64]
	peer_signature := data[64:]

	valid_signature := ed25519.Verify(ed25519.PublicKey(peer_account), cookie, peer_signature)
	if !valid_signature {
		return nil, errors.New("Received invalid handshake signature from peer")
	}

	signed_cookie := ed25519.Sign(*node_private_key, peer_cookie)
	signed_handshake_response := srv.makePacket(10, 0x02_00, *node_public_key, signed_cookie)

	conn.Write(signed_handshake_response)

	var node_id types.Address
	copy(node_id[:], peer_account)

	return &PeerNode{
		Conn:   conn,
		NodeID: &node_id,
	}, nil
}
