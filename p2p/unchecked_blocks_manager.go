package p2p

import (
	"sync"

	"github.com/Shryder/gnano/types"
)

type UncheckedBlocksManager struct {
	Table      map[types.Hash]*types.Block
	TableMutex sync.RWMutex
}

func NewUncheckedBlocksManager() UncheckedBlocksManager {
	return UncheckedBlocksManager{
		Table:      make(map[types.Hash]*types.Block),
		TableMutex: sync.RWMutex{},
	}
}
