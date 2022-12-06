package p2p

import (
	"log"
	"sync"
	"time"

	"github.com/Shryder/gnano/types"
	"github.com/shryder/ed25519-blake2b"
)

type UncheckedBlocksManager struct {
	P2PServer *P2P

	// Blocks that we are looking for votes for
	BatchVoteRequests      map[types.Hash]*types.Hash // mapping(hash => root)
	BatchVoteRequestsMutex sync.RWMutex

	Queue                chan *types.Block
	UncheckedBlocks      map[types.Hash]*types.Block
	UncheckedBlocksMutex sync.RWMutex
}

func NewUncheckedBlocksManager(srv *P2P) UncheckedBlocksManager {
	return UncheckedBlocksManager{
		P2PServer:         srv,
		Queue:             make(chan *types.Block, 256_000),
		UncheckedBlocks:   make(map[types.Hash]*types.Block, 256_000),
		BatchVoteRequests: make(map[types.Hash]*types.Hash),

		BatchVoteRequestsMutex: sync.RWMutex{},
		UncheckedBlocksMutex:   sync.RWMutex{},
	}
}

// Add block to list of unconfirmed blocks to ask peers for confirmations
func (manager *UncheckedBlocksManager) RequestVotesOnBlock(block *types.Block) {
	manager.BatchVoteRequestsMutex.Lock()
	defer manager.BatchVoteRequestsMutex.Unlock()

	manager.BatchVoteRequests[*block.Hash] = block.Root()
}

func (manager *UncheckedBlocksManager) InsertToUncheckedTable(block *types.Block) {
	manager.UncheckedBlocksMutex.Lock()
	defer manager.UncheckedBlocksMutex.Unlock()

	manager.UncheckedBlocks[*block.Hash] = block
}

func (manager *UncheckedBlocksManager) ValidateSignature(block *types.Block) bool {
	epochV2 := "epoch v2 block"
	if string(block.Link[0:len(epochV2)]) == epochV2 {
		return true
	}

	epochV1 := "epoch v1 block"
	if string(block.Link[0:len(epochV1)]) == epochV1 {
		return true
	}

	return ed25519.Verify(ed25519.PublicKey(block.Account[:]), block.Hash[:], block.Signature[:])
}

// Processes new incoming blocks (from bulk_pull_response/publish) and adds them to unchecked table
func (manager *UncheckedBlocksManager) ProcessNewBlocks() {
	for {
		block := <-manager.Queue

		valid_signature := manager.ValidateSignature(block)
		if !valid_signature {
			log.Println("Encountered block with invalid signature:", block.Hash.ToHexString(), string(block.Link[:]), *block)
			continue
		}

		manager.InsertToUncheckedTable(block)
		// manager.RequestVotesOnBlock(block) // TODO: Maybe wait a little before requesting other nodes for votes, depending on how we received the block
	}
}

// Returns a list of up to 64 hash pairs that we are looking for confirmations for
func (manager *UncheckedBlocksManager) UnconfirmedHashPairs() [][]byte {
	hash_pairs := make([][]byte, 0)

	manager.BatchVoteRequestsMutex.Lock()
	defer manager.BatchVoteRequestsMutex.Unlock()

	batchVoteRequestsCount := uint(len(manager.BatchVoteRequests))
	max := uint(1)
	if batchVoteRequestsCount < max {
		max = batchVoteRequestsCount
	}

	for hash, root := range manager.BatchVoteRequests {
		if max <= 0 {
			break
		}

		hash_pairs = append(hash_pairs, append(hash[:], root[:]...))
		delete(manager.BatchVoteRequests, hash)
		max--
	}

	return hash_pairs
}

// Process unchecked blocks that we don't have enough (or no) confirmations on
func (manager *UncheckedBlocksManager) BatchRequestVotesOnUncheckedBlocks() {
	for {
		time.Sleep(time.Second * 3)

		hash_pairs := manager.UnconfirmedHashPairs()
		if len(hash_pairs) == 0 {
			continue
		}

		manager.P2PServer.Workers.ConfirmReq.RequestVotesOnTheseBlocks(hash_pairs, nil)
	}
}

func (manager *UncheckedBlocksManager) UncheckedBlocksCount() uint {
	manager.UncheckedBlocksMutex.RLock()
	defer manager.UncheckedBlocksMutex.RUnlock()

	return uint(len(manager.UncheckedBlocks))
}

func (manager *UncheckedBlocksManager) GetRandomBlock() *types.Hash {
	manager.UncheckedBlocksMutex.RLock()
	defer manager.UncheckedBlocksMutex.RUnlock()

	for hash := range manager.UncheckedBlocks {
		return &hash
	}

	return nil
}

func (manager *UncheckedBlocksManager) GetUncheckedBlocks() map[types.Hash]*types.Block {
	manager.UncheckedBlocksMutex.RLock()
	defer manager.UncheckedBlocksMutex.RUnlock()

	return manager.UncheckedBlocks
}

func (manager *UncheckedBlocksManager) Start() {
	go manager.ProcessNewBlocks()
	go manager.BatchRequestVotesOnUncheckedBlocks()
}

func (manager *UncheckedBlocksManager) Get(hash *types.Hash) *types.Block {
	manager.UncheckedBlocksMutex.RLock()
	defer manager.UncheckedBlocksMutex.RUnlock()

	return manager.UncheckedBlocks[*hash]
}

func (manager *UncheckedBlocksManager) Remove(hash *types.Hash) {
	manager.UncheckedBlocksMutex.Lock()
	delete(manager.UncheckedBlocks, *hash)
	manager.UncheckedBlocksMutex.Unlock()
}

func (manager *UncheckedBlocksManager) Add(block *types.Block) {
	unchecked_block := manager.Get(block.Hash)
	if unchecked_block != nil {
		// Block is already in the unchecked table
		return
	}

	ledger_block := manager.P2PServer.Database.Backend.GetBlock(block.Hash)
	if ledger_block != nil {
		// Block is already cemented
		return
	}

	manager.Queue <- block
}
