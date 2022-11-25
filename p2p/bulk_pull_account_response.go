package p2p

import (
	"log"
	"math/big"

	"github.com/Shryder/gnano/p2p/networking"
	"github.com/Shryder/gnano/p2p/packets"
	"github.com/Shryder/gnano/types"
)

func (srv *P2P) HandleBulkPullAccountHashAndAmount(peer *networking.PeerNode, reader packets.PacketReader, frontier_hash *types.Hash, balance *types.Amount) (uint, error) {
	type BulkPullAccountHashAndAmountEntry struct {
		Hash   *types.Hash
		Amount *types.Amount
	}

	BATCH_SIZE := 32
	entries := make([]BulkPullAccountHashAndAmountEntry, 0)
	processed_count := uint(0)
	for {
		hash, err := reader.ReadHash()
		if err != nil {
			return processed_count, err
		}

		if hash.BigInt().Cmp(big.NewInt(0)) == 0 {
			break
		}

		amount, err := reader.ReadAmountLE()
		if err != nil {
			return processed_count, err
		}

		log.Println("Hash:", hash.ToHexString(), "Amount:", amount.String())
		entries = append(entries, BulkPullAccountHashAndAmountEntry{
			Hash:   hash,
			Amount: amount,
		})

		if len(entries) == BATCH_SIZE {
			entries = make([]BulkPullAccountHashAndAmountEntry, 0)
		}
	}

	return processed_count, nil
}

func (srv *P2P) HandleBulkPullAccountPendingAddressOnly(peer *networking.PeerNode, reader packets.PacketReader, frontier_hash *types.Hash, balance *types.Amount) error {
	entries := make([]*types.Hash, 0)
	for {
		hash, err := reader.ReadHash()
		if err != nil {
			return err
		}

		if hash.BigInt().Cmp(big.NewInt(0)) == 0 {
			break
		}

		log.Println("Hash:", hash.ToHexString())
		entries = append(entries, hash)
	}

	return nil
}

func (srv *P2P) HandleBulkPullAccountHashAmountAndAddress(peer *networking.PeerNode, reader packets.PacketReader, frontier_hash *types.Hash, balance *types.Amount) error {
	type BulkPullAccountHashAmountAndAddressEntry struct {
		Hash   *types.Hash
		Amount *types.Amount
		Source *types.Hash
	}

	entries := make([]BulkPullAccountHashAmountAndAddressEntry, 0)
	for {
		hash, err := reader.ReadHash()
		if err != nil {
			return err
		}

		if hash.BigInt().Cmp(big.NewInt(0)) == 0 {
			break
		}

		amount, err := reader.ReadAmountLE()
		if err != nil {
			return err
		}

		source, err := reader.ReadHash()
		if err != nil {
			return err
		}

		log.Println("Hash:", hash.ToHexString(), "Amount:", amount.String(), "Source:", source.ToHexString())
		entries = append(entries, BulkPullAccountHashAmountAndAddressEntry{
			Hash:   hash,
			Amount: amount,
			Source: source,
		})
	}

	return nil
}

// func (srv *P2P) HandleBulkPullAccountResponse(peer *networking.PeerNode, reader packets.PacketReader, flag byte) error {
// 	frontier_hash, err := reader.ReadHash()
// 	if err != nil {
// 		return err
// 	}

// 	balance, err := reader.ReadAmount()
// 	if err != nil {
// 		return err
// 	}

// 	if flag == BULK_PULL_ACCOUNT_HASH_AND_AMOUNT {
// 		return srv.HandleBulkPullAccountHashAndAmount(peer, reader, frontier_hash, balance)
// 	} else if flag == BULK_PULL_ACCOUNT_PENDING_ADDRESS_ONLY {
// 		return srv.HandleBulkPullAccountPendingAddressOnly(peer, reader, frontier_hash, balance)
// 	} else if flag == BULK_PULL_ACCOUNT_HASH_AMOUNT_AND_ADDRESS {
// 		return srv.HandleBulkPullAccountHashAmountAndAddress(peer, reader, frontier_hash, balance)
// 	}

// 	return errors.New("Invalid bulk_pull_account flag provided")
// }
