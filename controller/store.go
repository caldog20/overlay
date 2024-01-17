package controller

import (
	"sync"
	"sync/atomic"
)

type Store interface {
	GetPeers() ([]Peer, error)
	GetPeerByID(id uint32) (*Peer, error)
	GetPeerByKey(key string) (*Peer, error)
	GetPeerIPs() ([]string, error)
	CreatePeer(peer *Peer) error
	UpdatePeer(peer *Peer) error
	GetConnectedPeers() ([]Peer, error)
	UpdatePeerEndpoint(id uint32, endpoint string) error
	UpatePeerStatus(id uint32, connected bool) error
}

type MapStore struct {
	// primary key is uint32 ID
	m       sync.Map
	counter atomic.Uint32
}

func NewStore(config *Config) (Store, error) {
	if config.DbEnabled {
		return NewSqlStore(config.DbPath)
	} else {
		return NewMapStore(), nil
	}
}

func NewMapStore() Store {
	return &MapStore{m: sync.Map{}}
}

func (s *MapStore) GetPeers() ([]Peer, error) {
	var peers []Peer
	s.m.Range(func(k, v interface{}) bool {
		p := v.(*Peer)
		peers = append(peers, Peer{
			ID:        p.ID,
			IP:        p.IP,
			PublicKey: p.PublicKey,
			Endpoint:  p.Endpoint,
			Connected: p.Connected,
			CreatedAt: p.CreatedAt,
			UpdatedAt: p.UpdatedAt,
		})
		return true
	})

	return peers, nil
}

func (s *MapStore) GetPeerByID(id uint32) (*Peer, error) {
	p, ok := s.m.Load(id)
	if !ok {
		return nil, ErrNotFound
	}
	peer, ok := p.(*Peer)
	if !ok {
		return nil, ErrCastingObject
	}

	return peer, nil
}

func (s *MapStore) GetPeerByKey(key string) (*Peer, error) {
	var peer *Peer

	s.m.Range(func(k, v interface{}) bool {
		p := v.(*Peer)
		if p.PublicKey == key {
			peer = p
			return false
		}
		return true
	})

	if peer == nil {
		return nil, ErrNotFound
	}
	return peer, nil
}

func (s *MapStore) GetPeerIPs() ([]string, error) {
	var ips []string
	s.m.Range(func(k, v interface{}) bool {
		p := v.(*Peer)
		ips = append(ips, p.IP)
		return true
	})

	return ips, nil
}

func (s *MapStore) CreatePeer(peer *Peer) error {
	id := s.counter.Add(1)
	peer.ID = id
	_, existing := s.m.LoadOrStore(peer.ID, peer)
	if existing {
		return ErrAlreadyExists
	}
	return nil
}

func (s *MapStore) UpdatePeer(peer *Peer) error {
	_, existing := s.m.Load(peer.ID)
	if !existing {
		return ErrNotFound
	}
	s.m.Store(peer.ID, peer)
	return nil
}

func (s *MapStore) GetConnectedPeers() ([]Peer, error) {
	var peers []Peer
	s.m.Range(func(k, v interface{}) bool {
		p := v.(*Peer)
		peers = append(peers, p.Copy())
		return true
	})
	return peers, nil
}

func (s *MapStore) UpdatePeerEndpoint(id uint32, endpoint string) error {
	p, ok := s.m.Load(id)
	if !ok {
		return ErrNotFound
	}
	peer := p.(*Peer)
	peer.Endpoint = endpoint
	s.m.Store(id, peer)
	return nil
}

func (s *MapStore) UpatePeerStatus(id uint32, connected bool) error {
	p, ok := s.m.Load(id)
	if !ok {
		return ErrNotFound
	}
	peer := p.(*Peer)
	peer.Connected = connected
	s.m.Store(id, peer)
	return nil
}
