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
	BootstrapDataManager   BootstrapDataManager

	NodeKeyPair        NodeKeyPair
	NodeStartTimestamp uint64

	Workers WorkersManager

	GenesisBlock *types.Block
}

func New(cfg *Config, genesisBlock *types.Block) *P2P {
	srv := &P2P{
		Config:             cfg,
		NodeStartTimestamp: uint64(time.Now().UnixMilli()),
		VotingEnabled:      false,
		GenesisBlock:       genesisBlock,
	}

	srv.Workers = NewWorkerManager(srv)
	srv.PeersManager = NewPeersManager(srv)
	srv.UncheckedBlocksManager = NewUncheckedBlocksManager(srv)
	srv.BootstrapDataManager = NewBootstrapDataManager()
	return srv
}

// Check if we are above max peer count
func (srv *P2P) ValidateIncomingConnection(conn net.Conn) error {
	peer_count := srv.PeersManager.GetLivePeersCount()
	if peer_count >= srv.Config.P2P.MaxLivePeers {
		conn.Close()
		return fmt.Errorf("dropping connection with %s as we have reached the max limit of %d live peers", conn.RemoteAddr().String(), srv.Config.P2P.MaxLivePeers)
	}

	return nil
}

func (srv *P2P) HandleMessage(reader packets.PacketReader, header packets.Header, peer *networking.PeerNode) error {
	log.Println("Received message", header.MessageType.ToString(), "from", peer.Alias)

	if header.MessageType != packets.PACKET_TYPE_CONFIRM_ACK {
		srv.PeersManager.LogPacket(peer, header, []byte{}, true)
	}

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
	if err == io.EOF {
		return fmt.Sprintln("Peer", peer.Alias, "closed the connection.")
	} else if errors.Is(err, syscall.ECONNRESET) {
		return fmt.Sprintln("Peer", peer.Alias, "force closed the connection.")
	}

	return fmt.Sprintln("Error reading from peer", peer.Alias, ":", err, "disconnecting...")
}

func (srv *P2P) WriteToPeer(peer *networking.PeerNode, message_type byte, extension packets.HeaderExtension, data ...[]byte) error {
	header, packet := srv.MakePacket(message_type, extension, data...)
	srv.PeersManager.LogPacket(peer, header, packet, false)

	return peer.Write(append(header.Serialize(), packet...))
}

func (srv *P2P) HandleRegularConnection(conn net.Conn, reader *bufio.Reader) {
	remoteIP := conn.RemoteAddr().String()
	peer, err := srv.makeHandshake(conn, reader)
	if err != nil {
		log.Println("Error making initial handshake with:", remoteIP, err)
		return
	}

	// Request telemetry from peer right after connecting
	// err = srv.SendTelemetryReq(peer)
	// if err != nil {
	// 	log.Println(srv.FormatConnReadError(err, peer))

	// 	return
	// }

	srv.RegisterPeer(peer)
	defer srv.UnregisterPeer(peer)

	srv.PeersManager.LogMessage(peer, "=========== CONNECTION ESTABLISHED ===========")

	// err = srv.SendConfirmAck(peer, []*packets.HashPair{{
	// 	Hash: srv.GenesisBlock.Hash,
	// 	Root: srv.GenesisBlock.Previous,
	// }})
	// if err != nil {
	// 	log.Println("Error sending confirm ack to peer", peer.Alias)
	// 	return
	// }

	for {
		header, err := srv.ReadHeader(reader)
		if err != nil {
			// srv.PeersManager.LogMessage(peer, fmt.Sprintf("Error reading header from peer: %s", srv.FormatConnReadError(err, peer)))
			log.Println("Error reading header:", srv.FormatConnReadError(err, peer))

			break
		}

		err = srv.HandleMessage(packets.PacketReader{Buffer: reader}, header, peer)
		if err != nil {
			// srv.PeersManager.LogMessage(peer, fmt.Sprintf("Disconnecting. Error handling message from peer: %+v", err))
			log.Println("Disconnecting. Error handling message from peer", peer.NodeID.ToNodeAddress(), remoteIP, ":", err)

			break
		}
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

	log.Println("Successfully established connection with", remoteIP, "bootstrap_connection:", bootstrap_connection, "incoming:", incoming)
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
	srv.Workers.Start()
	srv.PeersManager.Start()
	srv.UncheckedBlocksManager.Start()

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

	if len(srv.Config.NetworkId) != 2 {
		return errors.New("NetworkId must be 2 bytes")
	}

	err := srv.LoadOrCreateNodeIdentity()
	if err != nil {
		return err
	}

	log.Println("Public Key:", hex.EncodeToString(srv.NodeKeyPair.PublicKey))

	// Store genesis block in the ledger if it wasn't stored already.
	ledgerGenesisBlock := database.Backend.GetBlock(srv.GenesisBlock.Hash)
	if ledgerGenesisBlock == nil {
		if database.Backend.GetBlockCount() != 0 {
			return errors.New("genesis block not found locally yet the ledger had more than 1 block in it")
		}

		err := database.Backend.PutBlock(srv.GenesisBlock)
		if err != nil {
			return errors.New("error storing genesis block in ledger")
		}
	}

	log.Println("Starting p2p server")

	go srv.Start()

	return nil
}
