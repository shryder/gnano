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
		Sideband: *account_json.Sideband,
	}
}

func (backend *JSONBackend) GetAccountCount() uint64 {
	backend.DataMutex.RLock()
	defer backend.DataMutex.RUnlock()

	return uint64(len(backend.Data.Accounts))
}

func (backend *JSONBackend) GetAccountChain(address *types.Address) []string {
	backend.DataMutex.RLock()
	defer backend.DataMutex.RUnlock()

	chain := make([]string, 0)
	account := backend.GetAccount(address)
	if account == nil {
		return chain
	}

	cursor := account.Frontier.Hash

	for {
		block := backend.GetBlock(cursor)
		chain = append(chain, block.Hash.ToHexString())

		if block.IsOpenBlock() {
			return chain
		}
	}
}

func (backend *JSONBackend) GetRandomAccountAddress() *types.Address {
	backend.DataMutex.RLock()
	defer backend.DataMutex.RUnlock()

	for address := range backend.Data.Accounts {
		addy, err := types.StringPublicKeyToAddress(address)
		if err != nil {
			continue
		}

		return addy
	}

	return nil
}

func (backend *JSONBackend) StoreAccount(account *types.Account) error {
	backend.DataMutex.Lock()
	defer backend.DataMutex.Unlock()

	backend.Data.Accounts[account.Frontier.Account.ToHexString()] = DBAccount{
		Frontier: account.Frontier.Hash,
		Sideband: &account.Sideband,
	}

	return nil
}
