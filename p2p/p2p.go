package p2p

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"syscall"

	"github.com/Shryder/gnano/database"
	"github.com/Shryder/gnano/p2p/networking"
	"github.com/Shryder/gnano/p2p/packets"
	"github.com/Shryder/gnano/types"

	"time"
)

type P2P struct {
	Config        *Config
	Server        *net.Listener
	Database      database.Database
	VotingEnabled bool

	PeersManager           PeersManager
	UncheckedBlocksManager UncheckedBlocksManager

	NodeKeyPair        NodeKeyPair
	NodeStartTimestamp uint64

	Workers      WorkersManager
	GenesisBlock types.Hash
}

func New(cfg *Config) *P2P {
	genesis_hash, err := types.StringToHash(string(cfg.GenesisBlock))
	if err != nil {
		panic(err)
	}

	srv := &P2P{
		Config:                 cfg,
		NodeStartTimestamp:     uint64(time.Now().UnixMilli()),
		UncheckedBlocksManager: NewUncheckedBlocksManager(),
		VotingEnabled:          false,
		GenesisBlock:           *genesis_hash,
	}

	srv.Workers = NewWorkerManager(srv)
	srv.PeersManager = NewPeersManager(srv)

	return srv
}

func (srv *P2P) ValidateIncomingConnection(conn net.Conn) error {
	peer_count := srv.PeersManager.GetLivePeersCount()
	if peer_count >= srv.Config.P2P.MaxLivePeers {
		conn.Close()
		return fmt.Errorf("Dropping connection with %s as we have reached the max limit of %d live peers", conn.RemoteAddr().String(), srv.Config.P2P.MaxLivePeers)
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

func (srv *P2P) RegisterPeer(peer *networking.PeerNode) {
	srv.PeersManager.RegisterPeer(peer)
	srv.Workers.ConfirmReq.RegisterNewPeer(peer)
	srv.Workers.ConfirmAck.RegisterNewPeer(peer)
}

func (srv *P2P) UnregisterPeer(peer *networking.PeerNode) {
	srv.PeersManager.UnregisterPeer(peer)
	srv.Workers.ConfirmReq.UnregisterNewPeer(peer)
	srv.Workers.ConfirmAck.UnregisterNewPeer(peer)
}

func (srv *P2P) FormatConnReadError(err error, peer *networking.PeerNode) string {
	nodeId := "BOOTSTRAP_CONNECTION"
	if !peer.BootstrapConnection {
		nodeId = peer.NodeID.ToNanoAddress()
	}

	if err == io.EOF {
		return fmt.Sprintln("Peer", nodeId, peer.Conn.RemoteAddr().String(), "closed the connection.")
	} else if errors.Is(err, syscall.ECONNRESET) {
		return fmt.Sprintln("Peer", nodeId, peer.Conn.RemoteAddr().String(), "force closed the connection.")
	}

	return fmt.Sprintln("Error reading from peer", nodeId, peer.Conn.RemoteAddr().String(), ":", err, "disconnecting...")
}

func (srv *P2P) HandleRegularConnection(conn net.Conn, reader *bufio.Reader) {
	remoteIP := conn.RemoteAddr().String()
	peer, err := srv.makeHandshake(conn, reader)
	if err != nil {
		log.Println("Error making initial handshake with:", remoteIP, err)
		return
	}

	log.Println("Successfully finished handshake with", peer.NodeID.ToNodeAddress())

	srv.RegisterPeer(peer)
	defer srv.UnregisterPeer(peer)

	// Request telemetry from peer right after connecting
	err = srv.SendTelemetryReq(peer)
	if err != nil {
		log.Println(srv.FormatConnReadError(err, peer))

		return
	}

	for {
		header, err := srv.ReadHeader(reader)
		if err != nil {
			log.Println(srv.FormatConnReadError(err, peer))

			break
		}

		err = srv.HandleMessage(packets.PacketReader{Buffer: reader}, header, peer)
		if err != nil {
			log.Println("Disconnecting. Error handling message from peer", peer.NodeID.ToNodeAddress(), remoteIP, ":", err)

			break
		}

		return
	}
}

func (srv *P2P) HandleBootstrapConnection(conn net.Conn, reader *bufio.Reader) {
	peer := networking.NewPeerNode(conn, nil, true)

	srv.RegisterPeer(peer)
	defer srv.UnregisterPeer(peer)

	genesis_wallet, _ := types.StringToHash("45C6FF9D1706D61F0821327752671BDA9F9ED2DA40326B01935AB566FB9E08ED")
	err := srv.SendBulkPull(peer, *genesis_wallet, types.Hash{})
	if err != nil {
		log.Println(srv.FormatConnReadError(err, peer))
		return
	}

	err = srv.HandleBulkPullResponse(packets.PacketReader{Buffer: reader}, peer)
	if err != nil {
		log.Println(srv.FormatConnReadError(err, peer))
		return
	}
}

func (srv *P2P) HandleConnection(conn net.Conn, incoming bool, bootstrap_connection bool) {
	remoteIP := conn.RemoteAddr().String()

	if incoming {
		err := srv.ValidateIncomingConnection(conn)
		if err != nil {
			log.Println("Connection validation failed:", err)
			return
		}
	}

	log.Println("Successfully established connection with", remoteIP)
	reader := bufio.NewReader(conn)

	if bootstrap_connection {
		srv.HandleBootstrapConnection(conn, reader)
	} else {
		srv.HandleRegularConnection(conn, reader)
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

		go srv.HandleConnection(conn, true, false)
	}
}

func (srv *P2P) Start() {
	srv.Workers.StartWorkers()
	srv.PeersManager.Start()

	for _, ip := range srv.Config.P2P.TrustedNodes {
		srv.PeersManager.ConnectToNode(ip, false)
	}

	srv.StartListening()
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
	if srv.Config.P2P.MaxLivePeers == 0 {
		return errors.New("MaxLivePeers cannot be 0")
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
