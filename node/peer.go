package node

import (
	"fmt"
	"github.com/caldog20/go-overlay/header"
	"log"
	"net"
	"net/netip"
	"sync"
	"sync/atomic"
	"time"

	"github.com/flynn/noise"
)

const (
	HandshakeNotStarted = iota
	HandshakeInitSent
	HandshakeInitRecv
	HandShakeRespSent
	HandshakeRespRecv
	HandshakeDone
)

const (
	NoHandshake    = 0
	DoingHandshake = 1
)

type Peer struct {
	mu       sync.RWMutex
	remoteID uint32
	remote   *net.UDPAddr
	ready    atomic.Bool
	vpnip    netip.Addr
	hs       *noise.HandshakeState
	rx       *noise.CipherState
	tx       *noise.CipherState
	rs       []byte
	state    int

	status atomic.Uint32

	node *Node

	inqueue    chan *Buffer
	outqueue   chan *Buffer
	pending    chan *Buffer
	handshakes chan *Buffer

	timers struct {
		handshake *time.Ticker
		rxtx      *time.Ticker
	}
}

func (p *Peer) Start(initiator bool) {
	if p.ready.Load() {
		return
	}

	if p.status.Load() == 1 {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	//p.timers.rxtx.Reset(time.Second * 10)
	p.status.Store(1)
	err := p.NewHandshake(initiator, p.node.keyPair)
	if err != nil {
		panic(err)
	}

	p.StartHandshake(initiator)
	if initiator {
		elem := <-p.handshakes
		log.Printf("Received handshake resp from peer %s\n", elem.raddr.String())
		if elem.h.SubType == header.Responder {
			_, p.tx, p.rx, err = p.hs.ReadMessage(nil, elem.in[header.Len:elem.size])
			if err != nil {
				p.ResetState(false)
				panic(err)
			}
			*p.remote = *elem.raddr
			p.hs = nil
			p.state = HandshakeDone
			p.timers.rxtx.Reset(time.Second * 10)
			p.timers.handshake.Stop()

			// send outbound pending packets
			p.ready.Store(true)
			p.SendPending()
		}
		PutBuffer(elem)
	}

	if p.ready.Load() {
		go p.RunTimers()
		go p.RunInbound()
		go p.RunOutbound()
		p.status.Store(0)
	}
}

func (p *Peer) SendPending() {
	for {
		if !p.ready.Load() {
			return
		}
		select {

		case elem, ok := <-p.pending:
			if !ok {
				// channel closed, just return
				return
			}
			out, err := elem.h.Encode(elem.data, header.Data, header.None, p.node.localID, p.tx.Nonce())
			out, err = p.tx.Encrypt(out, nil, elem.in[:elem.size])
			if err != nil {
				p.ResetState(true)
				panic(err)
			}
			p.node.conns[0].uc.WriteToUDP(out, p.remote)
			//log.Printf("Sent data to %s - len: %d", p.remote.String(), elem.size)
			PutBuffer(elem)
		default:
			return
		}

	}
}

func (p *Peer) RunInbound() {
	for elem := range p.inqueue {
		if !p.ready.Load() {
			return
		}
		p.rx.SetNonce(elem.h.MsgCounter)
		out, err := p.rx.Decrypt(elem.data[:0], nil, elem.in[header.Len:elem.size])
		if err != nil {
			p.ResetState(true)
			panic(err)
		}
		p.node.tun.Write(out)
		PutBuffer(elem)
		//log.Printf("Recv data from %s - len: %d", elem.raddr.String(), len(out))
		p.mu.RLock()
		if !p.remote.IP.Equal(elem.raddr.IP) {
			log.Printf("Peer Roamed: Updating remote from %s to %s", p.remote.String(), elem.raddr.String())
			p.mu.Lock()
			*p.remote = *elem.raddr
			p.mu.Unlock()
		}
		p.mu.RUnlock()
	}
}

func (p *Peer) RunOutbound() {
	for elem := range p.outqueue {
		if !p.ready.Load() {
			return
		}

		out, err := elem.h.Encode(elem.data, header.Data, header.None, p.node.localID, p.tx.Nonce())
		out, err = p.tx.Encrypt(out, nil, elem.in[:elem.size])
		if err != nil {
			p.ResetState(true)
			panic(err)
		}
		p.node.conns[0].uc.WriteToUDP(out, p.remote)
		//log.Printf("Sent data to %s - len: %d", p.remote.String(), elem.size)
		PutBuffer(elem)
	}
}

func (p *Peer) RunTimers() {
	//for {
	//	p.mu.RLock()
	//	select {
	//	case <-p.timers.handshake.C:
	//		if p.state == HandshakeInitSent {
	//			// Restart handshake
	//			p.StartHandshake(true)
	//		}
	//
	//	}
	//}
}

func (p *Peer) StartHandshake(initiator bool) {
	//p.mu.Lock()
	//defer p.mu.Unlock()
	//err := p.NewHandshake(true, p.node.keyPair)
	//if err != nil {
	//	return
	//}
	if initiator {
		elem := GetBuffer()
		out, _ := elem.h.Encode(elem.data, header.Handshake, header.Initiator, p.node.localID, 0)
		var err error
		out, _, _, err = p.hs.WriteMessage(out, nil)
		if err != nil {
			p.ResetState(false)
			panic(err)
		}

		p.state = HandshakeInitSent
		log.Printf("Sending handshake init to peer %s\n", p.remote.String())
		p.node.conns[0].uc.WriteToUDP(out, p.remote)
		PutBuffer(elem)
		p.timers.handshake.Reset(time.Second * 10)
	}

	if !initiator {
		elem := <-p.handshakes
		//if elem.h.ID != p.node.localID {
		//	p.ResetState(false)
		//	panic("bad ID")
		//}
		log.Printf("Received handshake init from peer %s\n", elem.raddr.String())
		_, _, _, err := p.hs.ReadMessage(nil, elem.in[header.Len:elem.size])
		if err != nil {
			log.Println("Failed to read handshake init")
			p.ResetState(false)
			panic(err)
		}

		p.state = HandshakeInitRecv

		out, _ := elem.h.Encode(elem.data, header.Handshake, header.Responder, p.node.localID, 1)
		out, p.rx, p.tx, err = p.hs.WriteMessage(out, nil)
		if err != nil {
			log.Println("Failed to write handshake response")
			p.ResetState(false)
			panic(err)
		}
		log.Printf("Sending handshake resp to peer %s\n", elem.raddr.String())
		p.node.conns[0].uc.WriteToUDP(out, elem.raddr)
		*p.remote = *elem.raddr
		PutBuffer(elem)

		p.state = HandShakeRespSent
		p.ready.Store(true)

	}

	return
}

func (p *Peer) NewHandshake(initiator bool, keyPair noise.DHKey) error {
	//p.mu.Lock()
	//defer p.mu.Unlock()

	var err error

	if initiator {
		p.hs, err = NewInitiatorHS(keyPair, p.rs)
	} else {
		p.hs, err = NewResponderHS(keyPair)
	}

	if err != nil {
		p.hs = nil
		return err
	}

	return nil
}

func (p *Peer) ResetState(lock bool) {
	if lock {
		p.mu.Lock()
		defer p.mu.Unlock()
	}

	p.ready.Store(false)
	p.state = HandshakeNotStarted
	p.hs = nil
	p.rx = nil
	p.tx = nil

	flushQueues(p.inqueue)
	flushQueues(p.outqueue)
	flushQueues(p.pending)
	flushQueues(p.handshakes)

}

func flushQueues(c chan *Buffer) {
LOOP:
	for {
		select {
		case e, ok := <-c:
			if !ok {
				break LOOP
			}
			PutBuffer(e)
		default:
			break LOOP
		}
	}
}

func (p *Peer) DoEncrypt(out []byte, in []byte) ([]byte, error) {
	//p.mu.Lock()
	//defer p.mu.Unlock()

	var err error
	out, err = p.tx.Encrypt(out, nil, in)
	if err != nil {
		return in, err
	}

	return out, nil
}

func (p *Peer) DoDecrypt(out []byte, in []byte, counter uint64) ([]byte, error) {
	//p.mu.Lock()
	//defer p.mu.Unlock()
	// Set nonce from received header
	p.rx.SetNonce(counter)
	// try to decrypt data
	var err error = nil
	out, err = p.rx.Decrypt(out, nil, in)
	// Need to close and rehandshake here
	if err != nil {
		err = fmt.Errorf("error decrypting packet from peer id: %d - %v\n", p.remoteID, err)
		p.ResetState(false)
	}

	return out, err
}

func (p *Peer) SetRemote(remote *net.UDPAddr) {
	p.mu.Lock()
	defer p.mu.Unlock()
	*p.remote = *remote
}

func (p *Peer) Remote() *net.UDPAddr {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.remote
}

type PeerMap struct {
	mu     sync.RWMutex
	peerIP map[netip.Addr]*Peer
	peerID map[uint32]*Peer
	//pending map[netip.Addr]*Peer
	//indices map[uint32]*Peer
}

func NewPeerMap() *PeerMap {
	pm := &PeerMap{
		peerIP: make(map[netip.Addr]*Peer),
		peerID: make(map[uint32]*Peer),
	}

	return pm
}

//func (p *PeerMap) Contains(vpnip netip.Addr) *Peer {
//	p.mu.RLock()
//	defer p.mu.RUnlock()
//
//	peer, found := p.peers[vpnip]
//	if !found {
//		return nil
//	}
//
//	return peer
//}
//
//func (p *PeerMap) AddPeer(peer *Peer) error {
//	p.mu.Lock()
//	defer p.mu.Unlock()
//	p.peers[peer.vpnip] = peer
//	return nil
//}
//
//func (p *PeerMap) AddPeerWithIndices(peer *Peer) error {
//	p.mu.Lock()
//	defer p.mu.Unlock()
//	p.peers[peer.vpnip] = peer
//	p.indices[peer.remoteID] = peer
//	return nil
//}
//
//func (p *PeerMap) ContainsPending(vpnip netip.Addr) *Peer {
//	p.mu.RLock()
//	defer p.mu.RUnlock()
//	peer := p.pending[vpnip]
//	return peer
//}
//
//func (p *PeerMap) ContainsPendingRemote(ip *net.UDPAddr) *Peer {
//	p.mu.RLock()
//	defer p.mu.RUnlock()
//
//	for _, v := range p.pending {
//		if v.remote.String() == ip.String() {
//			return v
//		}
//	}
//
//	return nil
//}
//
//func (p *PeerMap) AddPeerPending(peer *Peer) error {
//	p.mu.Lock()
//	defer p.mu.Unlock()
//	p.pending[peer.vpnip] = peer
//	return nil
//}
//
//func (p *PeerMap) DeletePendingPeer(peer *Peer) error {
//	p.mu.Lock()
//	defer p.mu.Unlock()
//	delete(p.pending, peer.vpnip)
//	return nil
//}
//
//func (p *PeerMap) ContainsLocalID(li uint32) *Peer {
//	p.mu.RLock()
//	defer p.mu.RUnlock()
//
//	for _, v := range p.peers {
//		if v.localID == li {
//			return v
//		}
//	}
//
//	return nil
//}
//
//func (p *PeerMap) ContainsRemoteID(ri uint32) *Peer {
//	p.mu.RLock()
//	defer p.mu.RUnlock()
//
//	for _, v := range p.peers {
//		if v.remoteID == ri {
//			return v
//		}
//	}
//
//	return nil
//}
