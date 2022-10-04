package database

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
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
	Nodes        map[string]uint         `json:"nodes"`    // ip => discovery_timestamp
	Blocks       map[string]types.Block  `json:"blocks"`   // hash => block
	Accounts     map[string]DBAccount    `json:"accounts"` // public_key => account
	VotingWeight map[string]types.Amount `json:"weights"`  // public_key => weight
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

func loadInitialWeights() (map[string]types.Amount, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	path := path.Join(cwd, "weights.json")

	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	initialWeightsRaw := make(map[string]types.Amount)
	err = json.Unmarshal(contents, &initialWeightsRaw)
	if err != nil {
		return nil, err
	}

	return initialWeightsRaw, nil
}

func loadOrCreateLedgerDB(path string) (*DBSchema, error) {
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
			Nodes:        make(map[string]uint),
			Blocks:       make(map[string]types.Block),
			Accounts:     make(map[string]DBAccount),
			VotingWeight: make(map[string]types.Amount),
		}

		initialWeights, err := loadInitialWeights()
		if err != nil {
			return nil, fmt.Errorf("Error loading initial weights: %w", err)
		}

		data.VotingWeight = initialWeights

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

	return &data, nil
}

func Initialize(path string) (*JSONBackend, error) {
	log.Println("Loading JSON backend from", path)

	data, err := loadOrCreateLedgerDB(path)
	if err != nil {
		return nil, err
	}

	backend := &JSONBackend{
		FilePath: path,
		Data:     *data,
	}

	go backend.PeriodicSaves()

	return backend, nil
}

func (backend *JSONBackend) PeriodicSaves() {
	for {
		if backend.Closed {
			return
		}

		time.Sleep(time.Second * 5)

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
	}
}
