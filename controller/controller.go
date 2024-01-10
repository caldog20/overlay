package controller

import (
	"context"
	"fmt"
	"net/netip"
	"sync"

	"github.com/caldog20/overlay/proto"
	"golang.org/x/sync/errgroup"
)

const (
	Prefix = "100.70.0.0/24"
)

type Controller struct {
	store        Store
	grpcServer   *GRPCServer
	prefix       netip.Prefix
	peerChannels sync.Map

	confg *Config
}

func NewController(config *Config, store Store) *Controller {
	c := new(Controller)
	c.InitIPAM(Prefix)
	c.confg = config

	gserver := NewGRPCServer(config, c)

	c.store = store
	c.grpcServer = gserver
	c.peerChannels = sync.Map{}
	return c
}

func (c *Controller) Run(ctx context.Context) error {
	eg, egCtx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		return c.grpcServer.Run()
	})

	eg.Go(func() error {
		<-egCtx.Done()
		c.ClosePeerUpdateChannels()
		c.grpcServer.server.GracefulStop()
		return nil
	})

	if err := eg.Wait(); err != nil {
		return err
	}
	return nil
}

// Provide main service functions agnostic of GRPC/Rest

// LoginPeer logs in an existing peer by public key
// If peer exists, returns peer configuration
// If the peer is not registered or does not exist, returns nil peer and ErrNotFound
func (c *Controller) LoginPeer(publicKey string) (*PeerConfig, error) {
	peer, err := c.store.GetPeerByKey(publicKey)
	if err != nil {
		return nil, err
	}

	config := peer.GetPeerConfig()

	return config, nil
}

func (c *Controller) RegisterPeer(publicKey string) error {
	ip, err := c.AllocateIP()
	if err != nil {
		return err
	}
	peer := &Peer{IP: ip, PublicKey: publicKey, Connected: false}

	err = c.store.CreatePeer(peer)
	if err != nil {
		return err
	}
	return nil
}

func (c *Controller) SetPeerEndpoint(id uint32, endpoint string) error {
	if id == 0 {
		return ErrInvalidPeerID
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

// TODO Write real IPAM
func (c *Controller) AllocateIP() (string, error) {
	var nextIP netip.Addr
	ips, err := c.store.GetPeerIPs()
	if err != nil {
		return "", err
	}
	nextIP = c.prefix.Addr().Next()

TOP:
	for _, ip := range ips {
		p := netip.MustParsePrefix(ip)
		if p.Addr() == nextIP {
			nextIP = nextIP.Next()
			goto TOP
		}
	}

	return fmt.Sprintf("%s/24", nextIP.String()), nil
}

func (c *Controller) GetPeerUpdateChan(id uint32) chan *proto.UpdateResponse {
	pc := make(chan *proto.UpdateResponse, 10)
	c.peerChannels.Store(id, pc)
	return pc
}

func (c *Controller) DeletePeerUpdateChan(id uint32) {
	pc, loaded := c.peerChannels.LoadAndDelete(id)
	if !loaded {
		return
	}
	peerChan := pc.(chan *proto.UpdateResponse)

	close(peerChan)
}

func (c *Controller) ClosePeerUpdateChannels() {
	c.peerChannels.Range(func(k, v interface{}) bool {
		pc := v.(chan *proto.UpdateResponse)
		close(pc)
		return true
	})
}

func (c *Controller) PeerConnected(id uint32) error {
	err := c.store.UpatePeerStatus(id, true)
	if err != nil {
		return err
	}
	c.EventPeerConnected(id)
	return nil
}

func (c *Controller) PeerDisconnected(id uint32) error {
	c.DeletePeerUpdateChan(id)
	c.EventPeerDisconnected(id)
	return c.store.UpatePeerStatus(id, false)
}

func (c *Controller) GetConnectedPeers() ([]Peer, error) {
	peers, err := c.store.GetConnectedPeers()
	if err != nil {
		return nil, err
	}
	return peers, nil
}
