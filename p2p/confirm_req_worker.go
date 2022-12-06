package p2p

import (
	"io"
	"log"
	"os"
	"sync"
	"time"

	"github.com/Shryder/gnano/p2p/networking"
	"github.com/Shryder/gnano/p2p/packets"
	"github.com/Shryder/gnano/types"
	"github.com/Shryder/gnano/utils"
)

type ConfirmReqWorker struct {
	Logger    *log.Logger
	P2PServer *P2P

	RequestForConfirmations      map[types.HashPair]bool // Hashpairs that we are looking for votes for, maybe store origin as well
	RequestForConfirmationsMutex sync.RWMutex

	IncomingConfirmReqQueue      map[*networking.PeerNode]chan []*packets.HashPair
	IncomingConfirmReqQueueMutex sync.RWMutex
}

func (worker *ConfirmReqWorker) RequestVotesOnTheseBlocks(hash_pairs [][]byte, block_origin *networking.PeerNode) {
	if block_origin != nil {
		worker.Logger.Println("Peer", block_origin.Alias, "notified us on", len(hash_pairs), "pairs that we are not aware of. Requesting votes from other peers.", utils.HashPairToString(hash_pairs))
	} else {
		worker.Logger.Println("Requesting votes on", len(hash_pairs), "pairs from peers", utils.HashPairToString(hash_pairs))
	}

	// Queue to ask other peers about these unknown blocks
	worker.RequestForConfirmationsMutex.Lock()
	for _, pair := range hash_pairs {
		hashPair := new(types.HashPair)
		hashPair.FromSlice(pair)

		worker.RequestForConfirmations[*hashPair] = true
	}
	worker.RequestForConfirmationsMutex.Unlock()
}

// Removes the hashpairs from worker.RequestForConfirmations once the block has been cemented
func (worker *ConfirmReqWorker) MarkBlockAsConfirmed(pair types.HashPair) {
	worker.RequestForConfirmationsMutex.Lock()
	delete(worker.RequestForConfirmations, pair)
	worker.RequestForConfirmationsMutex.Unlock()
}

// Returns max of 12 hashpairs to request votes for, read from RequestForConfirmations
func (worker *ConfirmReqWorker) GetHashPairsToRequestVotesFor() []types.HashPair {
	pending := make([]types.HashPair, 0)

	worker.RequestForConfirmationsMutex.RLock()
	for pair := range worker.RequestForConfirmations {
		pending = append(pending, pair)

		// max 12
		if len(pending) == 12 {
			break
		}
	}
	worker.RequestForConfirmationsMutex.RUnlock()

	return pending
}

func (worker *ConfirmReqWorker) StartRequestingConfirmations() {
	for {
		pending := worker.GetHashPairsToRequestVotesFor()
		if len(pending) > 0 {
			err := worker.P2PServer.SendConfirmReqToPeers(pending)
			if err != nil {
				worker.Logger.Println("Error sending confirm_req to peer", err)
			}
		}

		time.Sleep(time.Millisecond * 50)
	}
}

func (worker *ConfirmReqWorker) HandleHashPairRequest(peer *networking.PeerNode, hashPairs []*packets.HashPair) {
	// var cached_votes []*packets.VoteByHashes
	// var initial_vote_required []*packets.HashPair
	// var final_vote_required []*packets.HashPair
	var unknown_hash_pairs [][]byte

	worker.Logger.Println("Processing", len(hashPairs), "hashpair requests from", peer.NodeID.ToNodeAddress())

	for _, hashPair := range hashPairs {
		block := worker.P2PServer.Database.Backend.GetBlock(hashPair.Hash)
		if block != nil {
			// Block is already cemented
			continue
		}

		block = worker.P2PServer.UncheckedBlocksManager.Get(hashPair.Hash)
		if block != nil {
			// Block is in the unchecked table
			continue
		}

		// TODO: check if root is known
		unknown_hash_pairs = append(unknown_hash_pairs, hashPair.ToSlice())
	}

	worker.Logger.Println("Finished processing", len(hashPairs), "hashpair requests from", peer.NodeID.ToNodeAddress(), "unknown pairs count:", len(unknown_hash_pairs))
}

func (worker *ConfirmReqWorker) HandleBlockRequest(node *networking.PeerNode, block *types.Block) {
	worker.Logger.Println("Received confirm_req with block", block.Account.ToNanoAddress(), "from", node.NodeID.ToNodeAddress())
}

func (worker *ConfirmReqWorker) StartQueueProcessor() {
	for {
		worker.IncomingConfirmReqQueueMutex.RLock()

		for peer := range worker.IncomingConfirmReqQueue {
			// Read from peer's confirm_req queue, if there is anything to read
			select {
			case pair, ok := <-worker.IncomingConfirmReqQueue[peer]:
				if ok {
					worker.HandleHashPairRequest(peer, pair)
				}
			default:
				continue
			}
		}

		worker.IncomingConfirmReqQueueMutex.RUnlock()

		time.Sleep(time.Millisecond * 50)
	}
}

func (worker *ConfirmReqWorker) Start() {
	for i := 0; i < 16; i++ {
		go worker.StartQueueProcessor()
	}

	go worker.StartRequestingConfirmations()
}

func (worker *ConfirmReqWorker) AddConfirmReqHashPairsToQueue(peer *networking.PeerNode, pairs []*packets.HashPair) {
	worker.IncomingConfirmReqQueue[peer] <- pairs
}

func (worker *ConfirmReqWorker) RegisterNewPeer(peer *networking.PeerNode) {
	worker.IncomingConfirmReqQueueMutex.Lock()
	worker.IncomingConfirmReqQueue[peer] = make(chan []*packets.HashPair, 65536)
	worker.IncomingConfirmReqQueueMutex.Unlock()
}

func (worker *ConfirmReqWorker) UnregisterNewPeer(peer *networking.PeerNode) {
	worker.IncomingConfirmReqQueueMutex.Lock()
	delete(worker.IncomingConfirmReqQueue, peer)
	worker.IncomingConfirmReqQueueMutex.Unlock()
}

func NewConfirmReqWorker(srv *P2P) *ConfirmReqWorker {
	logger := log.New(os.Stdout, "[ConfirmReq] ", log.Ltime)
	if !srv.Config.P2P.Logs.ConfirmReqWorker {
		logger.SetOutput(io.Discard)
	}

	return &ConfirmReqWorker{
		Logger:    logger,
		P2PServer: srv,

		RequestForConfirmations:      make(map[types.HashPair]bool),
		RequestForConfirmationsMutex: sync.RWMutex{},

		IncomingConfirmReqQueue: make(map[*networking.PeerNode]chan []*packets.HashPair, 1024),
	}
}
