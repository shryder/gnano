package database

import (
	"errors"
	"path"
	"strings"

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
