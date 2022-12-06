package p2p

import (
	"log"

	"github.com/Shryder/gnano/p2p/networking"
	"github.com/Shryder/gnano/p2p/packets"
	"github.com/Shryder/gnano/types"
	"github.com/Shryder/gnano/utils"
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

	srv.Workers.ConfirmReq.AddConfirmReqHashPairsToQueue(peer, pairs)

	return nil
}

// Will be deprecated soon
func (srv *P2P) HandleConfirmReqBlock(reader packets.PacketReader, header *packets.Header, peer *networking.PeerNode) error {
	log.Println("Received a block of type", header.Extension.BlockType())

	_, err := reader.ReadBlock(header.Extension.BlockType())
	if err != nil {
		return err
	}

	// Ignore

	return nil
}

func (srv *P2P) HandleConfirmReq(reader packets.PacketReader, header *packets.Header, peer *networking.PeerNode) error {
	if header.Extension.BlockType() == packets.BLOCK_TYPE_NOT_A_BLOCK {
		return srv.HandleConfirmReqHashPairs(reader, header, peer)
	}

	return srv.HandleConfirmReqBlock(reader, header, peer)
}

func FlattenHashPairs(pairs []types.HashPair) [][]byte {
	flattened := make([][]byte, len(pairs))
	for i, pair := range pairs {
		flattened[i] = pair.ToSlice()
	}

	return flattened
}

// Try sending confirm_req to a subset of live peers
func (srv *P2P) SendConfirmReqToPeers(pairs []types.HashPair) error {
	subsetCount := srv.PeersManager.GetSubsetOfLivePeers()

	srv.PeersManager.PeersMutex.Lock()
	for _, peer := range srv.PeersManager.LivePeers {
		if subsetCount == 0 {
			break
		}

		go func(peer *networking.PeerNode) {
			err := srv.SendConfirmReq(peer, FlattenHashPairs(pairs))
			if err != nil {
				log.Println("Error sending confirm_req to peer:", peer.Alias, err)
			}
		}(peer)

		subsetCount--
	}
	srv.PeersManager.PeersMutex.Unlock()

	return nil
}

func (srv *P2P) SendConfirmReq(peer *networking.PeerNode, pairs [][]byte) error {
	log.Println("Requesting confirm_req for", len(pairs), "pairs:", utils.HashPairToString(pairs), "from", peer.Alias)

	var extension packets.HeaderExtension
	extension.SetBlockType(packets.BLOCK_TYPE_NOT_A_BLOCK)
	extension.SetCount(uint16(len(pairs)))

	return srv.WriteToPeer(peer, packets.PACKET_TYPE_CONFIRM_REQ, extension, pairs...)
}
