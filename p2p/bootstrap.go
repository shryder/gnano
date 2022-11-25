package p2p

import (
	"bufio"
	"log"
	"net"
	"sync"
	"time"

	"github.com/Shryder/gnano/p2p/networking"
	"github.com/Shryder/gnano/p2p/packets"
	"github.com/Shryder/gnano/types"
)

type BootstrapDataManager struct {
	NeedBlockBody      map[types.Hash]bool
	NeedBlockBodyMutex sync.RWMutex
}

func NewBootstrapDataManager() BootstrapDataManager {
	return BootstrapDataManager{
		NeedBlockBody:      make(map[types.Hash]bool),
		NeedBlockBodyMutex: sync.RWMutex{},
	}
}

func (manager *BootstrapDataManager) FoundBlockWithoutBody(hash_pairs [][]byte) {
	manager.NeedBlockBodyMutex.Lock()
	defer manager.NeedBlockBodyMutex.Unlock()

	for _, pair := range hash_pairs {
		var hash types.Hash
		var root types.Hash

		copy(hash[:], pair[0:32])
		copy(root[:], pair[32:64])

		manager.NeedBlockBody[hash] = true
		manager.NeedBlockBody[root] = true
	}
}

func (manager *BootstrapDataManager) FoundBlockBody(hash types.Hash) {
	manager.NeedBlockBodyMutex.Lock()
	defer manager.NeedBlockBodyMutex.Unlock()

	delete(manager.NeedBlockBody, hash)
}

func (manager *BootstrapDataManager) GetBlock() *types.Hash {
	manager.NeedBlockBodyMutex.RLock()
	defer manager.NeedBlockBodyMutex.RUnlock()

	for block := range manager.NeedBlockBody {
		return &block
	}

	return nil
}

func (srv *P2P) HandleBootstrapConnection(conn net.Conn, reader *bufio.Reader) {
	peer := networking.NewPeerNode(conn, nil, true)

	srv.RegisterPeer(peer)
	defer srv.UnregisterPeer(peer)

	err := srv.StartBootstrapingFromGenesis(conn, packets.PacketReader{Buffer: reader}, peer)
	if err != nil {
		log.Println("Error bootstrapping from genesis:", srv.FormatConnReadError(err, peer))
		return
	}
}

func (srv *P2P) StartBootstrapingFromGenesis(conn net.Conn, reader packets.PacketReader, peer *networking.PeerNode) error {
	// First bulk pull will be requesting genesis address
	genesis_address, _ := types.StringToHash("45C6FF9D1706D61F0821327752671BDA9F9ED2DA40326B01935AB566FB9E08ED")
	err := srv.SendBulkPull(peer, *genesis_address, types.Hash{})
	if err != nil {
		return err
	}

	err = srv.HandleBulkPullResponse(reader, peer, *genesis_address, types.Hash{})
	if err != nil {
		return err
	}

	// Continuously bulk_pull blocks that we know we did not have
	for {
		time.Sleep(time.Second * 1)

		block_to_bulk_pull := srv.BootstrapDataManager.GetBlock()
		if block_to_bulk_pull == nil {
			continue
		}

		err := srv.SendBulkPull(peer, *block_to_bulk_pull, types.Hash{})
		if err != nil {
			return err
		}

		err = srv.HandleBulkPullResponse(reader, peer, *block_to_bulk_pull, types.Hash{})
		if err != nil {
			return err
		}
	}
}
