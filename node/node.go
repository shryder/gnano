package node

import (
	"log"

	"github.com/Shryder/gnano/database"
	"github.com/Shryder/gnano/p2p"
	"github.com/Shryder/gnano/rpc"
)

type Node struct {
	http     *rpc.HTTPRPCServer
	p2p      *p2p.P2P
	database *database.Database

	StopChannel chan bool
}

func New(cfg *Config) (*Node, error) {
	node := Node{
		http:     rpc.NewHTTPRPCServer(&cfg.HTTP),
		p2p:      p2p.New(&cfg.Nano),
		database: database.New(&cfg.Database),

		StopChannel: make(chan bool),
	}

	return &node, nil
}

func (node *Node) Start() {
	err := node.http.ValidateAndStart()
	if err != nil {
		log.Fatalln("Error starting HTTP server:", err)
	}

	err = node.database.ValidateAndStart()
	if err != nil {
		log.Fatal("Error initializing database:", err)
	}

	err = node.p2p.ValidateAndStart(node.database.Backend)
	if err != nil {
		log.Fatalln("Error starting p2p server:", err)
	}

	<-node.StopChannel
}

func (node *Node) Cleanup() {
	node.StopChannel <- true

	err := node.database.Cleanup()
	if err != nil {
		log.Println("Error closing database:", err)
	}
}
