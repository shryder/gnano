package p2p

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Shryder/gnano/p2p/networking"
	"github.com/Shryder/gnano/p2p/packets"
)

type PeersManager struct {
	P2PServer *P2P

	Logger         *log.Logger
	BootstrapPeers map[string]*networking.PeerNode
	LivePeers      map[string]*networking.PeerNode
	PeersMutex     sync.RWMutex
}

func NewPeersManager(srv *P2P) PeersManager {
	logger := log.New(os.Stdout, "[Peers Manager] ", log.Ltime)
	if !srv.Config.P2P.Logs.PeersManager {
		logger.SetOutput(ioutil.Discard)
	}

	return PeersManager{
		Logger:         logger,
		LivePeers:      make(map[string]*networking.PeerNode),
		BootstrapPeers: make(map[string]*networking.PeerNode),
		P2PServer:      srv,
	}
}

func (manager *PeersManager) GetSavedPeers() (map[string]uint, error) {
	return map[string]uint{
		"168.119.169.116:17075": 1670022243,
		"168.119.169.134:17075": 1670022243,
		"168.119.169.220:17075": 1670022243,
		"168.119.169.221:17075": 1670022243,
	}, nil
	// return manager.P2PServer.Database.Backend.GetNodeIPs()
}

func (manager *PeersManager) MaintainLivePeersCount(peer_count uint) {
	if peer_count >= manager.P2PServer.Config.P2P.MaxLivePeers {
		return
	}

	remaining_slots := manager.P2PServer.Config.P2P.MaxLivePeers - peer_count
	saved_peers, err := manager.GetSavedPeers()
	if err != nil {
		manager.Logger.Println("Error loading saved peers:", err)

		return
	}

	manager.Logger.Println(remaining_slots, "remaining slots for live connections")
	for ip := range saved_peers {
		if remaining_slots == 0 {
			break
		}

		err := manager.ConnectToNode(ip, false)
		if err != nil {
			log.Println("Error connecting to live node:", err)
			continue
		}

		remaining_slots--
	}
}

func (manager *PeersManager) MaintainBootstrapPeersCount(peer_count uint) {
	if peer_count >= manager.P2PServer.Config.P2P.MaxBootstrapPeers {
		return
	}

	remaining_slots := manager.P2PServer.Config.P2P.MaxBootstrapPeers - peer_count
	saved_peers, err := manager.GetSavedPeers()
	if err != nil {
		manager.Logger.Println("Error loading saved peers:", err)
		return
	}

	manager.Logger.Println(remaining_slots, "remaining slots for bootstrap connections")
	for ip := range saved_peers {
		if remaining_slots == 0 {
			break
		}

		err = manager.ConnectToNode(ip, true)
		if err != nil {
			log.Println("Error connecting to bootstrap node:", err)
			continue
		}

		remaining_slots--
	}
}

func (manager *PeersManager) MaintainPeersCount() {
	for {
		time.Sleep(time.Second * 3)

		live_peers_count, bootstrap_peers_count := manager.GetPeersCount()
		manager.Logger.Println("Currently connected to a total of", live_peers_count, "live peers and", bootstrap_peers_count, "bootstrap peers, total:", live_peers_count+bootstrap_peers_count)

		manager.MaintainLivePeersCount(live_peers_count)
		manager.MaintainBootstrapPeersCount(bootstrap_peers_count)
	}
}

// Connects to the provided ip and returns after dialing attempt, will not return an error if we are already peered with this node
func (manager *PeersManager) ConnectToNode(ip string, bootstrap_connection bool) error {
	manager.PeersMutex.RLock()
	_, already_peered_live := manager.LivePeers[ip]
	_, already_peered_bootstrap := manager.BootstrapPeers[ip]
	manager.PeersMutex.RUnlock()

	if bootstrap_connection && already_peered_bootstrap {
		log.Println("Tried to peer with a node that we were already peered with:", ip, bootstrap_connection)
		return nil
	}

	if !bootstrap_connection && already_peered_live {
		log.Println("Tried to peer with a node that we were already peered with:", ip, bootstrap_connection)
		return nil
	}

	dialer := net.Dialer{Timeout: time.Second * 3}
	conn, err := dialer.Dial("tcp", ip)
	if err != nil {
		manager.Logger.Println("Couldn't initiate connection with:", ip, err)
		return err
	}

	go manager.P2PServer.HandleConnection(conn, false, bootstrap_connection)

	return nil
}

