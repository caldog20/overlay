package controller

import (
	"errors"

	"github.com/caldog20/overlay/proto"
)

const (
	NONE       = 0
	CONNECT    = 1
	DISCONNECT = 2
)

type Event struct {
	EventType int
	PeerID    uint32
}

func (c *Controller) EventPeerConnected(peerID uint32) {
	peer, err := c.store.GetPeerByID(peerID)
	if err != nil {
		return
	}

	// Send event to all peers about connect
	var rp []*proto.RemotePeer

	rp = append(rp, &proto.RemotePeer{Id: peerID, PublicKey: peer.PublicKey, Endpoint: peer.Endpoint, TunnelIp: peer.IP})
	update := &proto.UpdateResponse{
		UpdateType: proto.UpdateResponse_CONNECT,
		PeerList: &proto.RemotePeerList{
			Count:      1,
			RemotePeer: rp,
		},
	}

	c.peerChannels.Range(func(k, v interface{}) bool {
		if k.(uint32) != peerID {
			v.(chan *proto.UpdateResponse) <- update
		}
		return true
	})

}

func (c *Controller) EventPeerDisconnected(peerID uint32) {
	// Send event to all peers about disconnect except disconnected peer
	var rp []*proto.RemotePeer

	rp = append(rp, &proto.RemotePeer{Id: peerID})
	update := &proto.UpdateResponse{
		UpdateType: proto.UpdateResponse_DISCONNECT,
		PeerList: &proto.RemotePeerList{
			Count:      1,
			RemotePeer: rp,
		},
	}

	c.peerChannels.Range(func(k, v interface{}) bool {
		if k.(uint32) != peerID {
			v.(chan *proto.UpdateResponse) <- update
		}
		return true
	})
}

func (c *Controller) EventPunchRequest(peerID uint32, endpoint string) error {
	// TODO actually validate endpoint
	if endpoint == "" {
		return ErrInvalidEndpoint
	}

	update := &proto.UpdateResponse{
		UpdateType: proto.UpdateResponse_PUNCH,
		PeerList: &proto.RemotePeerList{
			Count: 1,
			RemotePeer: []*proto.RemotePeer{{
				Endpoint: endpoint,
			}},
		},
	}

	ch, ok := c.peerChannels.Load(peerID)
	if !ok {
		return errors.New("peer update channel not found")
	}

	ch.(chan *proto.UpdateResponse) <- update

	return nil
}

func (c *Controller) newEvent(eventType int, peerID uint32) Event {
	return Event{eventType, peerID}
}