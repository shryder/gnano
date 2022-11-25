package types

import (
	"encoding/hex"
	"strings"
)

type Signature [64]byte

func (sig *Signature) ToHexString() string {
	return hex.EncodeToString(sig[:])
}

func (signature Signature) MarshalJSON() ([]byte, error) {
	return []byte(`"` + hex.EncodeToString(signature[:]) + `"`), nil
}

func (signature Signature) UnmarshalJSON(signature_bytes []byte) error {
	decoded, err := hex.DecodeString(strings.Trim(string(signature_bytes), `"`))
	if err != nil {
		return err
	}

	copy(signature[:], decoded)

	return nil
}
