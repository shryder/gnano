package p2p

import (
	"encoding/binary"
	"io"
	"time"

	"github.com/Shryder/gnano/p2p/packets"
)

type TelemetryData struct {
	Signature         [64]byte
	NodeID            [32]byte
	BlockCount        [8]byte
	CementedCount     [8]byte
	UncheckedCount    [8]byte
	AccountCount      [8]byte
	BandwidthCap      [8]byte
	PeerCount         [4]byte
	ProtocolVersion   byte
	Uptime            [8]byte
	GenesisBlock      [32]byte
	MajorVersion      byte
	MinorVersion      byte
	PatchVersion      byte
	PreReleaseVersion byte
	Maker             byte
	Timestamp         [8]byte
	ActiveDifficulty  [8]byte
}

func (srv *P2P) SendTelemetryAck(peer *PeerNode) error {
	var packet packets.PacketBody

	packet.WriteBE(srv.NodeKeyPair.PublicKey)                                 // node_id
	packet.WriteBE(uint64(0x93146))                                           // block count
	packet.WriteBE(uint64(0x93146))                                           // cemented count
	packet.WriteBE(uint64(0))                                                 // unchecked count
	packet.WriteBE(uint64(0x31333))                                           // account count
	packet.WriteBE(uint64(0))                                                 // bandwidth count
	packet.WriteBE(uint64(4))                                                 // peer count
	packet.WriteBE(packets.PROTOCOL_VERSION)                                  // protocol ver
	packet.WriteBE((uint64(time.Now().UnixMilli()) - srv.NodeStartTimestamp)) // uptime
	packet.WriteBE(srv.Config.GenesisBlock.ByteArray())                       // genesis block
	packet.WriteBE(byte(32))                                                  // major ver
	packet.WriteBE(byte(3))                                                   // minor ver
	packet.WriteBE(byte(0))                                                   // patch ver
	packet.WriteBE(byte(0))                                                   // preprelease ver
	packet.WriteBE(byte(1))                                                   // maker
	packet.WriteBE(uint64(time.Now().UnixMilli()))                            // timestamp
	packet.WriteBE(uint64(0x2552552552480000))                                // active difficulty

	_, err := peer.Conn.Write(srv.MakePacket(packets.PACKET_TYPE_TELEMETRY_ACK, 0, packet.Buff.Bytes()))
	return err
}

func (srv *P2P) SendTelemetryReq(peer *PeerNode) error {
	_, err := peer.Conn.Write(srv.MakePacket(packets.PACKET_TYPE_TELEMETRY_REQ, 0))
	return err
}

func (srv *P2P) HandleTelemetryReq(reader io.Reader, header *packets.Header, peer *PeerNode) error {
	return srv.SendTelemetryAck(peer)
}

func (srv *P2P) HandleTelemetryAck(reader io.Reader, header *packets.Header, peer *PeerNode) error {
	// TODO: use header.Extension.TelemetrySize()
	var packet TelemetryData
	err := binary.Read(reader, binary.BigEndian, &packet)
	if err != nil {
		return err
	}

	return nil
}
