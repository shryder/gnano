package packets

import (
	"github.com/Shryder/gnano/types"
	"github.com/Shryder/gnano/utils"
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

func ParseSendBlock(data []byte) *types.Block {
	if len(data) != (32 + 32 + 16 + 64 + 8) {
		return nil
	}

	var previous types.Hash
	var destination types.Link
	var balance types.Amount
	var signature types.Signature
	var work types.Work

	copy(previous[:], data[0:32])
	copy(destination[:], data[32:64])
	balance = types.AmountFromBytesBE(data[64:80])
	copy(signature[:], data[80:144])
	copy(work[:], data[144:152])

	hash := utils.Blake2BHash(previous[:], destination[:], data[64:80])

	return &types.Block{
		Hash:      hash,
		Previous:  &previous,
		Link:      &destination,
		Balance:   &balance,
		Signature: &signature,
		Work:      &work,
	}
}

func ParseReceiveBlock(data []byte) *types.Block {
	if len(data) != (32 + 32 + 64 + 8) {
		return nil
	}

	var previous types.Hash
	var source types.Link
	var signature types.Signature
	var work types.Work

	copy(previous[:], data[0:32])
	copy(source[:], data[32:64])
	copy(signature[:], data[64:128])

	hash := utils.Blake2BHash(previous[:], source[:])

	return &types.Block{
		Type:      types.BLOCK_TYPE_RECEIVE,
		Hash:      hash,
		Previous:  &previous,
		Link:      &source,
		Signature: &signature,
		Work:      &work,
	}
}

func ParseOpenBlock(data []byte) *types.Block {
	if len(data) != (32 + 32 + 32 + 64 + 8) {
		return nil
	}

	var previous types.Hash
	var source types.Link
	var account types.Address
	var signature types.Signature
	var work types.Work
	var representative types.Address

	copy(previous[:], make([]byte, 32)) // `Previous` is 0 for open blocks

	copy(source[:], data[0:32])
	copy(representative[:], data[32:64])
	copy(account[:], data[64:96])
	copy(signature[:], data[96:160])
	copy(work[:], data[160:168])

	hash := utils.Blake2BHash(source[:], representative[:], account[:])

	return &types.Block{
		Type:           types.BLOCK_TYPE_OPEN,
		Hash:           hash,
		Account:        &account,
		Previous:       &previous,
		Link:           &source,
		Signature:      &signature,
		Representative: &representative,
		Work:           &work,
	}
}

func ParseChangeBlock(data []byte) *types.Block {
	if len(data) != (32 + 32 + 64 + 8) {
		return nil
	}

	var previous types.Hash
	var representative types.Address
	var signature types.Signature
	var work types.Work

	copy(previous[:], data[0:32])
	copy(representative[:], data[32:64])
	copy(signature[:], data[64:128])
	copy(work[:], data[128:136])

	hash := utils.Blake2BHash(previous[:], representative[:])

	return &types.Block{
		Type:           types.BLOCK_TYPE_CHANGE,
		Hash:           hash,
		Previous:       &previous,
		Signature:      &signature,
		Representative: &representative,
		Work:           &work,
	}
}

func ParseStateBlock(data []byte) *types.Block {
	if len(data) != (32 + 32 + 32 + 16 + 32 + 64 + 8) {
		return nil
	}

	var account types.Address
	var previous types.Hash
	var representative types.Address
	var balance types.Amount
	var link types.Link
	var signature types.Signature
	var work types.Work

	copy(account[:], data[0:32])
	copy(previous[:], data[32:64])
	copy(representative[:], data[64:96])
	balance = types.AmountFromBytesBE(data[96:112])
	copy(link[:], data[112:144])
	copy(signature[:], data[144:208])
	copy(work[:], data[208:216])

	state_block_header := append(make([]byte, 31), 0x06)
	hash := utils.Blake2BHash(
		state_block_header,
		account[:],
		previous[:],
		representative[:],
		data[96:112],
		link[:],
	)

	return &types.Block{
		Type:           types.BLOCK_TYPE_STATE,
		Hash:           hash,
		Account:        &account,
		Previous:       &previous,
		Representative: &representative,
		Balance:        &balance,
		Link:           &link,
		Signature:      &signature,
		Work:           &work,
	}
}
