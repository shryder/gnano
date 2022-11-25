package p2p

import (
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

	ConfirmedButWaitingForBlockBody      map[types.Hash]bool // We tried to cement but we did not have block body, contiously check if we received the body yet
	ConfirmedButWaitingForBlockBodyMutex sync.Mutex

	HashPairsQueue      map[*networking.PeerNode]chan *packets.ConfirmAckByHashes
	HashPairsQueueMutex sync.Mutex

	BlockVotes map[types.Hash]map[types.Address]types.Amount // mapping(block_hash => mapping(voter => weight))
}

func (worker *ConfirmAckWorker) IsTrustedPR(address types.Address) bool {
	is_trusted, found := worker.P2PServer.Config.Consensus.TrustedPRs[address.ToHexString()]
	return found && is_trusted
}

// Cement block if we have contents
func (worker *ConfirmAckWorker) TryCementBlock(hash types.Hash) bool {
	unchecked_block := worker.P2PServer.UncheckedBlocksManager.Get(&hash)
	if unchecked_block == nil {
		log.Println("Couldn't cement block", hash.ToHexString(), "because we don't have its body")

		worker.ConfirmedButWaitingForBlockBodyMutex.Lock()
		worker.ConfirmedButWaitingForBlockBody[hash] = true
		worker.ConfirmedButWaitingForBlockBodyMutex.Unlock()

		return false
	}

	log.Println("Cementing block", hash.ToHexString())

	worker.P2PServer.UncheckedBlocksManager.Remove(unchecked_block.Hash)
	worker.P2PServer.Database.Backend.PutBlock(unchecked_block)
	return true
}

func (worker *ConfirmAckWorker) HandleConfirmAck(peer *networking.PeerNode, vote *packets.ConfirmAckByHashes) {
	// Validate signature
	vote_hash, err := calculateVoteHashPtrs(*vote.TimestampAndVoteDuration, vote.Hashes)
	if err != nil {
		log.Println("Error calculating vote hash:", err)

		return
	}

	is_vote_signature_valid := ed25519.Verify(vote.Account.ToPublicKey(), vote_hash, vote.Signature[:])
	if !is_vote_signature_valid {
		log.Printf("Received invalid confirm_ack signature from node %s voted by address %s for %d blocks, blocks: %v", peer.NodeID.ToNodeAddress(), vote.Account.ToNanoAddress(), len(*vote.Hashes), *vote.Hashes)

		return
	}

	if !worker.P2PServer.VotingEnabled && !vote.TimestampAndVoteDuration.IsFinalVote() {
		log.Println("Received confirm_ack that we are ignoring:", vote.TimestampAndVoteDuration.IsFinalVote())
		return
	}

	for _, hash := range *vote.Hashes {
		log.Println("Received confirm_ack votes from", peer.NodeID.ToNodeAddress(), "on", hash.ToHexString())

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

func (worker *ConfirmAckWorker) StartQueueProcessor() {
	for {
		for peer := range worker.HashPairsQueue {
			select {
			case ack := <-worker.HashPairsQueue[peer]:
				worker.HandleConfirmAck(peer, ack)

			default:
				continue
			}
		}

		time.Sleep(time.Millisecond * 250)
	}
}

func (worker *ConfirmAckWorker) WaitForBlockBodyBeforeCementing() {
	for {
		worker.ConfirmedButWaitingForBlockBodyMutex.Lock()
		for hash := range worker.ConfirmedButWaitingForBlockBody {
			cemented := worker.TryCementBlock(hash)
			if cemented {
				delete(worker.ConfirmedButWaitingForBlockBody, hash)
			}
		}
		worker.ConfirmedButWaitingForBlockBodyMutex.Unlock()

		time.Sleep(time.Millisecond * 250)
	}
}

func (worker *ConfirmAckWorker) Start() {
	go worker.StartQueueProcessor()
	go worker.WaitForBlockBodyBeforeCementing()
}

func (worker *ConfirmAckWorker) AddConfirmAckToQueue(peer *networking.PeerNode, ack *packets.ConfirmAckByHashes) {
	worker.HashPairsQueue[peer] <- ack
}

func (worker *ConfirmAckWorker) RegisterNewPeer(peer *networking.PeerNode) {
	worker.HashPairsQueueMutex.Lock()
	worker.HashPairsQueue[peer] = make(chan *packets.ConfirmAckByHashes)
	worker.HashPairsQueueMutex.Unlock()
}

func (worker *ConfirmAckWorker) UnregisterNewPeer(peer *networking.PeerNode) {
	worker.HashPairsQueueMutex.Lock()
	delete(worker.HashPairsQueue, peer)
	worker.HashPairsQueueMutex.Unlock()
}

func NewConfirmAckWorker(srv *P2P) *ConfirmAckWorker {
	return &ConfirmAckWorker{
		P2PServer: srv,

		ConfirmedButWaitingForBlockBody:      make(map[types.Hash]bool),
		ConfirmedButWaitingForBlockBodyMutex: sync.Mutex{},

		HashPairsQueue:      make(map[*networking.PeerNode]chan *packets.ConfirmAckByHashes, 1024),
		HashPairsQueueMutex: sync.Mutex{},

		BlockVotes: make(map[types.Hash]map[types.Address]types.Amount),
	}
}
