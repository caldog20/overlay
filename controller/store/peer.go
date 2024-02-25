package store

import (
	"errors"
	"net/netip"

	"github.com/caldog20/overlay/controller/types"
	"gorm.io/gorm"
)

func (s *Store) GetPeers() ([]types.Peer, error) {
	var peers []types.Peer
	result := s.db.Find(&peers)
	if result.Error != nil {
		return nil, result.Error
	}
	return peers, nil
}

func (s *Store) GetPeerByID(id uint32) (*types.Peer, error) {
	var p types.Peer
	if err := s.db.First(&p, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, types.ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (s *Store) GetPeerByIP(ip string) (*types.Peer, error) {
	var p types.Peer
	if err := s.db.Where("ip = ?", ip).First(&p).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, types.ErrNotFound
	}
	return &p, nil
}

func (s *Store) GetPeerByKey(key string) (*types.Peer, error) {
	var p types.Peer
	if err := s.db.Where(&types.Peer{PublicKey: key}).First(&p).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, types.ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (s *Store) CreatePeer(peer *types.Peer) error {
	return s.db.Create(peer).Error
}

func (s *Store) UpdatePeer(peer *types.Peer) error {
	return s.db.Model(peer).Updates(peer).Error
}

func (s *Store) GetConnectedPeers() ([]types.Peer, error) {
	var peers []types.Peer
	result := s.db.Where("connected = ?", true).Find(&peers)
	if result.Error != nil {
		return nil, result.Error
	}
	return peers, nil
}

func (s *Store) SetPeerIP(peer *types.Peer, ip netip.Addr) error {
	return s.db.Model(peer).Updates(&types.Peer{IP: ip}).Error
}

func (s *Store) SetPeerPublicKey(peer *types.Peer, key string) error {
	return s.db.Model(peer).Updates(&types.Peer{PublicKey: key}).Error
}

func (s *Store) SetPeerEndpoint(peer *types.Peer, endpoint netip.AddrPort) error {
	return s.db.Model(peer).Updates(&types.Peer{Endpoint: endpoint}).Error
}

func (s *Store) SetPeerStatus(peer *types.Peer, connected bool) error {
	return s.db.Model(peer).Updates(&types.Peer{Connected: connected}).Error
}

func (s *Store) GetAllocatedIPs() ([]string, error) {
	var ips []string
	err := s.db.Model(&types.Peer{}).Pluck("ip", &ips).Error
	if err != nil {
		return nil, err
	}
	return ips, nil
}

func (s *Store) DeletePeer(id uint32) error {
	return s.db.Delete(&types.Peer{}, id).Error
}

func (s *Store) GetPeerByNodeKey(nodeKey string) (*types.Peer, error) {
	var p types.Peer
	if err := s.db.Where(&types.Peer{NodeKey: nodeKey}).First(&p).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, types.ErrNotFound
		} else {
			return nil, err
		}
	}
	return &p, nil
}
