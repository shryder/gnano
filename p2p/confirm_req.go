package p2p

import (
	"io"
)

func (srv *P2P) HandleConfirmReq(reader io.Reader, header *PacketHeader, peer *PeerNode) error {
	extension_block_type := header.Extension.BlockType()
	extension_count := header.Extension.Count()

	packet_size := uint(0)
	if extension_block_type == BLOCK_TYPE_NOT_A_BLOCK {
		packet_size = uint(64 * extension_count)
	} else {
		// extension_count should be 1 in this case
		packet_size = BlockTypeSize(extension_block_type)
	}

	packet := make([]byte, packet_size)
	_, err := io.ReadFull(reader, packet)
	if err != nil {
		return err
	}

	return nil
}
