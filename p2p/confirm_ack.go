package p2p

import (
	"fmt"
	"io"
	"log"

	"github.com/Shryder/gnano/p2p/networking"
	"github.com/Shryder/gnano/p2p/packets"
	"github.com/Shryder/gnano/types"
	"golang.org/x/crypto/blake2b"
)

func (srv *P2P) SendConfirmAck(reader packets.PacketReader, header *packets.Header, peer *networking.PeerNode) error {
	return nil
}

func calculateVoteHash(
	timestamp_and_vote_duration packets.TimestampAndVoteDuration,
	hashes *[]*types.Hash,
) ([]byte, error) {
	vote_hash, err := blake2b.New(32, nil)
	if err != nil {
		return nil, err
	}

	vote_hash.Write([]byte("vote "))
	for _, hash := range *hashes {
		vote_hash.Write((*hash)[:])
	}

	vote_hash.Write(timestamp_and_vote_duration[:])

	return vote_hash.Sum(nil), nil
}

func (srv *P2P) handleConfirmAckHashes(
	reader packets.PacketReader,
	header *packets.Header,
	peer *networking.PeerNode,
	vote_address *types.Address,
	signature *types.Signature,
	timestamp_and_vote_duration *packets.TimestampAndVoteDuration,
) error {
	hashes := make([]*types.Hash, header.Extension.Count())

	for i := 0; i < len(hashes); i++ {
		hash, err := reader.ReadHash()
		if err != nil {
			return fmt.Errorf("Encountered an error while reading confirm_ack hashes: %w", err)
		}

		hashes[i] = hash
	}

	srv.Workers.ConfirmAck.AddConfirmAckToQueue(peer, &packets.ConfirmAckByHashes{
		Account:                  vote_address,
		Signature:                signature,
		TimestampAndVoteDuration: timestamp_and_vote_duration,
		Hashes:                   &hashes,
	})

	return nil
}

func (srv *P2P) handleConfirmAckBlock(reader packets.PacketReader, header *packets.Header, peer *networking.PeerNode, vote_address *types.Address) error {
	block_data := make([]byte, header.Extension.BlockType().Size())
	_, err := io.ReadFull(reader, block_data)
	if err != nil {
		return err
	}

	_, err = reader.ReadBlock(header.Extension.BlockType())
	if err != nil {
		return err
	}

	log.Println("Discarded confirm_ack_by_block packet from", peer.NodeID.ToNodeAddress(), peer.Conn.RemoteAddr().String(), "voter:", vote_address.ToNanoAddress())

	return nil
}

func (srv *P2P) HandleConfirmAck(reader packets.PacketReader, header *packets.Header, peer *networking.PeerNode) error {
	account, err := reader.ReadAddress()
	if err != nil {
		return err
	}

	signature, err := reader.ReadSignature()
	if err != nil {
		return err
	}

	timestamp_and_vote_duration, err := reader.ReadTimestampAndVoteDuration()
	if err != nil {
		return err
	}

	if header.Extension.BlockType() != packets.BLOCK_TYPE_NOT_A_BLOCK {
		return srv.handleConfirmAckBlock(reader, header, peer, account)
	}

	return srv.handleConfirmAckHashes(reader, header, peer, account, signature, timestamp_and_vote_duration)
}
