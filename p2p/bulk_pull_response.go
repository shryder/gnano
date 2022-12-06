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

		// log.Println("Received block with hash:", block.Hash.ToHexString())

		ledgerBlock := srv.Database.Backend.GetBlock(block.Hash)
		if ledgerBlock == nil {
			// Block is unknown, add to unchecked table and request votes from live peers
			srv.UncheckedBlocksManager.Add(block)
			srv.BootstrapDataManager.FoundBlockBody(*block.Hash)
			srv.Workers.ConfirmReq.RequestVotesOnTheseBlocks([][]byte{
				append(block.Hash[:], block.Root()[:]...),
			}, peer)
		}

		count++
	}

	log.Println("Peer", peer.Alias, "returned", count, "blocks for our bulk_pull(", our_start.ToHexString(), ",", our_end.ToHexString(), ")")

	return nil
}
