package database

import (
	"log"

	"github.com/dgraph-io/badger/v3"
)

type BadgerBackend struct {
	Badger *badger.DB
}

func (backend BadgerBackend) BackendName() string {
	return "Badger"
}

func (backend BadgerBackend) AddNodeIP(ip string) error {
	return nil
}

func (backend BadgerBackend) Cleanup() error {
	return backend.Badger.Close()
}

func Initialize(path string) (*BadgerBackend, error) {
	log.Println("Loading Badger backend from", path)

	badger, err := badger.Open(badger.DefaultOptions(path))
	if err != nil {
		return nil, err
	}

	return &BadgerBackend{
		Badger: badger,
	}, nil
}
