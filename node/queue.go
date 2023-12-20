package node

import (
	"log"
	"net"
	"sync"
	"sync/atomic"
)

const BufferSize = 1500

type InboundBuffer struct {
	in     []byte       // Raw data from UDP Socket
	packet []byte       // Allocated for decrypted data
	raddr  *net.UDPAddr // Remote Address of packet
	size   int          // size of data read from UDP socket
	header *Header
	peer   *Peer // Peer this index belongs to
}

type OutboundBuffer struct {
	out    []byte // Final data to send over socket
	packet []byte // For tunnel inbound data
	size   int
	header *Header
	peer   *Peer
}

var (
	InboundBuffers  = sync.Pool{New: NewInboundBuffer}
	OutboundBuffers = sync.Pool{New: NewOutboundBuffer}
	BufferSwaps     atomic.Uint64
)

func NewInboundBuffer() interface{} {
	buffer := new(InboundBuffer)
	buffer.in = make([]byte, BufferSize)
	buffer.packet = make([]byte, BufferSize)
	buffer.raddr = nil
	buffer.size = 0
	buffer.header = NewHeader()
	buffer.peer = nil
	log.Println("NewInboundBuffer")
	return buffer
}

func GetInboundBuffer() *InboundBuffer {
	BufferSwaps.Add(1)
	return InboundBuffers.Get().(*InboundBuffer)
}

func PutInboundBuffer(buffer *InboundBuffer) {
	clear(buffer.in)
	clear(buffer.packet)
	buffer.raddr = nil
	buffer.size = 0
	buffer.header.Reset()
	buffer.peer = nil

	InboundBuffers.Put(buffer)

	n := BufferSwaps.Load()
	BufferSwaps.Store(n - 1)
}

func NewOutboundBuffer() interface{} {
	buffer := new(OutboundBuffer)
	buffer.out = make([]byte, BufferSize)
	buffer.packet = make([]byte, BufferSize)
	buffer.size = 0
	buffer.header = NewHeader()
	buffer.peer = nil
	log.Println("NewOutboundBuffer")
	return buffer
}

func GetOutboundBuffer() *OutboundBuffer {
	BufferSwaps.Add(1)
	return OutboundBuffers.Get().(*OutboundBuffer)
}

func PutOutboundBuffer(buffer *OutboundBuffer) {
	clear(buffer.out)
	clear(buffer.packet)
	buffer.size = 0
	buffer.peer = nil
	buffer.header.Reset()

	OutboundBuffers.Put(buffer)
	n := BufferSwaps.Load()
	BufferSwaps.Store(n - 1)

}
