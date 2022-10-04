package p2p

import (
	"log"
	"math"
	"net"
	"sync"
	"time"

	"github.com/Shryder/gnano/p2p/networking"
)

type PeersManager struct {
	P2PServer *P2P

	Peers               map[string]*networking.PeerNode
	BootstrapPeersCount uint
	LivePeersCount      uint
	PeersMutex          sync.RWMutex
}

func NewPeersManager(srv *P2P) PeersManager {
	return PeersManager{
		Peers:     make(map[string]*networking.PeerNode),
		P2PServer: srv,
	}
}

func (manager *PeersManager) MaintainLivePeersCount(peer_count uint) {
	if peer_count >= manager.P2PServer.Config.P2P.MaxLivePeers {
		return
	}

	remaining_slots := manager.P2PServer.Config.P2P.MaxLivePeers - peer_count
	saved_peers, err := manager.P2PServer.Database.Backend.GetNodeIPs()
	if err != nil {
		log.Println("Error loading saved peers:", err)

		return
	}

	log.Println(remaining_slots, "remaining slots for live connections")
	for ip := range saved_peers {
		if remaining_slots == 0 {
			break
		}

		already_peered := manager.ConnectToNode(ip, false)
		if !already_peered {
			remaining_slots--
		}
	}
}

func (manager *PeersManager) MaintainBootstrapPeersCount(peer_count uint) {
	if peer_count >= manager.P2PServer.Config.P2P.MaxBootstrapPeers {
		return
	}

	remaining_slots := manager.P2PServer.Config.P2P.MaxBootstrapPeers - peer_count
	saved_peers, err := manager.P2PServer.Database.Backend.GetNodeIPs()
	if err != nil {
		log.Println("Error loading saved peers:", err)
		return
	}

	log.Println(remaining_slots, "remaining slots for bootstrap connections")
	for ip := range saved_peers {
		if remaining_slots == 0 {
			break
		}

		already_peered := manager.ConnectToNode(ip, true)
		if !already_peered {
			remaining_slots--
		}
	}
}

func (manager *PeersManager) MaintainPeersCount() {
	for {
		time.Sleep(time.Second * 3)

		live_peers_count, bootstrap_peers_count := manager.GetPeersCount()
		log.Println("Currently connected to a total of", live_peers_count, "live peers and", bootstrap_peers_count, "bootstrap peers, total:", len(manager.Peers))

		manager.MaintainLivePeersCount(live_peers_count)
		manager.MaintainBootstrapPeersCount(bootstrap_peers_count)
	}
}

func (manager *PeersManager) ConnectToNode(ip string, bootstrap_connection bool) bool {
	manager.PeersMutex.RLock()
	_, already_peered := manager.Peers[ip]
	manager.PeersMutex.RUnlock()

	if !already_peered {
		go func() {
			conn, err := net.Dial("tcp", ip)
			if err != nil {
				log.Println("Couldn't initiate connection with", ip, err)
				return
			}

			manager.P2PServer.HandleConnection(conn, false, bootstrap_connection)
		}()
	}

	return already_peered
}

func (manager *PeersManager) Start() {
	go manager.MaintainPeersCount()
}

func (manager *PeersManager) RegisterPeer(peer *networking.PeerNode) {
	remoteIP := peer.Conn.RemoteAddr().String()
	log.Println("Registering peer", remoteIP)

	manager.PeersMutex.Lock()
	defer manager.PeersMutex.Unlock()

	_, found := manager.Peers[remoteIP]
	if !found {
		if peer.BootstrapConnection {
			manager.BootstrapPeersCount++
		} else {
			manager.LivePeersCount++
		}

		manager.Peers[remoteIP] = peer
	}
}

func (manager *PeersManager) UnregisterPeer(peer *networking.PeerNode) {
	remoteIP := peer.Conn.RemoteAddr().String()

	manager.PeersMutex.Lock()
	defer manager.PeersMutex.Unlock()

	_, found := manager.Peers[remoteIP]
	if found {
		if peer.BootstrapConnection {
			manager.BootstrapPeersCount--
		} else {
			manager.LivePeersCount--
		}

		delete(manager.Peers, remoteIP)
	}
}

func (manager *PeersManager) GetLivePeersCount() uint {
	manager.PeersMutex.RLock()
	defer manager.PeersMutex.RUnlock()

	return manager.LivePeersCount
}

func (manager *PeersManager) GetBootstrapPeersCount() uint {
	manager.PeersMutex.RLock()
	defer manager.PeersMutex.RUnlock()

	return manager.BootstrapPeersCount
}

func (manager *PeersManager) GetPeersCount() (uint, uint) {
	manager.PeersMutex.RLock()
	defer manager.PeersMutex.RUnlock()

	return manager.LivePeersCount, manager.BootstrapPeersCount
}

func (manager *PeersManager) GetSubsetOfLivePeers() int {
	return int(math.Sqrt(float64(manager.GetLivePeersCount())))
}
