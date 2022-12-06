package types

import (
	"encoding/hex"
	"strings"
)

type Signature [64]byte

func (sig *Signature) ToHexString() string {
	return hex.EncodeToString(sig[:])
}

func (signature *Signature) MarshalJSON() ([]byte, error) {
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

func SignatureFromString(sig_str string) (*Signature, error) {
	sig_slice, err := hex.DecodeString(sig_str)
	if err != nil {
		return nil, err
	}

	sig := new(Signature)
	copy(sig[:], sig_slice)

	return sig, nil
}
