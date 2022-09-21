package packets

import (
	"github.com/Shryder/gnano/types"
)

type SendBlock struct {
	Previous    types.Hash
	Destination types.Address
	Balance     types.Amount
	Signature   types.Signature
	Work        types.Work
}

type ReceiveBlock struct {
	Previous  types.Hash
	Source    types.Address
	Signature types.Signature
	Work      types.Work
}

type StateBlock struct {
	Address        types.Address
	Previous       types.Hash
	Representative types.Address
	Balance        types.Amount
	Link           [32]byte
	Signature      types.Signature
	Work           types.Work
}

type ChangeBlock struct {
	Previous       types.Hash
	Representative types.Address
	Signature      types.Signature
	Work           types.Work
}

type OpenBlock struct {
	Source         types.Hash
	Representative types.Address
	Address        types.Address
	Signature      types.Signature
	Work           types.Work
}
