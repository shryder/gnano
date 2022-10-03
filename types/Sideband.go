package types

import "math/big"

type Sideband struct {
	Height    *big.Int `json:"sideband"`
	Timestamp uint     `json:"timestamp"`
}
