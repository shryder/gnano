package utils

import (
	"encoding/hex"

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

func HashPairToString(hash_pairs [][]byte) []string {
	arr := make([]string, len(hash_pairs))
	for i, pair := range hash_pairs {
		arr[i] = "[" + hex.EncodeToString(pair[0:32]) + `, ` + hex.EncodeToString(pair[32:64]) + "]"
	}

	return arr
}