func (manager *PeersManager) ConnectToTrustedNodesIfNeeded() {
	// Connect to configured trusted nodes if we don't have any saved peers in the database
	saved_nodes, err := manager.GetSavedPeers()
	if err != nil {
		manager.Logger.Println("Error retrieving saved peers:", err)
		return
	}

	if len(saved_nodes) == 0 {
		for _, ip := range manager.P2PServer.Config.P2P.TrustedNodes {
			manager.ConnectToNode(ip, false)
		}
	}
}

func (manager *PeersManager) Start() {
	manager.ConnectToTrustedNodesIfNeeded()

	go manager.MaintainPeersCount()
}

func (manager *PeersManager) RegisterPeer(peer *networking.PeerNode) {
	remoteIP := peer.Conn.RemoteAddr().String()
	manager.Logger.Println("Registering peer", peer.Alias, "bootstrap_connection:", peer.BootstrapConnection)

	manager.PeersMutex.Lock()
	defer manager.PeersMutex.Unlock()

	if peer.BootstrapConnection {
		_, found := manager.BootstrapPeers[remoteIP]
		if !found {
			manager.BootstrapPeers[remoteIP] = peer
		} else {
			log.Println("Tried to register a bootstrap peer that was already registered:", peer.Alias)
		}
	} else {
		_, found := manager.LivePeers[remoteIP]
		if !found {
			manager.LivePeers[remoteIP] = peer
		} else {
			log.Println("Tried to register a live peer that was already registered:", peer.Alias)
		}
	}
}

func (manager *PeersManager) UnregisterPeer(peer *networking.PeerNode) {
	remoteIP := peer.Conn.RemoteAddr().String()
	manager.Logger.Println("Unregister peer", peer.Alias, peer.BootstrapConnection)

	manager.PeersMutex.Lock()
	defer manager.PeersMutex.Unlock()

	if peer.BootstrapConnection {
		_, found := manager.BootstrapPeers[remoteIP]
		if found {
			delete(manager.BootstrapPeers, remoteIP)
		} else {
			log.Println("Tried to unregister a bootstrap peer that was not registered:", peer.Alias)
		}
	} else {
		_, found := manager.LivePeers[remoteIP]
		if found {
			delete(manager.LivePeers, remoteIP)
		} else {
			log.Println("Tried to unregister a live peer that was not registered:", peer.Alias)
		}
	}
}

func (manager *PeersManager) GetLivePeersCount() uint {
	manager.PeersMutex.RLock()
	defer manager.PeersMutex.RUnlock()

	return uint(len(manager.LivePeers))
}

func (manager *PeersManager) GetBootstrapPeersCount() uint {
	manager.PeersMutex.RLock()
	defer manager.PeersMutex.RUnlock()

	return uint(len(manager.BootstrapPeers))
}

func (manager *PeersManager) LogMessage(peer *networking.PeerNode, message string) {
	fileName := strings.ReplaceAll(peer.Conn.RemoteAddr().String(), ":", "_")
	if peer.BootstrapConnection {
		fileName += "_bootstrap"
	}

	f, err := os.OpenFile("./logs/"+fileName+".log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
		return
	}

	defer f.Close()

	log_text := message + "\n"
	if _, err := f.WriteString(log_text); err != nil {
		log.Println(err)
	}
}

func (manager *PeersManager) LogPacket(peer *networking.PeerNode, header packets.Header, data []byte, incoming bool) {
	logText := fmt.Sprintf("%s %+v %s\n", header.MessageType.ToString(), header, hex.EncodeToString(data))

	if header.MessageType == packets.PACKET_TYPE_CONFIRM_ACK && incoming {
		logText = fmt.Sprintf("%s %+v %s\n", header.MessageType.ToString(), header, string(data))
	}

	if incoming {
		logText = "[IN] " + logText
	} else {
		logText = "[OUT] " + logText
	}

	// prepend ts
	logText = "[" + time.Now().Format("2006-01-02 15:04:05.000000") + "] " + logText

	manager.LogMessage(peer, logText)
}

func (manager *PeersManager) GetPeersCount() (uint, uint) {
	manager.PeersMutex.RLock()
	defer manager.PeersMutex.RUnlock()

	return uint(len(manager.LivePeers)), uint(len(manager.BootstrapPeers))
}

func (manager *PeersManager) GetSubsetOfLivePeers() int {
	return int(math.Sqrt(float64(manager.GetLivePeersCount())))
}
