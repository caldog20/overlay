package store

import (
	"github.com/caldog20/overlay/controller/types"
)

type Store interface {
	GetPeers() ([]types.Peer, error)
	GetPeerByID(id uint32) (*types.Peer, error)
	GetPeerByKey(key string) (*types.Peer, error)
	GetPeerIPs() ([]string, error)
	CreatePeer(peer *types.Peer) error
	UpdatePeer(peer *types.Peer) error
	GetConnectedPeers() ([]types.Peer, error)
	UpdatePeerEndpoint(id uint32, endpoint string) error
	UpdatePeerStatus(id uint32, connected bool) error
}
