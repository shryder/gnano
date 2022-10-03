package types

type Account struct {
	Frontier Block    `json:"frontier"`
	Sideband Sideband `json:"sideband"`
}
