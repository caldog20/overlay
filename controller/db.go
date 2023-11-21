package controller

import (
	"errors"
	"sync"
)

// Mock database for storing user IDs and information

type DB struct {
	mu       sync.RWMutex
	rows     map[string]uint32
	globalID uint32
}

func NewDB() *DB {
	db := &DB{
		mu:       sync.RWMutex{},
		rows:     make(map[string]uint32),
		globalID: 100,
	}

	return db
}

func (db *DB) AddDevice(key string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.rows[key] = db.globalID
	db.globalID += 1

	return nil
}

func (db *DB) GetDeviceByKey(key string) (uint32, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	id, found := db.rows[key]
	if !found {
		return 0, errors.New("device not found in database")
	}

	return id, nil
}

func (db *DB) GetDeviceByID(id uint32) (string, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	for k, v := range db.rows {
		if v == id {
			return k, nil
		}
	}

	return "", errors.New("device not found in database")
}

//func (db *DB) CheckKey(key string, dbkey string) bool {
//	eq := subtle.ConstantTimeCompare([]byte(key), []byte(dbkey))
//	if eq == 0 {
//		return false
//	}
//	return true
//}
