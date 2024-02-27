package store

import (
	"sync"
	"sync/atomic"

	"github.com/caldog20/overlay/controller/types"
)

type MapStore struct {
	// primary key is uint32 ID
	m       sync.Map
	counter atomic.Uint32
}

//func NewStore(config *controller.Config) (Store, error) {
//	if config.DbEnabled {
//		return NewSqlStore(config.DbPath)
//	} else {
//		return NewMapStore(), nil
//	}
//}

func NewMapStore() Store {
	return &MapStore{m: sync.Map{}}
}

func (s *MapStore) GetPeers() ([]types.Peer, error) {
	var peers []types.Peer
	s.m.Range(func(k, v interface{}) bool {
		p := v.(*types.Peer)
		peers = append(peers, types.Peer{
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

func (s *MapStore) GetPeerByID(id uint32) (*types.Peer, error) {
	p, ok := s.m.Load(id)
	if !ok {
		return nil, types.ErrNotFound
	}
	peer, ok := p.(*types.Peer)
	if !ok {
		return nil, types.ErrCastingObject
	}

	return peer, nil
}

func (s *MapStore) GetPeerByKey(key string) (*types.Peer, error) {
	var peer *types.Peer

	s.m.Range(func(k, v interface{}) bool {
		p := v.(*types.Peer)
		if p.PublicKey == key {
			peer = p
			return false
		}
		return true
	})

	if peer == nil {
		return nil, types.ErrNotFound
	}
	return peer, nil
}

func (s *MapStore) GetPeerIPs() ([]string, error) {
	var ips []string
	s.m.Range(func(k, v interface{}) bool {
		p := v.(*types.Peer)
		ips = append(ips, p.IP)
		return true
	})

	return ips, nil
}

func (s *MapStore) CreatePeer(peer *types.Peer) error {
	id := s.counter.Add(1)
	peer.ID = id
	_, existing := s.m.LoadOrStore(peer.ID, peer)
	if existing {
		return types.ErrAlreadyExists
	}
	return nil
}

func (s *MapStore) UpdatePeer(peer *types.Peer) error {
	_, existing := s.m.Load(peer.ID)
	if !existing {
		return types.ErrNotFound
	}
	s.m.Store(peer.ID, peer)
	return nil
}

func (s *MapStore) GetConnectedPeers() ([]types.Peer, error) {
	var peers []types.Peer
	s.m.Range(func(k, v interface{}) bool {
		p := v.(*types.Peer)
		peers = append(peers, p.Copy())
		return true
	})
	return peers, nil
}

func (s *MapStore) UpdatePeerEndpoint(id uint32, endpoint string) error {
	p, ok := s.m.Load(id)
	if !ok {
		return types.ErrNotFound
	}
	peer := p.(*types.Peer)
	peer.Endpoint = endpoint
	s.m.Store(id, peer)
	return nil
}

func (s *MapStore) UpdatePeerStatus(id uint32, connected bool) error {
	p, ok := s.m.Load(id)
	if !ok {
		return types.ErrNotFound
	}
	peer := p.(*types.Peer)
	peer.Connected = connected
	s.m.Store(id, peer)
	return nil
}
