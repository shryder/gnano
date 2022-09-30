package p2p

import (
	"io"
	"log"

	"github.com/Shryder/gnano/p2p/packets"
)

func (srv *P2P) HandleConfirmReqHashPairs(reader io.Reader, header *packets.Header, peer *PeerNode) error {
	pairs := make([][]byte, header.Extension.Count())
	for i := uint(0); i < header.Extension.Count(); i++ {
		pair := make([]byte, 64)
		_, err := io.ReadFull(reader, pair)
		if err != nil {
			return err
		}

		pairs[i] = pair
	}

	log.Println("Hash pairs:", pairs)

	return nil
}

func (srv *P2P) HandleConfirmReqBlock(reader io.Reader, header *packets.Header, peer *PeerNode) error {
	block_data := make([]byte, header.Extension.BlockType().Size())
	_, err := io.ReadFull(reader, block_data)
	if err != nil {
		return err
	}

	log.Println("Received a block of type", header.Extension.BlockType())

	return nil
}

func (srv *P2P) HandleConfirmReq(reader io.Reader, header *packets.Header, peer *PeerNode) error {
	if header.Extension.BlockType() == packets.BLOCK_TYPE_NOT_A_BLOCK {
		return srv.HandleConfirmReqHashPairs(reader, header, peer)
	}

	return srv.HandleConfirmReqBlock(reader, header, peer)
}
