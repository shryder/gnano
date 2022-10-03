package database

import (
	"bytes"
	"errors"
	"log"
	"os"
	"path"
	"strings"

	"github.com/shryder/ed25519-blake2b"

	json_backend "github.com/Shryder/gnano/database/json"
)

type DatabaseBackend interface {
	BackendName() string

	AddNodeIPs(address []string) error
	GetNodeIPs() (map[string]uint, error)

	Cleanup() error
}

type Database struct {
	Backend DatabaseBackend
	Config  *Config
}

func New(cfg *Config) *Database {
	return &Database{
		Config: cfg,
	}
}

func (db *Database) InitializeBackend() (DatabaseBackend, error) {
	switch strings.ToLower(db.Config.Backend) {
	// case "badger":
	// 	return badger_backend.Initialize(path.Join(db.Config.DataDir, "Badger"))
	case "json":
		return json_backend.Initialize(path.Join(db.Config.DataDir, "JSON", "database.json"))
	}

	return nil, errors.New("Invalid backend provided")
}

func (srv *Database) LoadOrCreateNodeIdentity() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	path := path.Join(srv.Config.DataDir, "node_id.dat")
	log.Println("Loading Node Identity from", path)

	stat, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		node_public_key, node_private_key, err := ed25519.GenerateKey(nil)
		if err != nil {
			log.Println("Error generating node_id key pair")
			return nil, nil, err
		}

		err = os.WriteFile(path, node_public_key, 0644)
		if err != nil {
			log.Println("Error saving node_id.dat to disk")
		}

		return node_public_key, node_private_key, nil
	}

	if err != nil {
		return nil, nil, err
	}

	if stat.IsDir() {
		return nil, nil, errors.New("node_id.dat cannot be a directory")
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}

	publicKey, privateKey, err := ed25519.GenerateKey(bytes.NewReader(contents))
	if err != nil {
		return nil, nil, err
	}

	return publicKey, privateKey, nil
}

func (db *Database) ValidateAndStart() error {
	if len(db.Config.DataDir) == 0 {
		return errors.New("Invalid DataDir provided")
	}

	backend, err := db.InitializeBackend()
	if err != nil {
		return err
	}

	db.Backend = backend

	return nil
}

func (db *Database) Cleanup() error {
	return db.Backend.Cleanup()
}
