package types

import "encoding/hex"

type Signature [64]byte

func (sig *Signature) ToHexString() string {
	return hex.EncodeToString(sig[:])
}
