package node

import (
	"log"

	"github.com/Shryder/gnano/ledger"
	"github.com/Shryder/gnano/p2p"
	"github.com/Shryder/gnano/rpc"
)

type Node struct {
	http   *rpc.HTTPRPCServer
	p2p    *p2p.P2P
	ledger *ledger.Ledger
}

func New(cfg *Config) (*Node, error) {
	node := Node{
		http: rpc.NewHTTPRPCServer(&cfg.HTTP),
		p2p:  p2p.New(&cfg.Nano),
	}

	return &node, nil
}

func (node *Node) Start() {
	err := node.p2p.ValidateAndStart()
	if err != nil {
		log.Fatalln("Error starting p2p server:", err)
	}

	err = node.http.ValidateAndStart()
	if err != nil {
		log.Fatalln("Error starting HTTP server:", err)
	}

	select {}
}
