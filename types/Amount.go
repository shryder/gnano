package types

import (
	. "lukechampine.com/uint128"
)

type Amount Uint128

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
	amount_str := string(data)[1:]              // remove leading double quotes
	amount_str = amount_str[:len(amount_str)-1] // remove trailing double quotes

	amount_uint128, err := FromString(amount_str)
	if err != nil {
		return err
	}

	amount.Hi = amount_uint128.Hi
	amount.Lo = amount_uint128.Lo

	return nil
}
