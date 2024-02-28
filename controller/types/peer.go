package types

import (
	"time"

	controllerv1 "github.com/caldog20/overlay/proto/gen/controller/v1"
)

type Peer struct {
	ID        uint32 `gorm:"primaryKey,autoIncrement"`
	PublicKey string `gorm:"uniqueIndex,not null"`
	IP        string `gorm:"uniqueIndex,not null"`
	Endpoint  string
	Connected bool `gorm:"index"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewPeer(id uint32, key string, ip string, endpoint string) *Peer {
	return &Peer{ID: id, PublicKey: key, IP: ip, Endpoint: endpoint, Connected: false}
}

func (p *Peer) Proto() *controllerv1.Peer {
	return &controllerv1.Peer{
		Id:        p.ID,
		PublicKey: p.PublicKey,
		Endpoint:  p.Endpoint,
		TunnelIp:  p.IP,
	}
}

func (p *Peer) ProtoConfig() *controllerv1.PeerConfig {
	return &controllerv1.PeerConfig{Id: p.ID, TunnelIp: p.IP}
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
