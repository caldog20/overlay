package controller

import (
	"errors"
	//"database/sql"
	"fmt"

	"gorm.io/driver/sqlite"
	//"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	//_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	db *gorm.DB
}

func NewStore(path string) (*Store, error) {
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

	return &Store{db: db}, nil
}

func (s *Store) Migrate() error {
	err := s.db.AutoMigrate(&Peer{})
	if err != nil {
		return errors.New("Error during auto migration of database")
	}
	return nil
}

func (s *Store) CreatePeer(peer *Peer) error {
	return s.db.Create(peer).Error
}

func (s *Store) CreateOrUpdatePeer(peer *Peer) error {
	return s.db.Save(peer).Error
}

func (s *Store) GetAllPeers() ([]Peer, error) {
	var peers []Peer
	result := s.db.Find(&peers)
	if result.Error != nil {
		return nil, result.Error
	}
	return peers, nil
}

func (s *Store) GetPeerByID(id uint32) (*Peer, error) {
	var p Peer
	if err := s.db.First(&p, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (s *Store) GetPeerByKey(key string) (*Peer, error) {
	var p Peer
	if err := s.db.Where(&Peer{PubKey: key}).First(&p).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (s *Store) UpdatePeer(peer *Peer) error {
	return s.db.Model(peer).Updates(peer).Error
}

func (s *Store) UpdatePeerConnectedByID(id uint32, connected bool) error {
	return s.db.Model(&Peer{}).Where("id = ?", id).Update("connected", connected).Error
}

func (s *Store) UpdatePeerConnected(peer *Peer) error {
	return s.db.Model(peer).Update("connected", peer.Connected).Error
}

func (s *Store) GetAllIPs() ([]string, error) {
	var IPs []string
	err := s.db.Model(&Peer{}).Pluck("ip", &IPs).Error
	if err != nil {
		return nil, err
	}
	return IPs, nil
}
