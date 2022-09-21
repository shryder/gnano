package p2p

import (
	"io"
	"log"

	"github.com/Shryder/gnano/p2p/packets"
)

func (srv *P2P) HandleConfirmReq(reader io.Reader, header *packets.Header, peer *PeerNode) error {
	packet := make([]byte, header.PacketSize())
	_, err := io.ReadFull(reader, packet)
	if err != nil {
		return err
	}

	if header.Extension.BlockType() == packets.BLOCK_TYPE_NOT_A_BLOCK {
		type HashPair struct {
			First  []byte
			Second []byte
		}

		// Read hash pairs
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
	} else {

	}

	return nil
}
