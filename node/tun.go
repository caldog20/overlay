package node

import (
	"log"
)

type OnTunnelPacket func(buffer *OutboundBuffer)

func (node *Node) ReadPackets(callback OnTunnelPacket) {
	for {
		buffer := GetOutboundBuffer()
		n, err := node.tun.Read(buffer.packet)
		if err != nil {
			PutOutboundBuffer(buffer)
			log.Println(err)
			continue
		}

		buffer.size = n
		callback(buffer)
	}
}
