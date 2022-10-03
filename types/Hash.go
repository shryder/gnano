package types

import (
	"encoding/hex"
	"math/big"
)

type Hash [32]byte

func (hash *Hash) BigInt() *big.Int {
	return new(big.Int).SetBytes((*hash)[:])
}

func (hash *Hash) ToHexString() string {
	return hex.EncodeToString((*hash)[:])
}

func (hash *Hash) MarshalJSON() ([]byte, error) {
	return []byte(hash.ToHexString()), nil
}

func (hash *Hash) UnmarshalJSON(data []byte) error {
	hash_slice, err := hex.DecodeString(string(data))
	if err != nil {
		return err
	}

	copy(hash[:], hash_slice)

	return nil
}
