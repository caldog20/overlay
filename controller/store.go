package controller

import (
	"sync"

	"gorm.io/gorm"
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

type IPAllocation struct {
	gorm.Model
	IP        string
	Allocated bool
}

type MapStore struct {
	// primary key is uint32 ID
	m sync.Map
}

func (s *MapStore) GetPeerIPs() ([]string, error) {
	//TODO implement me
	panic("implement me")
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

func (s *MapStore) CreatePeer(peer *Peer) error {
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
	//TODO implement me
	panic("implement me")
}

func (s *MapStore) UpdatePeerEndpoint(id uint32, endpoint string) error {
	//TODO implement me
	panic("implement me")
}

func (s *MapStore) UpatePeerStatus(id uint32, connected bool) error {
	//TODO implement me
	panic("implement me")
}
