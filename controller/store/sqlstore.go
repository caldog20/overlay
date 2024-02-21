package store

import (
	"errors"
	"fmt"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"

	"github.com/caldog20/overlay/controller/types"
)

type SqlStore struct {
	db *gorm.DB
}

// CreateRegisterKey implements Store.
func (s *SqlStore) CreateRegisterKey(key *types.RegisterKey) error {
	return s.db.Create(key).Error
}

// GetRegisterKey implements Store.
func (s *SqlStore) GetRegisterKey(key string) (*types.RegisterKey, error) {
	var rkey types.RegisterKey
	if err := s.db.Where("key = ?", key).First(&rkey).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, types.ErrNotFound
		}
		return nil, err
	}
	return &rkey, nil
}

// GetRegisterKeys implements Store.
func (s *SqlStore) GetRegisterKeys() ([]types.RegisterKey, error) {
	var keys []types.RegisterKey
	result := s.db.Find(&keys)
	if result.Error != nil {
		return nil, result.Error
	}
	return keys, nil
}

func NewSqlStore(path string) (Store, error) {
	db, err := gorm.Open(
		sqlite.Open(fmt.Sprintf("file:%s?cache=shared&_journal_mode=WAL", path)),
		&gorm.Config{
			PrepareStmt: true, Logger: logger.Default.LogMode(logger.Error),
		},
	)
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

	schema.RegisterSerializer("addr", AddrSerializer{})
	schema.RegisterSerializer("addrport", AddrPortSerializer{})

	err = db.AutoMigrate(&types.Peer{}, &types.RegisterKey{})
	if err != nil {
		return nil, err
	}
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
			return nil, types.ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (s *SqlStore) GetPeerByIP(ip string) (*types.Peer, error) {
	var p types.Peer
	if err := s.db.Where("ip = ?", ip).First(&p).Error; err != nil {
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

func (s *SqlStore) UpatePeerStatus(id uint32, connected bool) error {
	return s.db.Model(&types.Peer{}).Where("id = ?", id).Update("connected", connected).Error
}

func (s *SqlStore) GetAllocatedIPs() ([]string, error) {
	var ips []string
	err := s.db.Model(&types.Peer{}).Pluck("ip", &ips).Error
	if err != nil {
		return nil, err
	}
	return ips, nil
}

func (s *SqlStore) DeletePeer(id uint32) error {
	return s.db.Delete(&types.Peer{}, id).Error
}
