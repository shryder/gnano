package database

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Shryder/gnano/types"
)

type DBAccount struct {
	Frontier types.Hash     `json:"hash"`
	Sideband types.Sideband `json:"sideband"`
}
type DBSchema struct {
	Nodes    map[string]uint        `json:"nodes"`    // ip => discovery_timestamp
	Blocks   map[string]types.Block `json:"blocks"`   // hash => block
	Accounts map[string]DBAccount   `json:"accounts"` // public_key => account
}

type JSONBackend struct {
	FilePath string

	Data      DBSchema
	DataMutex sync.RWMutex

	Closed bool
}

func (backend *JSONBackend) BackendName() string {
	return "JSON"
}

func (backend *JSONBackend) Cleanup() error {
	backend.Closed = true

	return nil
}

func Initialize(path string) (*JSONBackend, error) {
	log.Println("Loading JSON backend from", path)

	var data DBSchema
	stat, err := os.Stat(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}

		err = os.MkdirAll(filepath.Dir(path), 0700)
		if err != nil {
			return nil, err
		}

		file, err := os.Create(path)
		if err != nil {
			return nil, err
		}

		// Fill with default empty values
		data = DBSchema{
			Nodes: make(map[string]uint),
		}

		defaultData, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}

		// Write empty object
		_, err = file.Write(defaultData)
		if err != nil {
			return nil, err
		}
	} else {
		if stat.IsDir() {
			return nil, errors.New("database.json cannot be a folder")
		}

		contents, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(contents, &data)
		if err != nil {
			return nil, err
		}
	}

	backend := &JSONBackend{
		FilePath: path,
		Data:     data,
	}

	go backend.PeriodicSaves()

	return backend, nil
}

func (backend *JSONBackend) PeriodicSaves() {
	for {
		jsonified, err := json.Marshal(backend.Data)
		if err != nil {
			log.Println("Error marshalling JSON data:", err)
			continue
		}

		err = os.WriteFile(backend.FilePath, jsonified, os.ModeAppend)
		if err != nil {
			log.Println("Error writing JSON database to file:", err)
			continue
		}

		time.Sleep(time.Second * 5)
	}
}
