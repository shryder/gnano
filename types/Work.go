package types

import (
	"encoding/hex"
	"strings"
)

type Work [8]byte

func (work Work) MarshalJSON() ([]byte, error) {
	return []byte(`"` + hex.EncodeToString(work[:]) + `"`), nil
}

func (work Work) UnmarshalJSON(work_hex []byte) error {
	decoded, err := hex.DecodeString(strings.Trim(string(work_hex), `"`))
	if err != nil {
		return err
	}

	copy(work[:], decoded)

	return nil
}
