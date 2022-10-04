package p2p

import (
	"io"
	"log"

	"github.com/Shryder/gnano/p2p/networking"
	"github.com/Shryder/gnano/p2p/packets"
)

func (srv *P2P) HandleBulkPullResponse(reader packets.PacketReader, peer *networking.PeerNode) error {
	for {
		block_type_byte := make([]byte, 1)
		_, err := io.ReadFull(reader, block_type_byte)
		if err != nil {
			return err
		}

		if block_type_byte[0] == packets.BLOCK_TYPE_NOT_A_BLOCK {
			log.Println("Reached bulk_pull_response end")
			break
		}

		block_type := packets.BlockType(block_type_byte[0])
		block, err := reader.ReadBlock(block_type)
		if err != nil {
			return err
		}

		log.Println("Read block of type:", block_type_byte[0], block.Hash.ToHexString())
	}

	return nil
}
