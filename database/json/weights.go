package database

import (
	"github.com/Shryder/gnano/types"
)

func (backend *JSONBackend) GetVotingWeight(address *types.Address) types.Amount {
	backend.DataMutex.RLock()
	defer backend.DataMutex.RUnlock()

	weight, found := backend.Data.VotingWeight[address.ToHexString()]
	if !found {
		return types.Amount{}
	}

	return weight
}
