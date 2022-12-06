package types

import "math/big"

type Sideband struct {
	Height    *big.Int `json:"height"`
	Timestamp uint     `json:"timestamp"`
}
