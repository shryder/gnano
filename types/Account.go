package types

import "math/big"

type Hash [32]byte

type Account struct {
	Address        Address
	Balance        *big.Int
	Height         *big.Int
	Representative Address
	Frontier       Hash
}
