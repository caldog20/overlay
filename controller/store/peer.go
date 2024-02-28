package store

import (
	"errors"

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

func (s *Store) UpdatePeerEndpoint(id uint32, endpoint string) error {
	return s.db.Model(&types.Peer{}).Where("id = ?", id).Update("endpoint", endpoint).Error
}

func (s *Store) UpdatePeerStatus(id uint32, connected bool) error {
	return s.db.Model(&types.Peer{}).Where("id = ?", id).Update("connected", connected).Error
}

func (s *Store) GetPeerIPs() ([]string, error) {
	var ips []string
	err := s.db.Model(&types.Peer{}).Pluck("ip", &ips).Error
	if err != nil {
		return nil, err
	}
	return ips, nil
}
