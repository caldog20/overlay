package controller

import (
	"log"

	apiv1 "github.com/caldog20/overlay/proto/gen/api/v1"
)

func (c *Controller) EventPeerConnected(peerID uint32) {
	peer, err := c.store.GetPeerByID(peerID)
	if err != nil {
		log.Printf("error sending peer connected event: %s", err)
	}

	var remotePeers []*apiv1.Peer
	remotePeers = append(remotePeers, peer.Proto())

	update := &apiv1.UpdateResponse{
		UpdateType: apiv1.UpdateType_UPDATE_TYPE_CONNECT,
		PeerList: &apiv1.RemotePeerList{
			Count:       uint32(len(remotePeers)),
			RemotePeers: remotePeers,
		}}

	c.peerChannels.Range(func(k, v interface{}) bool {
		// Update all peers except connecting peer
		if k.(uint32) != peerID {
			ch := v.(chan *apiv1.UpdateResponse)
			ch <- update
		}
		return true
	})
}
