package types

import (
	"encoding/hex"
	"strings"
)

type Link [32]byte

func (link *Link) MarshalJSON() ([]byte, error) {
	return []byte(`"` + hex.EncodeToString(link[:]) + `"`), nil
}

func (link Link) UnmarshalJSON(link_hex []byte) error {
	decoded, err := hex.DecodeString(strings.Trim(string(link_hex), `"`))
	if err != nil {
		return err
	}

	copy(link[:], decoded)

	return nil
}

func (link *Link) ToHexString() string {
	return hex.EncodeToString(link[:])
}

func LinkFromString(link_str string) (*Link, error) {
	link_slice, err := hex.DecodeString(link_str)
	if err != nil {
		return nil, err
	}

	link := new(Link)
	copy(link[:], link_slice)

	return link, nil
}
