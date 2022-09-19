package p2p

import "encoding/hex"

type P2PConfig struct {
	MaxPeers     uint
	TrustedNodes []string
	StaticNodes  []string
	ListenAddr   string
}

type GenesisBlock string

type Config struct {
	NetworkId    string // Try [2]byte
	GenesisBlock GenesisBlock

	P2P P2PConfig
}

func (genesis *GenesisBlock) ByteArray() []byte {
	// TODO: handle error
	arr, _ := hex.DecodeString(string(*genesis))
	return arr
}
