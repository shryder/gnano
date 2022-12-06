package p2p

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Shryder/gnano/p2p/networking"
	"github.com/Shryder/gnano/p2p/packets"
	"github.com/Shryder/gnano/types"
	"github.com/shryder/ed25519-blake2b"
)

type PeerConfirmAcks struct {
	HashPairRequests [][]*packets.HashPair
	Mutex            sync.RWMutex
}

type ConfirmAckWorker struct {
	P2PServer *P2P

	CementQueue chan *types.Hash // Queue of block hashes to cement

	ConfirmedButWaitingForBlockBody      map[types.Hash]bool // We tried to cement but we did not have block body, continuously check if we received the body yet
	ConfirmedButWaitingForBlockBodyMutex sync.RWMutex

	ConfirmAckQueue      map[*networking.PeerNode]chan *packets.ConfirmAckByHashes
	ConfirmAckQueueMutex sync.RWMutex

	BlockVotes map[types.Hash]map[types.Address]types.Amount // mapping(block_hash => mapping(voter => weight))
}

func (worker *ConfirmAckWorker) IsTrustedPR(address types.Address) bool {
	is_trusted, found := worker.P2PServer.Config.Consensus.TrustedPRs[address.ToHexString()]

	return found && is_trusted
}

// Traverses the account's blockchain until it reaches the cemented block.
func (worker *ConfirmAckWorker) GetChainUntilCementedFrontier(hash types.Hash) ([]*types.Hash, *types.Hash) {
	block := worker.P2PServer.UncheckedBlocksManager.Get(&hash)

	// If it's an open block then simply return this block. Also this account's chain should not exist yet.
	if block.IsOpenBlock() {
		// Make sure the account chain does not exist, means there is a bug somewhere
		account := worker.P2PServer.Database.Backend.GetAccount(block.Account)
		if account != nil {
			panic(fmt.Sprintf("Account chain should not exist yet %s %s", hash.ToHexString(), block.Account.ToNanoAddress()))
		}

		return []*types.Hash{&hash}, nil
	}

	chain := []*types.Hash{}
	cursor := block.Previous
	for {
		ledgerBlock := worker.P2PServer.Database.Backend.GetBlock(cursor)
		if ledgerBlock != nil {
			// Because this block was found in the ledger, that means we reached the frontier cemented block in this account's chain
			chain = append([]*types.Hash{cursor}, chain...)

			return chain, nil
		}

		uncheckedBlock := worker.P2PServer.UncheckedBlocksManager.Get(cursor)
		if uncheckedBlock == nil {
			// There is a gap
			return chain, cursor
		}

		chain = append([]*types.Hash{cursor}, chain...)

		if uncheckedBlock.IsOpenBlock() {
			// Reached open block, we have the entire account's chain, return what we found
			return chain, nil
		}

		cursor = uncheckedBlock.Previous
	}
}

// Cement block if we have contents, return false if we don't have content
func (worker *ConfirmAckWorker) TryCementBlock(hashToCement types.Hash) {
	worker.CementQueue <- &hashToCement
}

func (worker *ConfirmAckWorker) HandleConfirmAck(peer *networking.PeerNode, vote *packets.ConfirmAckByHashes) {
	// Validate signature
	voteHash, err := calculateVoteHashPtrs(*vote.TimestampAndVoteDuration, vote.Hashes)
	if err != nil {
		log.Println("Error calculating vote hash:", err)

		return
	}

	isVoteSignatureValid := ed25519.Verify(vote.Account.ToPublicKey(), voteHash, vote.Signature[:])
	if !isVoteSignatureValid {
		log.Printf("Received invalid confirm_ack signature from node %s voted by address %s for %d blocks, blocks: %v", peer.NodeID.ToNodeAddress(), vote.Account.ToNanoAddress(), len(*vote.Hashes), *vote.Hashes)

		return
	}

	if !worker.P2PServer.VotingEnabled && !vote.TimestampAndVoteDuration.IsFinalVote() {
		log.Println("Received confirm_ack that we are ignoring because it's not a final vote:", vote.TimestampAndVoteDuration.IsFinalVote())
		return
	}

	for _, hash := range *vote.Hashes {
		log.Println("Received confirm_ack votes from", peer.NodeID.ToNodeAddress(), "on", hash.ToHexString(), "using account", vote.Account.ToNanoAddress())

		ledgerBlock := worker.P2PServer.Database.Backend.GetBlock(hash)
		if ledgerBlock != nil {
			log.Println("Block already cemented:", ledgerBlock.Hash.ToHexString())
			continue
		}

		// Instantly cement block if it was a final vote from a trusted PR
		if worker.IsTrustedPR(*vote.Account) {
			worker.TryCementBlock(*hash)
			continue
		}

		// Add vote to list of votes that we already received for this block
		// _, found := worker.BlockVotes[*hash]
		// if !found {
		// 	// First vote on a block hash
		// 	worker.BlockVotes[*hash] = make(map[types.Address]types.Amount)
		// }

		// voting_weight := worker.P2PServer.Database.Backend.GetVotingWeight(vote.Account)
		// if voting_weight.IsZero() {
		// 	continue
		// }

		// _, found = worker.BlockVotes[*hash][*vote.Account]
		// if found {
		// 	continue
		// }

		// // Add this vote's weight
		// worker.BlockVotes[*hash][*vote.Account] = voting_weight

		// // Can be cached instead of re-calculated everytime
		// total_voting_weight := types.Amount{}
		// for _, i := range worker.BlockVotes[*hash] {
		// 	total_voting_weight = i.Add(total_voting_weight)
		// }

		// minimum_weight_to_cement, _ := types.AmountFromString("42000000000000000000000000000000000000") // 42,000,000 XNO
		// if total_voting_weight.Cmp(minimum_weight_to_cement) == 1 {
		// 	worker.TryCementBlock(*hash)
		// }
	}
}

