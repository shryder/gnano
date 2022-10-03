package database

import "time"

func (backend *JSONBackend) AddNodeIPs(addresses []string) error {
	backend.DataMutex.Lock()
	defer backend.DataMutex.Unlock()

	for _, address := range addresses {
		if _, already_exists := backend.Data.Nodes[address]; !already_exists {
			backend.Data.Nodes[address] = uint(time.Now().Unix())
		}
	}

	return nil
}

func (backend *JSONBackend) GetNodeIPs() (map[string]uint, error) {
	backend.DataMutex.RLock()
	defer backend.DataMutex.RUnlock()

	return backend.Data.Nodes, nil
}
