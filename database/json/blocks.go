package database

import (
	"github.com/Shryder/gnano/types"
)

func (backend *JSONBackend) GetBlock(hash *types.Hash) *types.Block {
	backend.DataMutex.RLock()
	defer backend.DataMutex.RUnlock()

	block, found := backend.Data.Blocks[hash.ToHexString()]
	if !found {
		return nil
	}

	return &block
}
