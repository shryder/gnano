package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"os/signal"

	"github.com/Shryder/gnano/node"
	"github.com/Shryder/gnano/types"
	"github.com/naoina/toml"
)

var configFileName = flag.String("config", "./config.toml", "TOML config file path")
var genesisFileName = flag.String("genesis", "./genesis.json", "Genesis block json file path")

func loadConfig() (*node.Config, error) {
	f, err := os.Open(*configFileName)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	var config node.Config
	if err := toml.NewDecoder(f).Decode(&config); err != nil {
		return nil, err
	}

	log.Printf("Loaded config:\n%+v", config)

	return &config, nil
}

func loadGenesisBlock() (*types.Block, error) {
	f, err := os.Open(*genesisFileName)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	var block types.JSONBlock
	err = json.NewDecoder(f).Decode(&block)
	if err != nil {
		return nil, err
	}

	log.Printf("Loaded Genesis:\n%+v", block)

	return block.ToBlock()
}

func main() {
	flag.Parse()

	config, err := loadConfig()
	if err != nil {
		log.Fatal("Error loading config file:", err)
	}

	genesisBlock, err := loadGenesisBlock()
	if err != nil {
		log.Fatal("Error loading genesis file:", err)
	}

	log.Println("Genesis block:", genesisBlock)

	node, err := node.New(config, genesisBlock)
	if err != nil {
		log.Fatal("Error initiation node instance:", err)
	}

	// Create an interruptions handler
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)

	go func() {
		<-c

		log.Println("Cleaning up before shutting down...")
		node.Cleanup()
		log.Println("Done cleaning up.")

		os.Exit(0)
	}()

	node.Start()
}
