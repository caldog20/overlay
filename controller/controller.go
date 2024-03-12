package controller

import (
	"fmt"
	"net/netip"
	"slices"
	"sync"

	store "github.com/caldog20/overlay/controller/store"
	"github.com/caldog20/overlay/controller/types"
	controllerv1 "github.com/caldog20/overlay/proto/gen/controller/v1"
)

const (
	Prefix = "100.70.0.0/24"
)

type Controller struct {
	store        *store.Store
	prefix       netip.Prefix
	peerChannels sync.Map

	config *Config
}

func NewController(store *store.Store) *Controller {
	c := &Controller{
		store:        store,
		peerChannels: sync.Map{},
		// TODO: temporary
		prefix: netip.MustParsePrefix(Prefix),
	}

	return c
}

// Provide main service functions agnostic of GRPC/Rest

// LoginPeer logs in an existing peer by public key
// If peer exists, returns peer configuration
// If the peer is not registered or does not exist, returns nil peer and ErrNotFound
func (c *Controller) LoginPeer(publicKey string) (*types.Peer, error) {
	peer, err := c.store.GetPeerByKey(publicKey)
	if err != nil {
		return nil, err
	}

	return peer, nil
}

func (c *Controller) RegisterPeer(publicKey string) error {
	ip, err := c.AllocateIP()
	if err != nil {
		return err
	}
	peer := &types.Peer{IP: ip, PublicKey: publicKey, Connected: false}

	err = c.store.CreatePeer(peer)
	if err != nil {
		return err
	}
	return nil
}

func (c *Controller) SetPeerEndpoint(id uint32, endpoint string) error {
	if id == 0 {
		return types.ErrInvalidPeerID
	}
	err := c.store.UpdatePeerEndpoint(id, endpoint)
	if err != nil {
		return err
	}
	return nil
}

func (c *Controller) InitIPAM(prefix string) error {
	p, err := netip.ParsePrefix(prefix)
	if err != nil {
		return err
	}
	c.prefix = p
	return nil
}

// TODO Write real IPAM functions
func (c *Controller) AllocateIP() (string, error) {
	var nextIP netip.Addr
	ips, err := c.store.GetPeerIPs()
	if err != nil {
		return "", err
	}
	// Skip zero address
	nextIP = c.prefix.Addr().Next()

	var used []netip.Addr

	for _, usedIP := range ips {
		used = append(used, netip.MustParsePrefix(usedIP).Addr())
	}

	// Loop over usedIP slice for each candidate IP to ensure no conflicts
	for {
		if slices.Contains(used, nextIP) {
			nextIP = nextIP.Next()
		} else {
			break
		}
	}

	return fmt.Sprintf("%s/24", nextIP.String()), nil
}

func (c *Controller) GetPeerUpdateChan(id uint32) chan *controllerv1.UpdateResponse {
	pc := make(chan *controllerv1.UpdateResponse, 10)
	c.peerChannels.Store(id, pc)
	return pc
}

func (c *Controller) DeletePeerUpdateChan(id uint32) {
	pc, loaded := c.peerChannels.LoadAndDelete(id)
	if !loaded {
		return
	}
	peerChan := pc.(chan *controllerv1.UpdateResponse)

	close(peerChan)
}

func (c *Controller) ClosePeerUpdateChannels() {
	c.peerChannels.Range(func(k, v interface{}) bool {
		pc := v.(chan *controllerv1.UpdateResponse)
		close(pc)
		return true
	})
}

func (c *Controller) PeerConnected(id uint32) error {
	err := c.store.UpdatePeerStatus(id, true)
	if err != nil {
		return err
	}
	go c.EventPeerConnected(id)
	return nil
}

// TODO: Still notify peers if store update fails???
func (c *Controller) PeerDisconnected(id uint32) error {
	err := c.store.UpdatePeerStatus(id, false)
	if err != nil {
		return err
	}
	c.DeletePeerUpdateChan(id)
	go c.EventPeerDisconnected(id)
	return nil
}

func (c *Controller) GetConnectedPeers() ([]types.Peer, error) {
	peers, err := c.store.GetConnectedPeers()
	if err != nil {
		return nil, err
	}
	return peers, nil
}
