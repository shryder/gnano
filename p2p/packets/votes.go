package packets

import "github.com/Shryder/gnano/types"

type HashPair struct {
	Hash *types.Hash
	Root *types.Hash
}

type VoteByHashes struct {
	Timestamp uint64
	HashPairs []*HashPair
	Account   *types.Address
	Signature *types.Signature
}
