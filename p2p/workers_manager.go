package p2p

type WorkersManager struct {
	P2PServer *P2P

	ConfirmReq *ConfirmReqWorker
	ConfirmAck *ConfirmAckWorker
}

func NewWorkerManager(srv *P2P) WorkersManager {
	return WorkersManager{
		P2PServer:  srv,
		ConfirmReq: NewConfirmReqWorker(srv),
		ConfirmAck: NewConfirmAckWorker(srv),
	}
}

func (manager *WorkersManager) StartWorkers() {
	go manager.ConfirmReq.Start()
	go manager.ConfirmAck.Start()
}
