package node

import (
	"context"
	"encoding/base64"
	"errors"
	"github.com/caldog20/go-overlay/msg"
	"log"
	"net"
	"net/netip"
	"time"
)

func (node *Node) QueryNewPeerIP(remoteIP netip.Addr) (*Peer, error) {
	reply, err := node.api.WhoIsIp(context.TODO(), &msg.WhoIsIPRequest{VpnIp: remoteIP.String()})
	if err != nil {
		log.Println(err)
		return nil, err
	}
	peer := &Peer{
		node:       node,
		remoteID:   reply.Id,
		vpnip:      netip.MustParseAddr(reply.VpnIp),
		state:      HandshakeNotStarted,
		inqueue:    make(chan *Buffer, 1024),
		outqueue:   make(chan *Buffer, 1024),
		handshakes: make(chan *Buffer, 10),
		pending:    make(chan *Buffer, 10),
	}

	peer.timers.handshake = time.NewTicker(time.Second * 10)
	peer.timers.rxtx = time.NewTicker(time.Second * 10)

	peer.remote, _ = net.ResolveUDPAddr("udp4", reply.Remote)
	peer.rs, _ = base64.StdEncoding.DecodeString(reply.Key)

	log.Println("Queried new peer")

	node.peermap.mu.Lock()
	defer node.peermap.mu.Unlock()

	node.peermap.peerIP[peer.vpnip] = peer
	node.peermap.peerID[peer.remoteID] = peer

	return peer, nil
	//if err != nil {
	//	log.Println("error adding peer to peermap")
	//}
}

func (node *Node) QueryNewPeerID(remoteID uint32) (*Peer, error) {
	reply, err := node.api.WhoIsID(context.TODO(), &msg.WhoIsIDRequest{Id: remoteID})
	if err != nil {
		return nil, err
	}
	peer := &Peer{
		node:       node,
		remoteID:   reply.Id,
		vpnip:      netip.MustParseAddr(reply.VpnIp),
		state:      HandshakeNotStarted,
		inqueue:    make(chan *Buffer, 1024),
		outqueue:   make(chan *Buffer, 1024),
		handshakes: make(chan *Buffer, 10),
		pending:    make(chan *Buffer, 10),
	}

	peer.remote, _ = net.ResolveUDPAddr("udp4", reply.Remote)
	peer.rs, err = base64.StdEncoding.DecodeString(reply.Key)
	if err != nil {
		return nil, errors.New("error decoding peer key")
	}
	log.Println("Queried new peer")
	node.peermap.mu.Lock()
	defer node.peermap.mu.Unlock()

	node.peermap.peerIP[peer.vpnip] = peer
	node.peermap.peerID[peer.remoteID] = peer

	return peer, nil
	//if err != nil {
	//	log.Println("error adding peer to peermap")
	//}
}
