package p2p

import (
	"encoding/hex"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"github.com/Shryder/gnano/p2p/networking"
	"github.com/Shryder/gnano/p2p/packets"
	"github.com/Shryder/gnano/types"
)

type VoteKey struct {
	Root types.Hash
	Hash types.Hash
}

type PeerConfirmRequests struct {
	HashPairRequests [][]*packets.HashPair
	BlockRequests    []*types.Block
	Mutex            sync.RWMutex
}

type ConfirmReqWorker struct {
	Logger    *log.Logger
	P2PServer *P2P

	RequestForConfirmations map[*networking.PeerNode]chan *[][]byte // array of hashpairs

	ReceivedHashPairsQueue map[*networking.PeerNode]chan []*packets.HashPair
	ReceivedBlocksQueue    map[*networking.PeerNode]chan *types.Block
	Mutex                  sync.Mutex
}

func HashPairToString(hash_pairs [][]byte) []string {
	arr := make([]string, len(hash_pairs))
	for i, pair := range hash_pairs {
		arr[i] = "[" + hex.EncodeToString(pair[0:32]) + `, ` + hex.EncodeToString(pair[32:64]) + "]"
	}

	return arr
}

func (worker *ConfirmReqWorker) RequestVotesOnTheseUnknownBlocks(hash_pairs [][]byte, block_origin *networking.PeerNode) {
	subset_amount := worker.P2PServer.PeersManager.GetSubsetOfLivePeers()

	if block_origin != nil {
		worker.Logger.Println("Peer", block_origin.Alias, "notified us on", len(hash_pairs), "pairs that we are not aware of. Requesting votes from", subset_amount, "other peers.", HashPairToString(hash_pairs))
	} else {
		worker.Logger.Println("Requesting votes on", len(hash_pairs), "pairs from", subset_amount, "peers", HashPairToString(hash_pairs))
	}

	// Ask other peers about these unknown blocks
	worker.P2PServer.PeersManager.PeersMutex.RLock()
	for _, peer := range worker.P2PServer.PeersManager.LivePeers {
		if peer == block_origin {
			continue
		}

		if subset_amount == 0 {
			break
		}

		worker.RequestForConfirmations[peer] <- &hash_pairs

		subset_amount--
	}
	worker.P2PServer.PeersManager.PeersMutex.RUnlock()
}

func (worker *ConfirmReqWorker) StartRequestingConfirmations() {
	for {
		time.Sleep(time.Millisecond * 250)

		for peer, channel := range worker.RequestForConfirmations {
			select {
			case pair := <-channel:
				err := worker.P2PServer.SendConfirmReq(peer, *pair)
				if err != nil {
					worker.Logger.Println("Error sending confirm_req to peer", peer.Alias, err)
				}
			default:
				continue
			}
		}
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
		unknown_hash_pairs = append(unknown_hash_pairs, hashPair.ToBytes())
	}

	// Ask other nodes to confirm_req these new unknown blocks
	// TODO: add the unkown block to a queue to prevent DDoS & duplicate requests
	if len(unknown_hash_pairs) > 0 {
		worker.P2PServer.BootstrapDataManager.FoundBlockWithoutBody(unknown_hash_pairs)
		worker.RequestVotesOnTheseUnknownBlocks(unknown_hash_pairs, peer) // maybe request votes after we receive the block
	}

	worker.Logger.Println("Finished processing", len(hashPairs), "hashpair requests from", peer.NodeID.ToNodeAddress())
}

func (worker *ConfirmReqWorker) HandleBlockRequest(node *networking.PeerNode, block *types.Block) {
	worker.Logger.Println("Received confirm_req with block", block.Account.ToNanoAddress(), "from", node.NodeID.ToNodeAddress())
}

func (worker *ConfirmReqWorker) StartQueueProcessor() {
	for {
		worker.Mutex.Lock()
		// If a key is in ReceivedHashPairsQueue then it should be in ReceivedBlocksQueue as well
		for peer := range worker.ReceivedHashPairsQueue {
			select {
			case pairs := <-worker.ReceivedHashPairsQueue[peer]:
				worker.HandleHashPairRequest(peer, pairs)

			case block := <-worker.ReceivedBlocksQueue[peer]:
				worker.HandleBlockRequest(peer, block)

			default:
				continue
			}
		}
		worker.Mutex.Unlock()

		time.Sleep(time.Millisecond * 250)
	}
}

func (worker *ConfirmReqWorker) Start() {
	go worker.StartQueueProcessor()
	go worker.StartRequestingConfirmations()
}

func (worker *ConfirmReqWorker) AddConfirmReqHashPairsToQueue(peer *networking.PeerNode, pairs []*packets.HashPair) {
	worker.ReceivedHashPairsQueue[peer] <- pairs
}

func (worker *ConfirmReqWorker) ProcessBlock(peer *networking.PeerNode, block *types.Block) {
	worker.ReceivedBlocksQueue[peer] <- block
}

func (worker *ConfirmReqWorker) RegisterNewPeer(peer *networking.PeerNode) {
	worker.Mutex.Lock()
	worker.ReceivedHashPairsQueue[peer] = make(chan []*packets.HashPair)
	worker.ReceivedBlocksQueue[peer] = make(chan *types.Block)
	worker.RequestForConfirmations[peer] = make(chan *[][]byte)
	worker.Mutex.Unlock()
}

func (worker *ConfirmReqWorker) UnregisterNewPeer(peer *networking.PeerNode) {
	worker.Mutex.Lock()
	delete(worker.ReceivedHashPairsQueue, peer)
	delete(worker.ReceivedBlocksQueue, peer)
	delete(worker.RequestForConfirmations, peer)
	worker.Mutex.Unlock()
}

func NewConfirmReqWorker(srv *P2P) *ConfirmReqWorker {
	logger := log.New(os.Stdout, "[ConfirmReq] ", log.Ltime)
	if !srv.Config.P2P.Logs.ConfirmReqWorker {
		logger.SetOutput(io.Discard)
	}

	return &ConfirmReqWorker{
		Logger:    logger,
		P2PServer: srv,

		RequestForConfirmations: make(map[*networking.PeerNode]chan *[][]byte),
		ReceivedHashPairsQueue:  make(map[*networking.PeerNode]chan []*packets.HashPair, 1024),
		ReceivedBlocksQueue:     make(map[*networking.PeerNode]chan *types.Block, 1024),
	}
}
