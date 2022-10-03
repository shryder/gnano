package database

import (
	"log"

	"github.com/Shryder/gnano/types"
)

func (backend *JSONBackend) GetAccount(public_address *types.Address) *types.Account {
	backend.DataMutex.RLock()
	defer backend.DataMutex.RUnlock()

	account_json, found := backend.Data.Accounts[public_address.ToHexString()]
	if !found {
		return nil
	}

	frontier_block_json, found := backend.Data.Blocks[account_json.Frontier.ToHexString()]
	if !found {
		log.Println("Account's frontier was not found.")
		return nil
	}

	return &types.Account{
		Frontier: frontier_block_json,
		Sideband: account_json.Sideband,
	}
}

func (backend *JSONBackend) StoreAccount(account *types.Account) error {
	backend.DataMutex.Lock()
	defer backend.DataMutex.Unlock()

	backend.Data.Accounts[account.Frontier.Account.ToHexString()] = DBAccount{
		Frontier: *account.Frontier.Hash(),
		Sideband: account.Sideband,
	}

	return nil
}
