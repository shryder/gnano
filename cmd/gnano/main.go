package main

import (
	"flag"
	"log"
	"os"

	"github.com/Shryder/gnano/node"
	"github.com/naoina/toml"
)

var configFileName = flag.String("config", "./config.toml", "TOML config file path")

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

func main() {
	flag.Parse()

	config, err := loadConfig()
	if err != nil {
		log.Fatal("Error loading config file:", err)
	}

	node, err := node.New(config)
	if err != nil {
		log.Fatal("Error initiation node instance:", err)
	}

	node.Start()
}
