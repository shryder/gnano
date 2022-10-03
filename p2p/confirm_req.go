package p2p

import (
	"log"

	"github.com/Shryder/gnano/p2p/networking"
	"github.com/Shryder/gnano/p2p/packets"
)

func (srv *P2P) HandleConfirmReqHashPairs(reader packets.PacketReader, header *packets.Header, peer *networking.PeerNode) error {
	pairs := make([]*packets.HashPair, header.Extension.Count())
	for i := uint(0); i < header.Extension.Count(); i++ {
		hash, err := reader.ReadHash()
		if err != nil {
			return err
		}

		root, err := reader.ReadHash()
		if err != nil {
			return err
		}

		pairs[i] = &packets.HashPair{
			Hash: hash,
			Root: root,
		}
	}

	log.Println("Received confirm_req with", len(pairs), "pairs")

	srv.Workers.ConfirmReq.ProcessHashPairs(peer, pairs)

	return nil
}

func (srv *P2P) HandleConfirmReqBlock(reader packets.PacketReader, header *packets.Header, peer *networking.PeerNode) error {
	log.Println("Received a block of type", header.Extension.BlockType())
	block, err := reader.ReadBlock(header.Extension.BlockType())
	if err != nil {
		return err
	}

	srv.Workers.ConfirmReq.ProcessBlock(peer, block)

	return nil
}

func (srv *P2P) HandleConfirmReq(reader packets.PacketReader, header *packets.Header, peer *networking.PeerNode) error {
	if header.Extension.BlockType() == packets.BLOCK_TYPE_NOT_A_BLOCK {
		return srv.HandleConfirmReqHashPairs(reader, header, peer)
	}

	return srv.HandleConfirmReqBlock(reader, header, peer)
}

func (srv *P2P) SendConfirmReq(peer *networking.PeerNode, pairs [][]byte) error {
	return peer.Write(srv.MakePacket(packets.PACKET_TYPE_CONFIRM_REQ, 0x0011, pairs...))
}
