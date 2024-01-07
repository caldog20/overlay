package node

import (
	"context"
	"errors"
	"log"
	"net"
	"net/netip"
	"sync"
	"sync/atomic"
	"time"

	"github.com/caldog20/overlay/proto"
	"github.com/flynn/noise"
)

const (
	// Timers
	TimerHandshakeTimeout = time.Second * 5
	TimerRxTimeout        = time.Second * 15
	TimerKeepalive        = time.Second * 10
	// Counts
	CountHandshakeRetries = 10
)

type Peer struct {
	mu          sync.RWMutex
	pendingLock sync.RWMutex
	Hostname    string
	raddr       *net.UDPAddr // Change later to list of endpoints and track active

	node *Node // Pointer back to node for stuff
	IP   netip.Addr
	ID   uint32

	noise struct {
		hs        *noise.HandshakeState
		rx        *noise.CipherState
		tx        *noise.CipherState
		state     atomic.Uint64 // probably doesn't need to be atomic, noise struct needs it's own lock
		initiator bool
		pubkey    []byte
		txNonce   atomic.Uint64
	}

	timers struct {
		handshakeSent  *time.Timer
		receivedPacket *time.Timer
		keepalive      *time.Timer
		// sentPacket *time.Timer
	}

	counters struct {
		handshakeRetries atomic.Uint64
	}

	inbound    chan *InboundBuffer
	outbound   chan *OutboundBuffer
	handshakes chan *InboundBuffer

	running     atomic.Bool
	inTransport atomic.Bool

	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
}

func NewPeer() *Peer {
	peer := new(Peer)

	// channels
	peer.inbound = make(chan *InboundBuffer, 16)   // buffered number???
	peer.outbound = make(chan *OutboundBuffer, 64) // allow up to 64 packets to be cached/pending handshake???
	peer.handshakes = make(chan *InboundBuffer, 3) // Handshake packet buffering???

	peer.timers.handshakeSent = time.AfterFunc(TimerHandshakeTimeout, peer.HandshakeTimeout)
	peer.timers.handshakeSent.Stop()

	peer.timers.receivedPacket = time.AfterFunc(TimerRxTimeout, peer.RXTimeout)
	peer.timers.receivedPacket.Stop()

	peer.timers.keepalive = time.AfterFunc(TimerKeepalive, peer.TXTimeout)
	peer.timers.keepalive.Stop()
	//
	//peer.timers.sentPacket = time.NewTimer(TimerKeepalive)
	//peer.timers.sentPacket.Stop()

	peer.wg = sync.WaitGroup{}

	// peer.ctx, peer.cancel = context.WithCancel(context.Background())
	return peer
}

// TODO Proper error text for context around the issue
func (node *Node) AddPeer(peerInfo *proto.RemotePeer) (*Peer, error) {
	peer := NewPeer()

	peer.mu.Lock()
	defer peer.mu.Unlock()

	peer.node = node

	var err error

	// TODO Fix this
	peer.ID = peerInfo.Id
	peer.IP, err = ParseAddr(peerInfo.TunnelIp)
	if err != nil {
		return nil, err
	}

	peer.noise.pubkey, err = DecodeBase64Key(peerInfo.PublicKey)
	if err != nil {
		return nil, err
	}

	//peer.Hostname = peerInfo.Hostname

	peer.raddr, err = net.ResolveUDPAddr("udp4", peerInfo.Endpoint)
	if err != nil {
		return nil, err
	}

	// TODO Add methods to manipulate map
	node.maps.l.Lock()
	defer node.maps.l.Unlock()
	node.maps.id[peer.ID] = peer
	node.maps.ip[peer.IP] = peer

	return peer, nil
}

func (peer *Peer) Start() error {
	peer.mu.Lock()
	defer peer.mu.Unlock()

	// Peer is already running
	if peer.running.Load() {
		return errors.New("peer already running")
	}

	// Lock here when starting peer so routines have to wait for handshake before trying to read data from channels
	peer.pendingLock.Lock()

	peer.wg.Add(3)
	go peer.Inbound()
	go peer.Outbound()
	go peer.Handshake()

	peer.running.Store(true)
	peer.inTransport.Store(false)
	return nil
}

