package types

import (
	"math/big"
)

type Block struct {
	Type           string     `json:"type"`
	Hash           *Hash      `json:"hash"`
	Previous       *Hash      `json:"previous"`
	Account        *Address   `json:"account"`
	Representative *Address   `json:"representative"`
	Balance        *Amount    `json:"balance"`
	Link           *Link      `json:"link"`
	Signature      *Signature `json:"signature"`
	Work           *Work      `json:"work"`
}

func (block *Block) Root() *Hash {
	if block.Previous.BigInt().Cmp(big.NewInt(0)) == 0 {
		return (*Hash)(block.Account)
	}

	return block.Previous
}

func (block *Block) Cmp(other_block *Block) int {
	return block.Hash.Cmp(other_block.Hash)
}
