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

	HashPairsQueue      map[*networking.PeerNode]chan *packets.ConfirmAckByHashes
	HashPairsQueueMutex sync.Mutex

	BlockVotes map[types.Hash]map[types.Address]types.Amount // mapping(block_hash => mapping(voter => weight))
}

func (worker *ConfirmAckWorker) IsTrustedPR(address types.Address) bool {
	is_trusted, found := worker.P2PServer.Config.Consensus.TrustedPRs[address.ToHexString()]
	return found && is_trusted
}

func (worker *ConfirmAckWorker) CementBlock(hash types.Hash) {
	// log.Println("Cementing block", hash.ToHexString())
}

func (worker *ConfirmAckWorker) HandleConfirmAck(peer *networking.PeerNode, vote *packets.ConfirmAckByHashes) {
	// Validate signature
	vote_hash, err := calculateVoteHash(*vote.TimestampAndVoteDuration, vote.Hashes)
	if err != nil {
		log.Println("Error calculating vote hash:", err)

		return
	}

	is_vote_signature_valid := ed25519.Verify(vote.Account.ToPublicKey(), vote_hash, vote.Signature[:])
	if !is_vote_signature_valid {
		log.Printf("Received invalid confirm_ack signature from node %s voted by address %s for %d blocks, blocks: %v", peer.NodeID.ToNodeAddress(), vote.Account.ToNanoAddress(), len(*vote.Hashes), *vote.Hashes)

		return
	}

	// Current codebase does not care about voting nodes yet, so we will only process final votes. Maybe we should rebroadcast these non-final votes tho
	if !vote.TimestampAndVoteDuration.IsFinalVote() {
		return
	}

	for _, hash := range *vote.Hashes {
		// Instantly cement block if it was a final vote from a trusted PR
		if worker.IsTrustedPR(*vote.Account) {
			worker.CementBlock(*hash)
			continue
		}

		// Add vote to list of votes that we already received for this block
		_, found := worker.BlockVotes[*hash]
		if !found {
			// First vote on a block hash
			worker.BlockVotes[*hash] = make(map[types.Address]types.Amount)
		}

		voting_weight := worker.P2PServer.Database.Backend.GetVotingWeight(vote.Account)
		if voting_weight.IsZero() {
			continue
		}

		_, found = worker.BlockVotes[*hash][*vote.Account]
		if found {
			continue
		}

		// Add this vote's weight
		worker.BlockVotes[*hash][*vote.Account] = voting_weight

		// Can be cached instead of re-calculated everytime
		total_voting_weight := types.Amount{}
		for _, i := range worker.BlockVotes[*hash] {
			total_voting_weight = i.Add(total_voting_weight)
		}

		minimum_weight_to_cement, _ := types.AmountFromString("42000000000000000000000000000000000000") // 42,000,000 XNO
		if total_voting_weight.Cmp(minimum_weight_to_cement) == 1 {
			worker.CementBlock(*hash)
		}
	}
}

func (worker *ConfirmAckWorker) StartQueueProcessor() {
	for {
		// If a key is in HashPairsQueue then it should be in BlocksQueue as well
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

func (worker *ConfirmAckWorker) Start() {
	go worker.StartQueueProcessor()
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

		HashPairsQueue: make(map[*networking.PeerNode]chan *packets.ConfirmAckByHashes, 1024),
		BlockVotes:     make(map[types.Hash]map[types.Address]types.Amount),
	}
}
