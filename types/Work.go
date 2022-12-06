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

func (work *Work) ToHexString() string {
	return hex.EncodeToString(work[:])
}

func WorkFromString(work_str string) (*Work, error) {
	work_slice, err := hex.DecodeString(work_str)
	if err != nil {
		return nil, err
	}

	work := new(Work)
	copy(work[:], work_slice)

	return work, nil
}
