package packets

import (
	"bytes"
	"encoding/binary"
)

type PacketBody struct {
	Buff bytes.Buffer
}

func (packetBody *PacketBody) WriteBE(data interface{}) {
	binary.Write(&packetBody.Buff, binary.BigEndian, data)
}

func (packetBody *PacketBody) WriteLE(data interface{}) {
	binary.Write(&packetBody.Buff, binary.LittleEndian, data)
}
