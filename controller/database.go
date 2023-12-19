package controller

import (
	"math/rand"
	"net/netip"
	"sync"
	"sync/atomic"
)

const (
	ErrNodeExists   = Error("Node already exists in database")
	ErrNodeNotFound = Error("Node not found in database")
)

type DB struct {
	l       sync.RWMutex
	id      map[uint32]*Node
	ip      map[netip.Addr]*Node
	key     map[string]*Node
	counter atomic.Uint64
}

func NewDB() *DB {
	db := new(DB)

	db.l = sync.RWMutex{}
	db.id = make(map[uint32]*Node)
	db.ip = make(map[netip.Addr]*Node)
	db.key = make(map[string]*Node)

	return db
}

func (db *DB) GenerateID() uint32 {
	db.l.Lock()
	defer db.l.Unlock()
	for {
		id := rand.Uint32()
		if _, ok := db.id[id]; !ok {
			return id
		}
	}
}

// Adds a new node to the database
// Returns error if node already exists
func (db *DB) AddNode(node *Node) error {
	db.l.Lock()
	defer db.l.Unlock()

	if _, ok := db.id[node.ID]; ok {
		return ErrNodeExists
	}

	db.id[node.ID] = node
	db.ip[node.VpnIP] = node
	db.key[node.NodeKey] = node
	db.counter.Add(1)
	return nil
}

func (db *DB) DeleteNode(node *Node) error {
	db.l.Lock()
	defer db.l.Unlock()

	if _, ok := db.id[node.ID]; ok {
		return ErrNodeNotFound
	}

	delete(db.id, node.ID)
	delete(db.ip, node.VpnIP)
	delete(db.key, node.NodeKey)

	count := db.counter.Load()
	if count > 0 {
		count--
	} else {
		count = 0
	}

	db.counter.Store(count)

	return nil
}

func (db *DB) GetNodeByID(id uint32) (*Node, error) {
	db.l.RLock()
	defer db.l.RUnlock()

	node, ok := db.id[id]

	if !ok {
		return nil, ErrNodeNotFound
	}

	return node, nil
}

func (db *DB) GetNodeByIP(ip netip.Addr) (*Node, error) {
	db.l.RLock()
	defer db.l.RUnlock()

	node, ok := db.ip[ip]

	if !ok {
		return nil, ErrNodeNotFound
	}

	return node, nil
}

func (db *DB) GetNodeByKey(key string) (*Node, error) {
	db.l.RLock()
	defer db.l.RUnlock()

	node, ok := db.key[key]

	if !ok {
		return nil, ErrNodeNotFound
	}

	return node, nil
}
