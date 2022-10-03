package types

import "math/big"

type Amount [16]byte

func (amount *Amount) BigInt() *big.Int {
	return new(big.Int).SetBytes((*amount)[:])
}
