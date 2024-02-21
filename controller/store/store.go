package store

import (
	"github.com/caldog20/overlay/controller/types"
)

type Store interface {
	GetPeers() ([]types.Peer, error)
	GetPeerByID(id uint32) (*types.Peer, error)
	GetPeerByKey(key string) (*types.Peer, error)
	GetPeerByIP(ip string) (*types.Peer, error)
	GetAllocatedIPs() ([]string, error)
	CreatePeer(peer *types.Peer) error
	UpdatePeer(peer *types.Peer) error
	GetConnectedPeers() ([]types.Peer, error)
	UpdatePeerEndpoint(id uint32, endpoint string) error
	UpatePeerStatus(id uint32, connected bool) error
	DeletePeer(id uint32) error
	CreateRegisterKey(key *types.RegisterKey) error
	GetRegisterKeys() ([]types.RegisterKey, error)
	GetRegisterKey(key string) (*types.RegisterKey, error)
}
