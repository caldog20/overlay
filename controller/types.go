package controller

import (
	"net/netip"
	"sync"
	"time"
)

type Node struct {
	mu       sync.RWMutex
	ID       uint32
	Hostname string
	NodeKey  string
	//PubKey 	 string
	VpnIP    netip.Addr
	EndPoint netip.AddrPort // Change to list of Endpoint() later
	//User string // Make separate type for user details
	timestamp time.Time
}

type Endpoint struct {
	e        netip.AddrPort
	active   bool
	lastSeen time.Time
}

type User struct {
	User  string
	Email string

	Active bool
}

type Error string

func (e Error) Error() string { return string(e) }
