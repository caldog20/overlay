package node

import (
	"github.com/caldog20/go-overlay/header"
	"net"
	"sync"
)

type Buffer struct {
	in       []byte
	data     []byte
	h        *header.Header
	raddr    *net.UDPAddr
	size     int
	fwpacket *FWPacket
	peer     *Peer
}

var buffers = sync.Pool{
	New: func() interface{} {
		element := &Buffer{
			in:       make([]byte, 1400),
			data:     make([]byte, 1400),
			h:        new(header.Header),
			raddr:    nil,
			fwpacket: &FWPacket{},
			peer:     nil,
		}
		return element
	},
}

func GetBuffer() *Buffer {
	return buffers.Get().(*Buffer)
}

func PutBuffer(e *Buffer) {
	//buf = buf[:0]
	e.size = 0
	e.peer = nil
	clear(e.in)
	clear(e.data)
	buffers.Put(e)
}

//type Queue struct {
//	items [][]byte
//	mu    sync.Mutex
//	cond  *sync.Cond
//}
//
//func NewQueue() *Queue {
//	q := &Queue{}
//	q.cond = sync.NewCond(&q.mu)
//	return q
//}
//
//func (q *Queue) Push(in []byte) {
//	q.mu.Lock()
//	defer q.mu.Unlock()
//
//	q.items = append(q.items, in)
//	q.cond.Signal()
//}
//
//func (q *Queue) Pop() []byte {
//	q.mu.Lock()
//	defer q.mu.Unlock()
//
//	for len(q.items) == 0 {
//		q.cond.Wait()
//	}
//
//	item := q.items[0]
//	q.items = q.items[1:]
//	return item
//}
//
//func (q *Queue) IsEmpty() bool {
//	return len(q.items) == 0
//}
//
//func (q *Queue) Len() int {
//	return len(q.items)
//}
//
//type Buffer struct {
//	t time.Time
//	b []byte
//}
//
//func NewBuffer() []byte {
//	return make([]byte, 1400)
//}
//
//type BufferAllocator struct {
//	pool       *list.List
//	getBuffer  chan []byte
//	giveBuffer chan []byte
//	counter    atomic.Uint32
//}
//
//func NewBufferAllocator() *BufferAllocator {
//	ba := &BufferAllocator{
//		pool:       new(list.List),
//		getBuffer:  make(chan []byte),
//		giveBuffer: make(chan []byte),
//		counter:    atomic.Uint32{},
//	}
//
//	return ba
//}
//
//func (ba *BufferAllocator) RunBufferAllocator() {
//	for {
//		if ba.pool.Len() == 0 {
//			ba.pool.PushFront(Buffer{t: time.Now(), b: make([]byte, 1400)})
//		}
//
//		e := ba.pool.Front()
//
//		timeout := time.NewTimer(time.Minute * 5)
//
//		select {
//		case b := <-ba.giveBuffer:
//			timeout.Stop()
//			ba.pool.PushFront(Buffer{t: time.Now(), b: b})
//		case ba.getBuffer <- e.Value.(Buffer).b:
//			timeout.Stop()
//			ba.pool.Remove(e)
//		case <-timeout.C:
//			e := ba.pool.Front()
//			for e != nil {
//				n := e.Next()
//				if time.Since(e.Value.(Buffer).t) > time.Minute*5 {
//					ba.pool.Remove(e)
//					e.Value = nil
//				}
//				e = n
//			}
//		}
//	}
//}
//
//func (ba *BufferAllocator) GetBuffer() []byte {
//	return <-ba.getBuffer
//}
//
//func (ba *BufferAllocator) GiveBuffer(b []byte) {
//	ba.giveBuffer <- b
//	return
//}
