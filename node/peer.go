package node

import (
	"fmt"
	"net"
	"net/netip"
	"sync"
	"sync/atomic"

	"github.com/flynn/noise"
)

const (
	HandshakeNotStarted = iota
	HandshakeInitSent
	HandShakeRespSent
	HandshakeDone
)

type Peer struct {
	mu       sync.RWMutex
	remoteID uint32
	remote   *net.UDPAddr
	raddr    atomic.Pointer[*net.UDPAddr]
	ready    atomic.Bool
	vpnip    netip.Addr
	hs       *noise.HandshakeState
	rx       *noise.CipherState
	tx       *noise.CipherState
	rs       []byte
	state    int
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
