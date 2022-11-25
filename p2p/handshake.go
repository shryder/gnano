package p2p

import (
	"bufio"
	"crypto/rand"
	"errors"
	"io"
	"net"

	"github.com/Shryder/gnano/p2p/networking"
	"github.com/Shryder/gnano/p2p/packets"
	"github.com/Shryder/gnano/types"
	"github.com/shryder/ed25519-blake2b"
)

func (srv *P2P) makeHandshake(conn net.Conn, reader *bufio.Reader) (*networking.PeerNode, error) {
	// Send handshake with random cookie that the peer will have to sign
	cookie := make([]byte, 32)
	rand.Read(cookie)

	var extension packets.HeaderExtension
	extension.SetQuery(true)

	handshake_cookie_header, handshake_body := srv.MakePacket(packets.PACKET_TYPE_NODE_ID_HANDSHAKE, extension, cookie)
	conn.Write(append(handshake_cookie_header.Serialize(), handshake_body...))

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

	var node_id types.Address
	copy(node_id[:], peer_account)

	peer := networking.NewPeerNode(conn, &node_id, false)

	extension = packets.HeaderExtension{}
	extension.SetResponse(true)

	signed_cookie := ed25519.Sign(srv.NodeKeyPair.PrivateKey, peer_cookie)
	err = srv.WriteToPeer(peer, packets.PACKET_TYPE_NODE_ID_HANDSHAKE, extension, srv.NodeKeyPair.PublicKey, signed_cookie)
	if err != nil {
		return nil, err
	}

	return peer, nil
}
