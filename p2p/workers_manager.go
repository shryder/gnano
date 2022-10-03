package p2p

type WorkersManager struct {
	P2PServer  *P2P
	ConfirmReq *ConfirmReqWorker
}

func NewWorkerManager(srv *P2P) *WorkersManager {
	return &WorkersManager{
		P2PServer:  srv,
		ConfirmReq: NewConfirmReqWorker(srv),
	}
}

func (manager *WorkersManager) StartWorkers() {
	go manager.ConfirmReq.Start()
}
