package p2p

import (
	"log"
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
	P2PServer *P2P

	HashPairsQueue map[*networking.PeerNode]chan []*packets.HashPair
	BlocksQueue    map[*networking.PeerNode]chan *types.Block
	Mutex          sync.Mutex

	LocalVotes map[VoteKey]*packets.VoteByHashes
}

func (worker *ConfirmReqWorker) HandleHashPairRequest(peer *networking.PeerNode, hashPairs []*packets.HashPair) {
	var cached_votes []*packets.VoteByHashes
	// var initial_vote_required []*packets.HashPair
	// var final_vote_required []*packets.HashPair
	var unknown_blocks [][]byte

	log.Println("Processing", len(hashPairs), "hashpair requests from", peer.NodeID.ToNodeAddress())

	for _, hashPair := range hashPairs {
		vote, found := worker.LocalVotes[VoteKey{Root: *hashPair.Root, Hash: *hashPair.Hash}]
		if found {
			cached_votes = append(cached_votes, vote)
			continue
		}

		pair := append((*hashPair.Hash)[:], (*hashPair.Root)[:]...)
		unknown_blocks = append(unknown_blocks, pair)
	}

	log.Println("Finished processing", len(hashPairs), "hashpair requests from", peer.NodeID.ToNodeAddress())

	// Ask other nodes to confirm_req these new unknown blocks
	if len(unknown_blocks) > 0 {
		subset_amount := worker.P2PServer.GetSubsetOfPeers()
		log.Println("Peer", peer.NodeID.ToNodeAddress(), "requested votes on", len(unknown_blocks), "blocks that we are not aware of. Requesting vote from other peers.")

		// TODO: add the unkown block to a queue to prevent DDoS & duplicate requests
		for _, other_peer := range worker.P2PServer.Peers[:subset_amount] {
			if other_peer == peer {
				continue
			}

			err := worker.P2PServer.SendConfirmReq(other_peer, unknown_blocks)
			if err != nil {
				log.Println("Error sending confirm_req to node", err)
			}
		}
	}
}

func (worker *ConfirmReqWorker) HandleBlockRequest(node *networking.PeerNode, block *types.Block) {
	log.Println("Received confirm_req with block", block.Account.ToNanoAddress(), "from", node.NodeID.ToNodeAddress())
}

func (worker *ConfirmReqWorker) RequestsQueueProcessor() {
	for {
		// If a key is in HashPairsQueue then it should be in BlocksQueue as well
		for peer := range worker.HashPairsQueue {
			select {
			case pairs := <-worker.HashPairsQueue[peer]:
				worker.HandleHashPairRequest(peer, pairs)

			case block := <-worker.BlocksQueue[peer]:
				worker.HandleBlockRequest(peer, block)

			default:
				continue
			}
		}

		time.Sleep(time.Millisecond * 250)
	}
}

func (worker *ConfirmReqWorker) Start() {
	go worker.RequestsQueueProcessor()
}

func (worker *ConfirmReqWorker) ProcessHashPairs(peer *networking.PeerNode, pairs []*packets.HashPair) {
	worker.HashPairsQueue[peer] <- pairs
}

func (worker *ConfirmReqWorker) ProcessBlock(peer *networking.PeerNode, block *types.Block) {
	worker.BlocksQueue[peer] <- block
}

func (worker *ConfirmReqWorker) RegisterNewPeer(peer *networking.PeerNode) {
	worker.Mutex.Lock()
	worker.HashPairsQueue[peer] = make(chan []*packets.HashPair)
	worker.BlocksQueue[peer] = make(chan *types.Block)
	worker.Mutex.Unlock()
}

func NewConfirmReqWorker(srv *P2P) *ConfirmReqWorker {
	return &ConfirmReqWorker{
		P2PServer:  srv,
		LocalVotes: make(map[VoteKey]*packets.VoteByHashes),

		HashPairsQueue: make(map[*networking.PeerNode]chan []*packets.HashPair), // 1024 is queue size
		BlocksQueue:    make(map[*networking.PeerNode]chan *types.Block),        // 1024 queue size
	}
}
