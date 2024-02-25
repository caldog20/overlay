package types

import (
	"net/netip"
	"time"

	apiv1 "github.com/caldog20/overlay/proto/gen/api/v1"
)

type Peer struct {
	ID        uint32 `gorm:"primaryKey,autoIncrement"`
	NodeKey   string
	PublicKey string         `gorm:"unique"`
	IP        netip.Addr     `gorm:"uniqueIndex;serializer:addr;type:string"`
	Endpoint  netip.AddrPort `gorm:"serializer:addrport;type:string"`
	Connected bool
	User      string

	CreatedAt time.Time
	UpdatedAt time.Time
}

//func NewPeer(id uint32, key string, ip string, endpoint string) *Peer {
//	return &Peer{ID: id, PublicKey: key, IP: ip, Endpoint: endpoint, Connected: false}
//}

func (p *Peer) Proto() *apiv1.Peer {
	return &apiv1.Peer{
		Id:        p.ID,
		PublicKey: p.PublicKey,
		Endpoint:  p.Endpoint.String(),
		TunnelIp:  p.IP.String(),
		Connected: p.Connected,
	}
}

func (p *Peer) ProtoConfig() *apiv1.PeerConfig {
	return &apiv1.PeerConfig{
		Id:       p.ID,
		TunnelIp: p.IP.String(),
	}
}

func (p *Peer) Copy() Peer {
	return Peer{
		ID:        p.ID,
		PublicKey: p.PublicKey,
		IP:        p.IP,
		Endpoint:  p.Endpoint,
		Connected: p.Connected,
	}
}
