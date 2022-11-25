package types

import (
	"encoding/binary"
	"strings"

	. "lukechampine.com/uint128"
)

type Amount Uint128

func (u Amount) Bytes() []byte {
	return Uint128(u).Big().Bytes()
}

func (u Amount) Add(v Amount) Amount {
	return Amount(Uint128(v).Add(Uint128(u)))
}

func (u Amount) IsZero() bool {
	return Uint128(u).IsZero()
}

// Cmp compares u and v and returns:
//
//	-1 if u <  v
//	 0 if u == v
//	+1 if u >  v
func (u Amount) Cmp(v Amount) int {
	return Uint128(u).Cmp(Uint128(v))
}

func (amount *Amount) String() string {
	return Uint128(*amount).String()
}

func AmountFromBytesLE(amount_bytes []byte) Amount {
	return Amount(FromBytes(amount_bytes))
}

func AmountFromBytesBE(amount_bytes []byte) Amount {
	return Amount{
		Hi: binary.BigEndian.Uint64(amount_bytes[:8]),
		Lo: binary.BigEndian.Uint64(amount_bytes[8:]),
	}
}

func AmountFromString(amount_str string) (Amount, error) {
	amount, err := FromString(amount_str)
	if err != nil {
		return Amount{}, err
	}

	return Amount(amount), nil
}

func (amount Amount) MarshalJSON() ([]byte, error) {
	return []byte(`"` + Uint128(amount).String() + `"`), nil
}

func (amount *Amount) UnmarshalJSON(data []byte) error {
	amount_uint128, err := FromString(strings.Trim(string(data), `"`))
	if err != nil {
		return err
	}

	amount.Hi = amount_uint128.Hi
	amount.Lo = amount_uint128.Lo

	return nil
}
