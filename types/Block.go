package types

type Block struct {
	Hash           *Hash      `json:"hash"`
	Previous       *Hash      `json:"previous"`
	Account        *Address   `json:"account"`
	Representative *Address   `json:"representative"`
	Balance        *Amount    `json:"balance"`
	Link           *Link      `json:"link"`
	Signature      *Signature `json:"signature"`
	Work           *Work      `json:"work"`
}
