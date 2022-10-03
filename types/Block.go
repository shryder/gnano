package types

type Block struct {
	Previous       *Hash      `json:"hash"`
	Account        *Address   `json:"account"`
	Representative *Address   `json:"representative"`
	Balance        *Amount    `json:"balance"`
	Link           *Link      `json:"link"`
	Signature      *Signature `json:"signature"`
	Work           *Work      `json:"work"`
}

func (block *Block) Hash() *Hash {
	return nil
}
