package p2p

import "github.com/shryder/ed25519-blake2b"

type NodeKeyPair struct {
	PublicKey  *ed25519.PublicKey
	PrivateKey *ed25519.PrivateKey
}
