package controller

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	//"github.com/glebarez/sqlite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func setup() *Store {
	return &Store{db: DB.Debug().Begin()}
}

func TestMain(m *testing.M) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared&_journal_mode=WAL"), &gorm.Config{
		PrepareStmt: true, Logger: logger.Default.LogMode(logger.Error),
	})

	if err != nil {
		log.Fatal(err)
	}

	err = db.AutoMigrate(&Peer{})
	if err != nil {
		log.Fatal(err)
	}

	// Create mock peer for initial tests
	db.Create(&Peer{
		ID:        1,
		IP:        "1.1.1.1",
		PubKey:    "pubkey",
		EndPoint:  "1.1.1.1:5000",
		Connected: true,
	})

	DB = db
	os.Exit(m.Run())
}

func TestStore_CreatePeer(t *testing.T) {
	s := setup()
	peer := &Peer{
		ID:        1000,
		IP:        "100.65.0.1",
		PubKey:    "testpublickey",
		EndPoint:  "1.1.1.1:5000",
		Connected: true,
		CreatedAt: time.Now(),
	}

	err := s.CreatePeer(peer)
	assert.Nil(t, err, "Error creating Peer in database")
}

func TestStore_GetPeerByID(t *testing.T) {
	s := setup()
	peer, err := s.GetPeerByID(1)
	assert.Nil(t, err, "error lookup peer by ID")
	assert.NotNil(t, peer, "peer is nil")
	assert.Equal(t, uint32(1), peer.ID, "peer ID incorrect")
}

func TestStore_GetPeerByKey(t *testing.T) {
	s := setup()
	peer, err := s.GetPeerByKey("pubkey")
	assert.Nil(t, err, "error lookup peer by key")
	assert.NotNil(t, peer, "peer is nil")
	assert.Equal(t, "pubkey", peer.PubKey)
}

func TestStore_UpdatePeer(t *testing.T) {
	s := setup()
	peer, err := s.GetPeerByID(1)
	peer.ID = 10
	err = s.UpdatePeer(peer)
	assert.Nil(t, err, "error updating peer")
}

func TestStore_GetAllIPs(t *testing.T) {
	s := setup()
	ips, err := s.GetAllIPs()
	assert.Nil(t, err, "error selecting all IP fields")
	assert.NotNil(t, ips, "slice of IPs is nil")
	assert.Len(t, ips, 1)
	assert.Equal(t, ips[0], "1.1.1.1", "ip address incorrect for lookup")
}

func TestStore_UpdatePeerConnected(t *testing.T) {
	s := setup()
	peer, _ := s.GetPeerByID(1)
	assert.True(t, peer.Connected, "test peer connected status should be true")
	peer.Connected = false
	err := s.UpdatePeerConnected(peer)
	assert.Nil(t, err, "error updating peer connected status")
	peer, _ = s.GetPeerByID(1)
	assert.False(t, peer.Connected, "peer connected status not updated correctly")
}

func TestStore_GetAllPeers(t *testing.T) {
	s := setup()
	peers, err := s.GetAllPeers()
	assert.Nil(t, err, "error lookup all peers")
	assert.NotNil(t, peers, "slice of peers nil")
	assert.Len(t, peers, 1)
	assert.Equal(t, peers[0].ID, uint32(1), "peer ID mismatch")
}
