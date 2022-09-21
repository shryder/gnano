package p2p

import "encoding/hex"

type P2PConfig struct {
	MaxPeers     uint
	TrustedNodes []string
	StaticNodes  []string
	ListenAddr   string
}

type Config struct {
	NetworkId    string
	GenesisBlock GenesisBlock

	P2P P2PConfig
}

type GenesisBlock string

func (genesis *GenesisBlock) ByteArray() []byte {
	// TODO: handle error
	arr, _ := hex.DecodeString(string(*genesis))
	return arr
}
