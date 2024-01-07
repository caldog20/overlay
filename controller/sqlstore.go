package controller

import (
	"errors"
	//"database/sql"
	"fmt"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type SqlStore struct {
	db *gorm.DB
}

func NewStore(path string) (Store, error) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?cache=shared&_journal_mode=WAL", path)), &gorm.Config{
		PrepareStmt: true, Logger: logger.Default.LogMode(logger.Info),
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

	err = db.AutoMigrate(&Peer{})

	return &SqlStore{db: db}, nil
}

func (s *SqlStore) GetPeers() ([]Peer, error) {
	var peers []Peer
	result := s.db.Find(&peers)
	if result.Error != nil {
		return nil, result.Error
	}
	return peers, nil
}

func (s *SqlStore) GetPeerByID(id uint32) (*Peer, error) {
	var p Peer
	if err := s.db.First(&p, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, ErrNotFound
	}
	return &p, nil
}

func (s *SqlStore) GetPeerByKey(key string) (*Peer, error) {
	var p Peer
	if err := s.db.Where(&Peer{PublicKey: key}).First(&p).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (s *SqlStore) CreatePeer(peer *Peer) error {
	return s.db.Create(peer).Error
}

func (s *SqlStore) UpdatePeer(peer *Peer) error {
	return s.db.Model(peer).Updates(peer).Error
}

func (s *SqlStore) GetConnectedPeers() ([]Peer, error) {
	var peers []Peer
	result := s.db.Where("connected = ?", true).Find(&peers)
	if result.Error != nil {
		return nil, result.Error
	}
	return peers, nil
}

func (s *SqlStore) UpdatePeerEndpoint(id uint32, endpoint string) error {
	return s.db.Model(&Peer{}).Where("id = ?", id).Update("endpoint", endpoint).Error
}

func (s *SqlStore) UpatePeerStatus(id uint32, connected bool) error {
	return s.db.Model(&Peer{}).Where("id = ?", id).Update("connected", connected).Error
}

func (s *SqlStore) GetPeerIPs() ([]string, error) {
	var ips []string
	err := s.db.Model(&Peer{}).Pluck("ip", &ips).Error
	if err != nil {
		return nil, err
	}
	return ips, nil
}
