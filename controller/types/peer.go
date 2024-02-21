package types

import (
	"net/netip"
	"time"
)

type Peer struct {
	ID        uint32         `gorm:"primaryKey,autoIncrement"`
	NodeKey   string         `gorm:"uniqueIndex"`
	PublicKey string         `gorm:"uniqueIndex,not null"`
	IP        netip.Addr     `gorm:"uniqueIndex;serializer:addr;type:string"`
	Endpoint  netip.AddrPort `gorm:"serializer:addrport;type:string"`
	Connected bool           `gorm:"index"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

//func NewPeer(id uint32, key string, ip string, endpoint string) *Peer {
//	return &Peer{ID: id, PublicKey: key, IP: ip, Endpoint: endpoint, Connected: false}
//}

func (p *Peer) Copy() Peer {
	return Peer{
		ID:        p.ID,
		PublicKey: p.PublicKey,
		IP:        p.IP,
		Endpoint:  p.Endpoint,
		Connected: p.Connected,
	}
}

type PeerConfig struct {
	ID uint32
	IP string
}

//func (p *Peer) GetConfig() *PeerConfig {
//	return &PeerConfig{p.ID, p.IP}
//}
