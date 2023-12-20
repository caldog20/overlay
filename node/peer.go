package node

import (
	"context"
	"encoding/base64"
	"github.com/caldog20/go-overlay/proto"
	"github.com/flynn/noise"
	"net"
	"net/netip"
	"sync"
	"sync/atomic"
)

type Peer struct {
	mu sync.RWMutex

	Hostname string
	raddr    *net.UDPAddr // Change later to list of endpoints and track active

	node *Node // Pointer back to node for stuff
	Ip   netip.Addr
	Id   uint32

	noise struct {
		hs        *noise.HandshakeState
		rx        *noise.CipherState
		tx        *noise.CipherState
		state     atomic.Uint64 // probably doesn't need to be atomic
		initiator bool
		pubkey    []byte
		txNonce   atomic.Uint64
	}

	inbound    chan *InboundBuffer
	outbound   chan *OutboundBuffer
	pending    chan *OutboundBuffer
	handshakes chan *InboundBuffer

	running atomic.Bool

	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
}

func NewPeer() *Peer {
	peer := new(Peer)

	// channels
	peer.inbound = make(chan *InboundBuffer, 64)
	peer.outbound = make(chan *OutboundBuffer, 64)
	peer.pending = make(chan *OutboundBuffer, 8)
	peer.handshakes = make(chan *InboundBuffer, 3)

	peer.wg = sync.WaitGroup{}

	//peer.ctx, peer.cancel = context.WithCancel(context.Background())
	return peer
}

func (node *Node) AddPeer(peerInfo *proto.Node) (*Peer, error) {
	peer := NewPeer()

	peer.node = node

	// TODO Fix this
	peer.Id = peerInfo.Id
	peer.Ip = netip.MustParseAddr(peerInfo.Ip)
	peer.noise.pubkey, _ = base64.StdEncoding.DecodeString(peerInfo.Key)
	peer.Hostname = peerInfo.Hostname

	peer.raddr, _ = net.ResolveUDPAddr("udp4", peerInfo.Endpoint)

	// TODO Add methods to manipulate map
	node.maps.l.Lock()
	defer node.maps.l.Unlock()
	node.maps.id[peer.Id] = peer
	node.maps.ip[peer.Ip] = peer

	return peer, nil
}

func (peer *Peer) Run(initiator bool) {
	if peer.running.Load() {
		return
	}

	peer.mu.Lock() // Lock the peer state
	peer.ctx, peer.cancel = context.WithCancel(context.Background())

	peer.running.Store(true)
	peer.wg.Add(3)

	peer.mu.Unlock() // Unlock and launch routines

	go peer.Inbound()
	go peer.Outbound()
	go peer.Handshake(initiator)

	// Wait here for goroutines to finish
	peer.wg.Wait()

	peer.mu.Lock()
	defer peer.mu.Unlock()

	// Cleanup peer state and return to idle peer
	peer.flushQueues()
	peer.noise.hs = nil
	peer.noise.rx = nil
	peer.noise.tx = nil
	peer.noise.state.Store(0)
}

func (peer *Peer) InitHandshake(initiator bool) error {
	//if !locked {
	//	peer.mu.Lock()
	//	defer peer.mu.Unlock()
	//}

	peer.noise.initiator = initiator

	var err error
	peer.noise.hs, err = CreateHandshake(initiator, peer.node.noise.keyPair, peer.noise.pubkey)
	if err != nil {
		return err
	}

	return nil
}

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

func (peer *Peer) flushPendingQueue() {
LOOP:
	for {
		select {
		case b, ok := <-peer.pending:
			if !ok {
				break LOOP
			}
			PutOutboundBuffer(b)
		default:
			break LOOP
		}
	}
}

func (peer *Peer) flushQueues() {
	peer.flushHandshakeQueue()
	peer.flushOutboundQueue()
	peer.flushInboundQueue()
}
