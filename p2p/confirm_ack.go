package p2p

import (
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/Shryder/gnano/p2p/networking"
	"github.com/Shryder/gnano/p2p/packets"
	"github.com/Shryder/gnano/types"
	"github.com/Shryder/gnano/utils"
	"github.com/shryder/ed25519-blake2b"
	"golang.org/x/crypto/blake2b"
)

func (srv *P2P) SendConfirmAck(peer *networking.PeerNode, hashPairs []*packets.HashPair) error {
	pairs_bytes := make([][]byte, len(hashPairs))
	pairs_bytes_flat := make([]byte, 0)
	all_hashes := make([]types.Hash, 0)
	for i, pair := range hashPairs {
		pairs_bytes[i] = append(pair.Hash[:], pair.Root[:]...)
		pairs_bytes_flat = append(pairs_bytes_flat, pairs_bytes[i]...)
		all_hashes = append(all_hashes, *pair.Hash, *pair.Root)
	}

	if len(all_hashes) > 16 {
		return errors.New("can't confirm_ack more than 16 hash pairs")
	}

	log.Println("sending confirm_ack on", utils.HashPairToString(pairs_bytes), "to", peer.Alias)

	timestamp_and_vote_duration := packets.TimestampAndVoteDuration{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	vote_hash, err := calculateVoteHash(timestamp_and_vote_duration, &all_hashes)
	if err != nil {
		log.Println("Error calculating vote hash:", err)

		return err
	}

	vote_signature := ed25519.Sign(srv.NodeKeyPair.PrivateKey, vote_hash)
	var extension packets.HeaderExtension

	extension.SetBlockType(packets.BLOCK_TYPE_NOT_A_BLOCK)
	extension.SetCount(uint16(len(all_hashes)))

	return srv.WriteToPeer(peer, packets.PACKET_TYPE_CONFIRM_ACK, extension, srv.NodeKeyPair.PublicKey[:], vote_signature, timestamp_and_vote_duration[:], pairs_bytes_flat)
}

func calculateVoteHash(
	timestamp_and_vote_duration packets.TimestampAndVoteDuration,
	hashes *[]types.Hash,
) ([]byte, error) {
	vote_hash, err := blake2b.New(32, nil)
	if err != nil {
		return nil, err
	}

	vote_hash.Write([]byte("vote "))
	for _, hash := range *hashes {
		vote_hash.Write(hash[:])
	}

	vote_hash.Write(timestamp_and_vote_duration[:])

	return vote_hash.Sum(nil), nil
}

func calculateVoteHashPtrs(
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
	hashes_str := fmt.Sprintf("%d hashes: ", len(hashes))

	for i := 0; i < len(hashes); i++ {
		hash, err := reader.ReadHash()
		if err != nil {
			return fmt.Errorf("encountered an error while reading confirm_ack hashes: %w", err)
		}

		hashes[i] = hash
		hashes_str += hash.ToHexString() + ", "
	}

	srv.PeersManager.LogPacket(peer, *header, []byte(hashes_str), true)

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

	log.Println("Discarded confirm_ack_by_block packet from", peer.NodeID.ToNodeAddress(), peer.Alias, "voter:", vote_address.ToNanoAddress())

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
