package node

import (
	"log"
)

type OnUDPPacket func(buffer *InboundBuffer, index int)

func (node *Node) ReadUDPPackets(callback OnUDPPacket, index int) {
	for {
		buffer := GetInboundBuffer()
		n, raddr, err := node.conn.ReadFromUDP(buffer.in)
		if err != nil {
			PutInboundBuffer(buffer)
			log.Println(err)
			continue
		}

		buffer.size = n
		buffer.raddr = raddr
		callback(buffer, index)
	}
}
