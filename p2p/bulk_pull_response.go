package p2p

import (
	"log"

	"github.com/Shryder/gnano/p2p/networking"
	"github.com/Shryder/gnano/p2p/packets"
	"github.com/Shryder/gnano/types"
)

func (srv *P2P) HandleBulkPullResponse(reader packets.PacketReader, peer *networking.PeerNode, our_start types.Hash, our_end types.Hash) error {
	count := uint(0)
	for {
		block_type_byte, err := reader.Buffer.ReadByte()
		if err != nil {
			return err
		}

		if block_type_byte == packets.BLOCK_TYPE_NOT_A_BLOCK {
			break
		}

		block, err := reader.ReadBlock(packets.BlockType(block_type_byte))
		if err != nil {
			return err
		}

		srv.UncheckedBlocksManager.Add(block)
		srv.BootstrapDataManager.FoundBlockBody(*block.Hash)
		count++
	}

	log.Println("Peer", peer.Alias, "returned", count, "blocks for our bulk_pull(", our_start.ToHexString(), ",", our_end.ToHexString(), ")")

	return nil
}
