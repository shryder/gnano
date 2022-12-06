package rpc

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/Shryder/gnano/p2p"
	"github.com/Shryder/gnano/types"
	"github.com/gorilla/mux"
)

type HTTPConfig struct {
	ListenAddr string
	Modules    []string
}

type HTTPRPCServer struct {
	Config *HTTPConfig
	Server *http.Server

	P2PServer *p2p.P2P
}

func NewHTTPRPCServer(cfg *HTTPConfig) *HTTPRPCServer {
	server := HTTPRPCServer{
		Config: cfg,
	}

	router := mux.NewRouter()
	router.HandleFunc("/", server.Handle)

	server.Server = &http.Server{
		Handler: router,
		Addr:    cfg.ListenAddr,

		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	return &server
}

func (srv *HTTPRPCServer) HandleMemoryViewer(bodyStr []byte) ([]byte, error) {
	var body struct {
		Params map[string]bool `json:"params"`
	}

	err := json.Unmarshal(bodyStr, &body)
	if err != nil {
		return nil, err
	}

	response := make(map[string]interface{})
	if enabled, found := body.Params["ConfirmedButWaitingForBlockBody"]; enabled && found {
		response["ConfirmedButWaitingForBlockBody"] = srv.P2PServer.PeersManager.GetLivePeersCount()
	}

	if enabled, found := body.Params["UncheckedBlocksManager"]; enabled && found {
		srv.P2PServer.UncheckedBlocksManager.UncheckedBlocksMutex.RLock()

		tableSize := uint(len(srv.P2PServer.UncheckedBlocksManager.UncheckedBlocks))
		table := make([]*types.Block, 0)
		for _, block := range srv.P2PServer.UncheckedBlocksManager.UncheckedBlocks {
			table = append(table, block)
		}

		log.Println("Table size:", len(table), tableSize)
		response["UncheckedBlocksManager"] = struct {
			Count  uint
			Blocks []*types.Block
		}{
			Count:  tableSize,
			Blocks: table,
		}

		srv.P2PServer.UncheckedBlocksManager.UncheckedBlocksMutex.RUnlock()
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}

	return responseJSON, nil
}

func (srv *HTTPRPCServer) HandlePeersInfo(bodyStr []byte) ([]byte, error) {
	response := make(map[string]interface{})
	response["LivePeersCount"] = srv.P2PServer.PeersManager.GetLivePeersCount()
	response["BootstrapPeersCount"] = srv.P2PServer.PeersManager.GetBootstrapPeersCount()

	response["LivePeers"] = srv.P2PServer.PeersManager.LivePeers
	response["BootstrapPeers"] = srv.P2PServer.PeersManager.BootstrapPeers

	responseJSON, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}

	return responseJSON, nil
}

func (srv *HTTPRPCServer) Handle(w http.ResponseWriter, r *http.Request) {
	r.Header.Add("Content-Type", "application/json")

	if r.Method == http.MethodOptions {
		return
	}

	bodyStr, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	var reqBody struct{ Method string }
	err = json.Unmarshal(bodyStr, &reqBody)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	var response []byte

	switch reqBody.Method {
	case "cemented_block_count":
		response = []byte(fmt.Sprintf("%d", srv.P2PServer.Database.Backend.GetBlockCount()))
	case "gnano_memoryViewer":
		response, err = srv.HandleMemoryViewer(bodyStr)
	case "gnano_peersInfo":
		response, err = srv.HandlePeersInfo(bodyStr)
	default:
		err = fmt.Errorf("method %s is not supported", reqBody.Method)
	}

	if err != nil {
		errorJSON, _ := json.Marshal(struct {
			Success bool   `json:"success"`
			Error   string `json:"error"`
		}{true, err.Error()})

		w.Write(errorJSON)
	} else {
		w.Write(response)
	}
}

func (srv *HTTPRPCServer) WriteSuccess(data struct{}) {

}

func (srv *HTTPRPCServer) Start() {
	err := srv.Server.ListenAndServe()

	log.Println("Error serving HTTP Server:", err)
}

func (srv *HTTPRPCServer) ValidateAndStart(p2p *p2p.P2P) error {
	srv.P2PServer = p2p

	log.Println("Starting HTTP RPC Server")

	go srv.Start()

	return nil
}
