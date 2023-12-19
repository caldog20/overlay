package node

import (
	"github.com/flynn/noise"
	"net"
	"net/netip"
	"sync"
)

type Peer struct {
	mu sync.RWMutex
	hs *noise.HandshakeState
	rx *noise.Cipher
	tx *noise.Cipher

	raddr *net.UDPAddr // Change later to list of endpoints and track active

	node *Node // Pointer back to node for stuff
	Ip   netip.Addr
}
