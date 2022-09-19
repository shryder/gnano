package rpc

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

type HTTPConfig struct {
	ListenAddr string
	Modules    []string
}

type HTTPRPCServer struct {
	Config *HTTPConfig
	Server *http.Server
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

func (srv *HTTPRPCServer) Handle(w http.ResponseWriter, r *http.Request) {
	r.Header.Add("Content-Type", "application/json")

	if r.Method == http.MethodOptions {
		return
	}

	w.Write([]byte(`{"success":true}`))
}

func (srv *HTTPRPCServer) Start() {
	err := srv.Server.ListenAndServe()

	log.Println("Error serving HTTP Server:", err)
}

func (srv *HTTPRPCServer) ValidateAndStart() error {
	log.Println("Starting HTTP RPC Server")

	go srv.Start()

	return nil
}
