package p2p

type P2PConfig struct {
	MaxLivePeers      uint
	MaxBootstrapPeers uint
	TrustedNodes      []string
	StaticNodes       []string
	ListenAddr        string
}

type ConsensusConfig struct {
	TrustedPRs map[string]bool
}

type Config struct {
	NetworkId    string
	GenesisBlock string
	Consensus    ConsensusConfig

	P2P P2PConfig
}
