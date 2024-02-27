package store

import (
	"errors"
	"fmt"

	"github.com/caldog20/overlay/controller/types"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type SqlStore struct {
	db *gorm.DB
}

func NewSqlStore(path string) (Store, error) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?cache=shared&_journal_mode=WAL", path)), &gorm.Config{
		PrepareStmt: true, Logger: logger.Default.LogMode(logger.Error),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxIdleConns(3)

	err = sqlDB.Ping()
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(&types.Peer{})

	return &SqlStore{db: db}, nil
}

func (s *SqlStore) GetPeers() ([]types.Peer, error) {
	var peers []types.Peer
	result := s.db.Find(&peers)
	if result.Error != nil {
		return nil, result.Error
	}
	return peers, nil
}

func (s *SqlStore) GetPeerByID(id uint32) (*types.Peer, error) {
	var p types.Peer
	if err := s.db.First(&p, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, types.ErrNotFound
	}
	return &p, nil
}

func (s *SqlStore) GetPeerByKey(key string) (*types.Peer, error) {
	var p types.Peer
	if err := s.db.Where(&types.Peer{PublicKey: key}).First(&p).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, types.ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (s *SqlStore) CreatePeer(peer *types.Peer) error {
	return s.db.Create(peer).Error
}

func (s *SqlStore) UpdatePeer(peer *types.Peer) error {
	return s.db.Model(peer).Updates(peer).Error
}

func (s *SqlStore) GetConnectedPeers() ([]types.Peer, error) {
	var peers []types.Peer
	result := s.db.Where("connected = ?", true).Find(&peers)
	if result.Error != nil {
		return nil, result.Error
	}
	return peers, nil
}

func (s *SqlStore) UpdatePeerEndpoint(id uint32, endpoint string) error {
	return s.db.Model(&types.Peer{}).Where("id = ?", id).Update("endpoint", endpoint).Error
}

func (s *SqlStore) UpdatePeerStatus(id uint32, connected bool) error {
	return s.db.Model(&types.Peer{}).Where("id = ?", id).Update("connected", connected).Error
}

func (s *SqlStore) GetPeerIPs() ([]string, error) {
	var ips []string
	err := s.db.Model(&types.Peer{}).Pluck("ip", &ips).Error
	if err != nil {
		return nil, err
	}
	return ips, nil
}
