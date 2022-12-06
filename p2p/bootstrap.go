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

func (manager *BootstrapDataManager) AddUnknownBlockHash(hash *types.Hash) {
	manager.NeedBlockBodyMutex.Lock()
	defer manager.NeedBlockBodyMutex.Unlock()

	manager.NeedBlockBody[*hash] = true
}

func (manager *BootstrapDataManager) FoundBlockBody(hash types.Hash) {
	manager.NeedBlockBodyMutex.Lock()
	defer manager.NeedBlockBodyMutex.Unlock()

	delete(manager.NeedBlockBody, hash)
}

func (manager *BootstrapDataManager) GetMissingBlock() *types.Hash {
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

	err := srv.StartBootstrapping(conn, packets.PacketReader{Buffer: reader}, peer)
	if err != nil {
		log.Println("Error with bootstrap connection:", srv.FormatConnReadError(err, peer))
		return
	}
}

func (srv *P2P) StartBootstrapping(conn net.Conn, reader packets.PacketReader, peer *networking.PeerNode) error {
	// First bulk pull will be requesting genesis address
	err := srv.SendBulkPull(peer, types.Hash(*srv.GenesisBlock.Account), types.Hash{})
	if err != nil {
		return err
	}

	err = srv.HandleBulkPullResponse(reader, peer, types.Hash(*srv.GenesisBlock.Account), types.Hash{})
	if err != nil {
		return err
	}

	// Continuously bulk_pull blocks that we know we do not have
	for {
		time.Sleep(time.Millisecond * 250)

		block_to_bulk_pull := srv.BootstrapDataManager.GetMissingBlock()
		if block_to_bulk_pull == nil {
			// Try finding a random address in the ledger to pull
			random_address := srv.Database.Backend.GetRandomAccountAddress()
			random_address_hash := types.Hash(*random_address)
			block_to_bulk_pull = &random_address_hash

			log.Println("Bulk pulling random address:", random_address.ToNanoAddress())
			if block_to_bulk_pull == nil {
				continue
			}
		}

		log.Println("requesting bulk_pull for:", block_to_bulk_pull.ToHexString())
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
