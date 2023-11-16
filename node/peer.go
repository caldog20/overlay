package node

import (
	noiseimpl "github.com/caldog20/go-overlay/noise"
	"github.com/flynn/noise"
	"math/rand"
	"net"
	"net/netip"
	"sync"
)

var IDMap = make(map[uint32]struct{})

var IDMu sync.Mutex

const (
	HandshakeNotStarted = iota
	HandshakeInitSent
	HandShakeRespSent
	HandshakeDone
)

type Peer struct {
	mu       sync.RWMutex
	localID  uint32
	remoteID uint32
	remote   *net.UDPAddr
	ready    bool
	vpnip    netip.Addr
	hs       *noise.HandshakeState
	rx       *noise.CipherState
	tx       *noise.CipherState
	rs       []byte
	state    int
}

func (p *Peer) NewHandshake(initiator bool, keyPair noise.DHKey) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var hs *noise.HandshakeState
	var err error

	if initiator {
		hs, err = noiseimpl.NewInitiatorHS(keyPair, p.rs)
	} else {
		hs, err = noiseimpl.NewResponderHS(keyPair)
	}

	if err != nil {
		p.hs = nil
		return err
	}

	p.hs = hs

	return nil
}

func GenerateID() uint32 {
	IDMu.Lock()
	defer IDMu.Unlock()
	for {
		try := rand.Uint32()
		_, dupe := IDMap[try]
		if dupe {
			continue
		}
		IDMap[try] = struct{}{}
		return try
	}
}

func (p *Peer) UpdateState(state int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.state = state
}

func (p *Peer) UpdateStatus(status bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.ready = status
}

func (p *Peer) isReady() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.ready
}

func (p *Peer) VpnIP() netip.Addr {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.vpnip
}

func (p *Peer) LocalID() uint32 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.localID
}

func (p *Peer) RemoteID() uint32 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.remoteID
}

func (p *Peer) Remote() *net.UDPAddr {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.remote
}

type PeerMap struct {
	mu      sync.RWMutex
	peers   map[netip.Addr]*Peer
	pending map[netip.Addr]*Peer
	indices map[uint32]*Peer
}

func NewPeerMap() *PeerMap {
	pm := &PeerMap{
		peers:   make(map[netip.Addr]*Peer),
		pending: make(map[netip.Addr]*Peer),
		indices: make(map[uint32]*Peer),
	}

	return pm
}

func (p *PeerMap) Contains(vpnip netip.Addr) *Peer {
	p.mu.RLock()
	defer p.mu.RUnlock()

	peer, found := p.peers[vpnip]
	if !found {
		return nil
	}

	return peer
}

func (p *PeerMap) AddPeer(peer *Peer) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers[peer.vpnip] = peer
	return nil
}

func (p *PeerMap) AddPeerWithIndices(peer *Peer) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers[peer.vpnip] = peer
	p.indices[peer.remoteID] = peer
	return nil
}

func (p *PeerMap) ContainsPending(vpnip netip.Addr) *Peer {
	p.mu.RLock()
	defer p.mu.RUnlock()
	peer := p.pending[vpnip]
	return peer
}

func (p *PeerMap) ContainsPendingRemote(ip *net.UDPAddr) *Peer {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, v := range p.pending {
		if v.remote.String() == ip.String() {
			return v
		}
	}

	return nil
}

func (p *PeerMap) AddPeerPending(peer *Peer) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pending[peer.vpnip] = peer
	return nil
}

func (p *PeerMap) DeletePendingPeer(peer *Peer) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.pending, peer.vpnip)
	return nil
}

func (p *PeerMap) ContainsLocalID(li uint32) *Peer {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, v := range p.peers {
		if v.localID == li {
			return v
		}
	}

	return nil
}

func (p *PeerMap) ContainsRemoteID(ri uint32) *Peer {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, v := range p.peers {
		if v.remoteID == ri {
			return v
		}
	}

	return nil
}
