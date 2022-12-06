package types

import (
	"fmt"
	"math/big"
)

const (
	BLOCK_TYPE_SEND    byte = 0x02
	BLOCK_TYPE_RECEIVE byte = 0x03
	BLOCK_TYPE_OPEN    byte = 0x04
	BLOCK_TYPE_CHANGE  byte = 0x05
	BLOCK_TYPE_STATE   byte = 0x06
)

type Block struct {
	Type           byte       `json:"type"`
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

func (block *Block) IsOpenBlock() bool {
	return block.Previous.Cmp(&Hash{}) == 0
}

/*
	func (block Block) MarshalJSON() ([]byte, error) {
		return json.Marshal(JSONBlock{
			Hash:           block.Hash.ToHexString(),
			Previous:       block.Previous.ToHexString(),
			Account:        block.Account.ToNanoAddress(),
			Representative: block.Representative.ToNanoAddress(),
			Balance:        block.Balance.String(),
			Link:           block.Link.ToHexString(),
			Signature:      block.Signature.ToHexString(),
			Work:           block.Work.ToHexString(),
		})
	}

	func (block *Block) UnmarshalJSON(data []byte) error {
		var jsonBlock JSONBlock

		err := json.Unmarshal(data, &jsonBlock)
		if err != nil {
			return err
		}

		if jsonBlock.Type != "open" && jsonBlock.Type != "receive" && jsonBlock.Type != "change" && jsonBlock.Type != "send" {
			return fmt.Errorf("invalid block type %s", jsonBlock.Type)
		}

		block.Type = jsonBlock.Type
		block.Hash, err = StringToHash(jsonBlock.Hash)
		if err != nil {
			return err
		}

		block.Account, err = DecodeNanoAddress(jsonBlock.Account)
		if err != nil {
			return err
		}

		block.Previous, err = StringToHash(jsonBlock.Previous)
		if err != nil {
			return err
		}

		block.Representative, err = DecodeNanoAddress(jsonBlock.Representative)
		if err != nil {
			return err
		}

		block.Balance, err = AmountFromString(jsonBlock.Balance)
		if err != nil {
			return err
		}

		block.Link, err = LinkFromString(jsonBlock.Link)
		if err != nil {
			return err
		}

		block.Signature, err = SignatureFromString(jsonBlock.Signature)
		if err != nil {
			return err
		}

		block.Work, err = WorkFromString(jsonBlock.Work)
		if err != nil {
			return err
		}

		return nil
	}
*/

type JSONBlock struct {
	Type           string `json:"type"`
	Hash           string `json:"hash"`
	Previous       string `json:"previous"`
	Account        string `json:"account"`
	Representative string `json:"representative"`
	Balance        string `json:"balance"`
	Link           string `json:"link"`
	Signature      string `json:"signature"`
	Work           string `json:"work,omitempty"`
}

func (jsonBlock JSONBlock) ToBlock() (*Block, error) {
	var block Block
	var err error

	if jsonBlock.Type == "open" {
		block.Type = BLOCK_TYPE_OPEN
	} else if jsonBlock.Type == "change" {
		block.Type = BLOCK_TYPE_CHANGE
	} else if jsonBlock.Type == "send" {
		block.Type = BLOCK_TYPE_SEND
	} else if jsonBlock.Type == "receive" {
		block.Type = BLOCK_TYPE_RECEIVE
	} else if jsonBlock.Type == "state" {
		block.Type = BLOCK_TYPE_STATE
	} else {
		return nil, fmt.Errorf("invalid block type provided %s", jsonBlock.Type)
	}

	block.Hash, err = StringToHash(jsonBlock.Hash)
	if err != nil {
		return nil, err
	}

	block.Account, err = DecodeNanoAddress(jsonBlock.Account)
	if err != nil {
		return nil, err
	}

	block.Previous, err = StringToHash(jsonBlock.Previous)
	if err != nil {
		return nil, err
	}

	block.Representative, err = DecodeNanoAddress(jsonBlock.Representative)
	if err != nil {
		return nil, err
	}

	block.Balance, err = AmountFromString(jsonBlock.Balance)
	if err != nil {
		return nil, err
	}

	block.Link, err = LinkFromString(jsonBlock.Link)
	if err != nil {
		return nil, err
	}

	block.Signature, err = SignatureFromString(jsonBlock.Signature)
	if err != nil {
		return nil, err
	}

	block.Work, err = WorkFromString(jsonBlock.Work)
	if err != nil {
		return nil, err
	}

	return &block, nil
}
