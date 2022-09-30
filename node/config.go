package node

import (
	"github.com/Shryder/gnano/database"
	"github.com/Shryder/gnano/p2p"
	"github.com/Shryder/gnano/rpc"
)

type IPCConfig struct {
	Path string
}

type TxPoolConfig struct {
	MaxUncheckedCount uint
	UncheckedLifetime uint
}

type Config struct {
	Nano     p2p.Config
	HTTP     rpc.HTTPConfig
	WS       rpc.WSConfig
	IPC      IPCConfig
	TxPool   TxPoolConfig
	Database database.Config
}
