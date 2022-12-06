package database

import (
	"fmt"
	"math/big"
	"time"

	"github.com/Shryder/gnano/types"
)

func (backend *JSONBackend) GetBlockCount() uint64 {
	backend.DataMutex.RLock()
	defer backend.DataMutex.RUnlock()

	return uint64(len(backend.Data.Blocks))
}

func (backend *JSONBackend) GetBlock(hash *types.Hash) *types.Block {
	backend.DataMutex.RLock()
	defer backend.DataMutex.RUnlock()

	block, found := backend.Data.Blocks[hash.ToHexString()]
	if !found {
		return nil
	}

	return &block
}

// Update local account ledger entry. Contains height, timestamp and frontier
func (backend *JSONBackend) UpdateLocalAccount(block *types.Block) error {
	blockAccountHex := block.Account.ToHexString()

	account, ok := backend.Data.Accounts[blockAccountHex]
	if !ok {
		if !block.IsOpenBlock() {
			return fmt.Errorf("error inserting block %s into the ledger because an OPEN block was expected", block.Hash.ToHexString())
		}

		// Initialize account in ledger
		backend.Data.Accounts[blockAccountHex] = DBAccount{
			Frontier: block.Hash,
			Sideband: &types.Sideband{
				Height:    big.NewInt(1),
				Timestamp: uint(time.Now().Unix()),
			},
		}
	} else {
		if block.Previous.Cmp(account.Frontier) != 0 {
			return fmt.Errorf("error inserting block %s into the ledger because current frontier block is %s but this block's previous is %s", block.Hash.ToHexString(), account.Frontier.ToHexString(), block.Previous.ToHexString())
		}

		backend.Data.Accounts[blockAccountHex] = DBAccount{
			Frontier: block.Hash,
			Sideband: &types.Sideband{
				Height:    account.Sideband.Height.Add(account.Sideband.Height, big.NewInt(1)), // Increase height by 1
				Timestamp: account.Sideband.Timestamp,
			},
		}
	}

	return nil
}

func (backend *JSONBackend) PutBlock(block *types.Block) error {
	backend.DataMutex.Lock()
	defer backend.DataMutex.Unlock()

	err := backend.UpdateLocalAccount(block)
	if err != nil {
		return err
	}

	backend.Data.Blocks[block.Hash.ToHexString()] = *block

	return nil
}