func (worker *ConfirmAckWorker) StartCementProcessor() {
	for {
		hashToCement := <-worker.CementQueue

		unchecked_block := worker.P2PServer.UncheckedBlocksManager.Get(hashToCement)
		if unchecked_block == nil {
			log.Println("Couldn't cement block", hashToCement.ToHexString(), "because we don't have its body")

			worker.ConfirmedButWaitingForBlockBodyMutex.Lock()
			worker.ConfirmedButWaitingForBlockBody[*hashToCement] = true
			worker.ConfirmedButWaitingForBlockBodyMutex.Unlock()

			// Notify bootstrapper to request the missing block from peers
			worker.P2PServer.BootstrapDataManager.AddUnknownBlockHash(hashToCement)
			continue
		}

		// Get all parent blocks all the way until frontier
		chain, gap := worker.GetChainUntilCementedFrontier(*hashToCement)

		// There is a gap, no cementing until we have the entire chain
		if gap != nil {
			log.Println("Found gap in chain, unable to cement for now. Chain length:", len(chain), "account:", unchecked_block.Account.ToNanoAddress(), "gap at:", gap.ToHexString())

			// Notify bootstrapper to request the missing gap block from peers
			worker.P2PServer.BootstrapDataManager.AddUnknownBlockHash(gap)

			continue
		}

		// Loop the chain in reverse to start cementing from lowest height
		for _, hash := range chain {
			log.Println("Cementing block", hash.ToHexString())

			// Block exists otherwise GetChainUntilCementedFrontier wouldn't return it
			block := worker.P2PServer.UncheckedBlocksManager.Get(hash)

			// Save to ledger
			err := worker.P2PServer.Database.Backend.PutBlock(block)
			if err != nil {
				chain_jsonified, _ := json.Marshal(chain)
				log.Println("Error saving block to ledger:", err, "chain:", string(chain_jsonified), "account cemented chain:", worker.P2PServer.Database.Backend.GetAccountChain(block.Account))

				panic("Error saving block to ledger")
			}

			// Don't request votes on this block anymore
			worker.P2PServer.Workers.ConfirmReq.MarkBlockAsConfirmed(types.HashPair{Root: *block.Previous, Hash: *block.Hash})

			// Remove from unchecked table
			worker.P2PServer.UncheckedBlocksManager.Remove(hash)

			// Don't request this block's body anymore
			worker.ConfirmedButWaitingForBlockBodyMutex.Lock()
			delete(worker.ConfirmedButWaitingForBlockBody, *hash)
			worker.ConfirmedButWaitingForBlockBodyMutex.Unlock()
		}
	}
}

func (worker *ConfirmAckWorker) StartQueueProcessor() {
	for {
		worker.ConfirmAckQueueMutex.RLock()

		for peer := range worker.ConfirmAckQueue {
			select {
			case ack, ok := <-worker.ConfirmAckQueue[peer]:
				if ok {
					worker.HandleConfirmAck(peer, ack)
				}
			default:
				continue
			}
		}

		worker.ConfirmAckQueueMutex.RUnlock()

		time.Sleep(time.Millisecond * 50)
	}
}

// Continuously retry cementing the blocks that we couldn't cement before because we didn't have block body
func (worker *ConfirmAckWorker) WaitForBlockBodyBeforeCementing() {
	for {
		worker.ConfirmedButWaitingForBlockBodyMutex.RLock()
		for hash := range worker.ConfirmedButWaitingForBlockBody {
			worker.TryCementBlock(hash)
		}
		worker.ConfirmedButWaitingForBlockBodyMutex.RUnlock()

		time.Sleep(time.Millisecond * 1000)
	}
}

func (worker *ConfirmAckWorker) Start() {
	for i := 0; i < 16; i++ {
		go worker.StartQueueProcessor()
	}

	go worker.WaitForBlockBodyBeforeCementing()
	go worker.StartCementProcessor()
}

func (worker *ConfirmAckWorker) AddConfirmAckToQueue(peer *networking.PeerNode, ack *packets.ConfirmAckByHashes) {
	worker.ConfirmAckQueue[peer] <- ack
}

func (worker *ConfirmAckWorker) RegisterNewPeer(peer *networking.PeerNode) {
	worker.ConfirmAckQueueMutex.Lock()
	worker.ConfirmAckQueue[peer] = make(chan *packets.ConfirmAckByHashes)
	worker.ConfirmAckQueueMutex.Unlock()
}

func (worker *ConfirmAckWorker) UnregisterNewPeer(peer *networking.PeerNode) {
	worker.ConfirmAckQueueMutex.Lock()
	delete(worker.ConfirmAckQueue, peer)
	worker.ConfirmAckQueueMutex.Unlock()
}

func NewConfirmAckWorker(srv *P2P) *ConfirmAckWorker {
	return &ConfirmAckWorker{
		P2PServer: srv,

		CementQueue:                          make(chan *types.Hash, 512_000),
		ConfirmedButWaitingForBlockBody:      make(map[types.Hash]bool),
		ConfirmedButWaitingForBlockBodyMutex: sync.RWMutex{},

		ConfirmAckQueue:      make(map[*networking.PeerNode]chan *packets.ConfirmAckByHashes),
		ConfirmAckQueueMutex: sync.RWMutex{},

		BlockVotes: make(map[types.Hash]map[types.Address]types.Amount),
	}
}
