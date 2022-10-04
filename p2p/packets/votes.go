package packets

import (
	"encoding/binary"
	"math"

	"github.com/Shryder/gnano/types"
)

type HashPair struct {
	Hash *types.Hash
	Root *types.Hash
}

// type VoteByHashes struct {
// 	Timestamp TimestampAndVoteDuration
// 	HashPairs []*HashPair
// 	Account   *types.Address
// 	Signature *types.Signature
// }

type ConfirmAckByHashes struct {
	Account                  *types.Address
	Signature                *types.Signature
	TimestampAndVoteDuration *TimestampAndVoteDuration

	Hashes *[]*types.Hash
}

type TimestampAndVoteDuration [8]byte

func (tvd *TimestampAndVoteDuration) Uint64() uint64 {
	return binary.LittleEndian.Uint64(tvd[:])
}

func (tvd *TimestampAndVoteDuration) IsFinalVote() bool {
	return tvd.Uint64() == math.MaxUint64
}

func (pair *HashPair) ToBytes() []byte {
	return append((*pair.Hash)[:], (*pair.Root)[:]...)
}
