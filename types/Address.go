package types

import "encoding/hex"

type Address [32]byte

func (address *Address) ToHex() string {
	return hex.EncodeToString(address[0:32])
}
