package p2p

import (
	"fmt"
	"io"
	"log"

	"github.com/Shryder/gnano/p2p/networking"
	"github.com/Shryder/gnano/p2p/packets"
	"github.com/Shryder/gnano/types"
	"github.com/shryder/ed25519-blake2b"
	"golang.org/x/crypto/blake2b"
)

func (srv *P2P) SendConfirmAck(reader packets.PacketReader, header *packets.Header, peer *networking.PeerNode) error {
	return nil
}

func calculateVoteHash(
	timestamp_and_vote_duration []byte,
	hashes *[][]byte,
) ([]byte, error) {
	vote_hash, err := blake2b.New(32, nil)
	if err != nil {
		return nil, err
	}

	vote_hash.Write([]byte("vote "))
	for _, hash := range *hashes {
		vote_hash.Write(hash)
	}

	vote_hash.Write(timestamp_and_vote_duration)

	return vote_hash.Sum(nil), nil
}

func (srv *P2P) handleConfirmAckHashes(
	reader packets.PacketReader,
	header *packets.Header,
	peer *networking.PeerNode,
	vote_address *types.Address,
	signature *types.Signature,
	timestamp_and_vote_duration []byte,
) error {
	hashes := make([][]byte, header.Extension.Count())

	for i := 0; i < len(hashes); i++ {
		hash, err := reader.ReadHash()
		if err != nil {
			return fmt.Errorf("Encountered an error while reading confirm_ack hashes: %w", err)
		}

		hashes[i] = hash[:]
	}

	log.Println(peer.NodeID.ToNodeAddress(), "sent us confirmation for", len(hashes), "blocks")

	// Validate signature
	vote_hash, err := calculateVoteHash(timestamp_and_vote_duration, &hashes)
	if err != nil {
		return fmt.Errorf("Error calculating vote hash: %w", err)
	}

	is_vote_signature_valid := ed25519.Verify(vote_address.ToPublicKey(), vote_hash, signature[:])
	if !is_vote_signature_valid {
		return fmt.Errorf("Received invalid confirm_ack signature from %s %s for %d blocks", peer.NodeID.ToNodeAddress(), peer.NodeID.ToNanoAddress(), len(hashes))
	}

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

	log.Println("Discarded confirm_ack by block packet from", peer.NodeID.ToNodeAddress(), "voter:", vote_address.ToNanoAddress())

	return nil
}

func (srv *P2P) HandleConfirmAck(reader packets.PacketReader, header *packets.Header, peer *networking.PeerNode) error {
	log.Println("Received confirm ack from", peer.NodeID.ToNodeAddress())

	account, err := reader.ReadAddress()
	if err != nil {
		return err
	}

	signature, err := reader.ReadSignature()
	if err != nil {
		return err
	}

	timestamp_and_vote_duration := make([]byte, 8)
	_, err = io.ReadFull(reader, timestamp_and_vote_duration)
	if err != nil {
		return err
	}

	if header.Extension.BlockType() != packets.BLOCK_TYPE_NOT_A_BLOCK {
		return srv.handleConfirmAckBlock(reader, header, peer, account)
	}

	return srv.handleConfirmAckHashes(reader, header, peer, account, signature, timestamp_and_vote_duration)
}
