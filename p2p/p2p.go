package p2p

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"strconv"
	"sync"
	"syscall"

	"github.com/Shryder/gnano/database"
	"github.com/Shryder/gnano/p2p/networking"
	"github.com/Shryder/gnano/p2p/packets"

	"time"
)

type P2P struct {
	Config        *Config
	Server        *net.Listener
	Database      database.Database
	VotingEnabled bool

	Peers      []*networking.PeerNode
	peersMutex sync.Mutex

	NodeKeyPair        NodeKeyPair
	NodeStartTimestamp uint64

	Workers WorkersManager
}

func New(cfg *Config) *P2P {
	srv := &P2P{
		Config:             cfg,
		NodeStartTimestamp: uint64(time.Now().UnixMilli()),
		VotingEnabled:      false,
	}

	workersManager := NewWorkerManager(srv)
	srv.Workers = *workersManager

	return srv
}

func (srv *P2P) ValidateConnection(conn net.Conn) error {
	srv.peersMutex.Lock()
	peer_count := len(srv.Peers)
	srv.peersMutex.Unlock()

	if peer_count >= int(srv.Config.P2P.MaxPeers) {
		conn.Close()
		return fmt.Errorf("Dropping connection with %s as we have reached the max limit of %d", conn.RemoteAddr().String(), srv.Config.P2P.MaxPeers)
	}

	return nil
}

func (srv *P2P) HandleMessage(reader packets.PacketReader, header packets.Header, peer *networking.PeerNode) error {
	log.Println("Received message", header.MessageType, "from", peer.NodeID.ToNodeAddress())

	switch header.MessageType {
	case packets.PACKET_TYPE_KEEPALIVE:
		return srv.HandleKeepAlive(reader, &header, peer)
	case packets.PACKET_TYPE_CONFIRM_REQ:
		return srv.HandleConfirmReq(reader, &header, peer)
	case packets.PACKET_TYPE_CONFIRM_ACK:
		return srv.HandleConfirmAck(reader, &header, peer)
	case packets.PACKET_TYPE_TELEMETRY_REQ:
		return srv.HandleTelemetryReq(reader, &header, peer)
	case packets.PACKET_TYPE_TELEMETRY_ACK:
		return srv.HandleTelemetryAck(reader, &header, peer)
	}

	return errors.New("Unsupported packet type: " + strconv.FormatUint(uint64(header.MessageType), 10))
}

func (srv *P2P) HandleConnection(conn net.Conn) {
	remoteIP := conn.RemoteAddr().String()

	err := srv.ValidateConnection(conn)
	if err != nil {
		log.Println("Connection validation failed:", err)
		return
	}

	log.Println("Successfully established connection with", remoteIP)
	reader := bufio.NewReader(conn)

	peer, err := srv.makeHandshake(conn, reader)
	if err != nil {
		log.Println("Error making initial handshake with:", remoteIP, err)
		return
	}

	log.Println("Successfully finished handshake with", peer.NodeID.ToNodeAddress())

	srv.peersMutex.Lock()
	srv.Peers = append(srv.Peers, peer)
	srv.peersMutex.Unlock()

	// Register new peer on the components that care
	srv.Workers.ConfirmReq.RegisterNewPeer(peer)

	err = srv.SendTelemetryReq(peer)
	if err != nil {
		log.Println("Error sending telemetry_req:", err)
		return
	}

	for {
		header, err := srv.ReadHeader(reader)
		if err != nil {
			if err == io.EOF {
				log.Println("Peer", peer.NodeID.ToNodeAddress(), remoteIP, "closed the connection.")
			} else if errors.Is(err, syscall.ECONNRESET) {
				log.Println("Peer", peer.NodeID.ToNodeAddress(), remoteIP, "force closed the connection.")
			} else {
				log.Println("Error reading from peer", peer.NodeID.ToNodeAddress(), remoteIP, ":", err, "disconnecting...")
			}

			break
		}

		err = srv.HandleMessage(packets.PacketReader{Buffer: reader}, header, peer)
		if err != nil {
			log.Println("Disconnecting. Error handling message from peer", peer.NodeID.ToNodeAddress(), remoteIP, ":", err)
			break
		}
	}
}

func (srv *P2P) StartListening() {
	listener, err := net.Listen("tcp", srv.Config.P2P.ListenAddr)
	if err != nil {
		log.Println("Error listening on TCP", srv.Config.P2P.ListenAddr, err)
		return
	}

	log.Println("Successfully started listen server")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Error accepting TCP Connection:", err)
			continue
		}

		go srv.HandleConnection(conn)
	}
}

func (srv *P2P) ConnectToNode(ip string) {
	log.Println("Initiating connection with", ip, "...")

	conn, err := net.Dial("tcp", ip)
	if err != nil {
		log.Println("Couldn't initiate connection with", ip, err)
		return
	}

	srv.HandleConnection(conn)
}

func (srv *P2P) Start() {
	srv.Workers.StartWorkers()

	for _, ip := range srv.Config.P2P.TrustedNodes {
		go srv.ConnectToNode(ip)
	}

	srv.StartListening()
}

func (srv *P2P) GetSubsetOfPeers() int {
	return int(math.Sqrt(float64(len(srv.Peers))))
}

func (srv *P2P) LoadOrCreateNodeIdentity() error {
	node_public_key, node_private_key, err := srv.Database.LoadOrCreateNodeIdentity()
	if err != nil {
		return err
	}

	srv.NodeKeyPair = NodeKeyPair{
		PrivateKey: node_private_key,
		PublicKey:  node_public_key,
	}

	return nil
}

func (srv *P2P) ValidateAndStart(database database.Database) error {
	srv.Database = database

	// Example validation
	if srv.Config.P2P.MaxPeers == 0 {
		return errors.New("MaxPeers cannot be 0")
	}

	if len(srv.Config.NetworkId) != 2 {
		return errors.New("NetworkId must be 2 bytes.")
	}

	err := srv.LoadOrCreateNodeIdentity()
	if err != nil {
		return err
	}

	log.Println("Public Key:", hex.EncodeToString(srv.NodeKeyPair.PublicKey))
	log.Println("Starting p2p server")

	go srv.Start()

	return nil
}