func (peer *Peer) Stop() {
	peer.ResetState()

	// send nil value to kill goroutines
	peer.mu.Lock()
	defer peer.mu.Unlock()
	peer.handshakes <- nil
	peer.inbound <- nil
	peer.outbound <- nil

	// Wait until all routines are finished
	peer.wg.Wait()
	log.Printf("peer %d goroutines have stopped", peer.ID)
	peer.running.Store(false)
}

//func (peer *Peer) Run(initiator bool) {
//	if peer.running.Load() {
//		return
//	}
//
//	peer.mu.Lock() // Lock the peer state
//	// peer.ctx, peer.cancel = context.WithCancel(context.Background())
//
//	peer.running.Store(true)
//	peer.wg.Add(3)
//
//	peer.mu.Unlock() // Unlock and launch routines
//
//	// go peer.Handshake(initiator)
//	go peer.Inbound()
//	go peer.Outbound()
//
//	// Wait here for goroutines to finish
//	peer.wg.Wait()
//
//	peer.mu.Lock()
//	defer peer.mu.Unlock()
//
//	// Cleanup peer state and return to idle peer
//	peer.ResetState()
//	log.Println("Shutting peer down")
//}

func (peer *Peer) InboundPacket(buffer *InboundBuffer) {
	if !peer.running.Load() {
		PutInboundBuffer(buffer)
		return
	}

	// peer.timers.receivedPacket.Stop()

	select {
	case peer.inbound <- buffer:
	default:
		log.Printf("peer id %d: inbound channel full", peer.ID)
	}
}

func (peer *Peer) OutboundPacket(buffer *OutboundBuffer) {
	if !peer.running.Load() {
		PutOutboundBuffer(buffer)
		return
	}

	// For tracking full channels
	select {
	case peer.outbound <- buffer:
	default:
		log.Printf("peer id %d: outbound channel full", peer.ID)
	}

	if !peer.inTransport.Load() && peer.noise.state.Load() == 0 {
		peer.TrySendHandshake(false)
	}
}

func (peer *Peer) ResetState() {
	peer.running.Store(false)

	peer.mu.Lock()
	defer peer.mu.Unlock()

	peer.counters.handshakeRetries.Store(0)
	peer.timers.receivedPacket.Stop()
	peer.timers.keepalive.Stop()

	peer.flushQueues()
	peer.noise.hs = nil
	peer.noise.rx = nil
	peer.noise.tx = nil
	peer.noise.initiator = false
	peer.noise.state.Store(0)
	peer.inTransport.Store(false)
	peer.running.Store(true)
}

func (peer *Peer) InitHandshake(initiator bool) error {
	// Lock here incase something is changing with the nodes keys
	peer.node.noise.l.RLock()
	defer peer.node.noise.l.RUnlock()

	peer.noise.initiator = initiator

	var err error
	peer.noise.hs, err = CreateHandshake(initiator, peer.node.noise.keyPair, peer.noise.pubkey)
	if err != nil {
		return err
	}

	return nil
}

// TODO Fix these in the case channel is never closed
func (peer *Peer) flushInboundQueue() {
LOOP:
	for {
		select {
		case b, ok := <-peer.inbound:
			if !ok {
				break LOOP
			}
			PutInboundBuffer(b)
		default:
			break LOOP
		}
	}
}

func (peer *Peer) flushOutboundQueue() {
LOOP:
	for {
		select {
		case b, ok := <-peer.outbound:
			if !ok {
				break LOOP
			}
			PutOutboundBuffer(b)
		default:
			break LOOP
		}
	}
}

func (peer *Peer) flushHandshakeQueue() {
LOOP:
	for {
		select {
		case b, ok := <-peer.handshakes:
			if !ok {
				break LOOP
			}
			PutInboundBuffer(b)
		default:
			break LOOP
		}
	}
}

//func (peer *Peer) flushPendingQueue() {
//LOOP:
//	for {
//		select {
//		case b, ok := <-peer.pending:
//			if !ok {
//				break LOOP
//			}
//			PutOutboundBuffer(b)
//		default:
//			break LOOP
//		}
//	}
//}

func (peer *Peer) flushQueues() {
	peer.flushHandshakeQueue()
	peer.flushOutboundQueue()
	peer.flushInboundQueue()
}
