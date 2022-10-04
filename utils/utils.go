package utils

import (
	"github.com/Shryder/gnano/types"
	"golang.org/x/crypto/blake2b"
)

func Blake2BHash(data ...[]byte) *types.Hash {
	b2b_hash, _ := blake2b.New(32, nil)
	for _, item := range data {
		b2b_hash.Write(item)
	}

	hash_bytes := b2b_hash.Sum(nil)
	hash := new(types.Hash)
	copy(hash[:], hash_bytes)

	return hash
}
