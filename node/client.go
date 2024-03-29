package node

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"log"
	"net"
	"net/netip"

	"github.com/caldog20/overlay/node/conn"
	"github.com/caldog20/overlay/pkg/header"
	controllerv1 "github.com/caldog20/overlay/proto/gen/controller/v1"
)

func (node *Node) Login() error {
	node.noise.l.Lock()
	defer node.noise.l.Unlock()

	pubkey := base64.StdEncoding.EncodeToString(node.noise.keyPair.Public)
	login, err := node.controller.LoginPeer(
		context.TODO(),
		&controllerv1.LoginRequest{PublicKey: pubkey},
	)
	if err != nil {
		return err
	}

	node.id = login.Config.Id
	node.ip = netip.MustParsePrefix(login.Config.TunnelIp)

	return nil
}

func (node *Node) SetRemoteEndpoint(endpoint string) error {
	_, err := node.controller.SetPeerEndpoint(context.TODO(), &controllerv1.Endpoint{
		Id:       node.id,
		Endpoint: endpoint,
	})
	if err != nil {
		return err
	}
	return nil
}

func (node *Node) Register() error {
	node.noise.l.RLock()
	defer node.noise.l.RUnlock()
	pubkey := base64.StdEncoding.EncodeToString(node.noise.keyPair.Public)

	regmsg := &controllerv1.RegisterRequest{
		PublicKey:   pubkey,
		RegisterKey: "registermeplz!",
	}

	_, err := node.controller.RegisterPeer(context.TODO(), regmsg)
	if err != nil {
		return err
	}

	return nil
}

func (node *Node) StartUpdateStream(ctx context.Context) {
	stream, err := node.controller.Update(context.Background(), &controllerv1.UpdateRequest{
		Id: node.id,
	})
	if err != nil {
		log.Fatal(err)
	}

	response, err := stream.Recv()
	if err != nil {
		stream = nil
		log.Fatal(err)
	}
	node.HandleUpdate(response)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				response, err = stream.Recv()
				if err != nil {
					return
				}
				node.HandleUpdate(response)
			}
		}
	}()
}

func (node *Node) HandleUpdate(update *controllerv1.UpdateResponse) {
	switch update.UpdateType {
	case controllerv1.UpdateResponse_INIT:
		node.handleInitialSync(update)
	case controllerv1.UpdateResponse_CONNECT:
		node.handlePeerConnectUpdate(update)
	case controllerv1.UpdateResponse_DISCONNECT:
		//node.handlePeerDisconnectUpdate(update)
	case controllerv1.UpdateResponse_PUNCH:
		node.handlePeerPunchRequest(update)
	default:
		log.Println("unmatched update message type")
		return
	}
}

func (node *Node) handlePeerPunchRequest(update *controllerv1.UpdateResponse) {
	endpoint := update.PeerList.Peers[0].Endpoint
	ua, err := net.ResolveUDPAddr(conn.UDPType, endpoint)
	if err != nil {
		log.Printf("error parsing udp punch address: %s", err)
		return
	}
	punch := make([]byte, 16)

	h := header.NewHeader()

	punch, err = h.Encode(punch, header.Punch, node.id, 0)
	if err != nil {
		log.Println("error encoding header for punch message")
	}

	node.conn.WriteToUDP(punch, ua)
	log.Printf("sent punch message to udp address: %s", ua.String())
}

func (node *Node) handleInitialSync(update *controllerv1.UpdateResponse) {
	for _, peer := range update.PeerList.Peers {
		p, err := node.AddPeer(peer)
		if err != nil {
			panic(err)
			continue
		}
		err = p.Start()
		if err != nil {
			panic(err)
		}
	}
}

func (node *Node) handlePeerConnectUpdate(update *controllerv1.UpdateResponse) {
	if update.PeerList.Count < 1 {
		return
	}
	if update.UpdateType != controllerv1.UpdateResponse_CONNECT {
		return
	}

	rp := update.PeerList.Peers[0]

	node.maps.l.RLock()
	p, found := node.maps.id[rp.Id]
	node.maps.l.RUnlock()
	if !found {
		peer, err := node.AddPeer(rp)
		if err != nil {
			return
		}
		err = peer.Start()
		if err != nil {
			panic(err)
		}
		return
	}

	// Peer already found, update
	err := p.Update(rp)
	if err != nil {
		panic(err)
	}
}

// // TODO Fix variable naming and compares
func (peer *Peer) Update(info *controllerv1.Peer) error {
	peer.mu.RLock()
	currentEndpoint := peer.raddr.AddrPort()
	currentKey := peer.noise.pubkey
	//currentHostname := peer.Hostname
	currentIP := peer.IP
	peer.mu.RUnlock()

	// TODO Helper function for parsing IPs
	newEndpoint, err := ParseAddrPort(info.Endpoint)
	if err != nil {
		log.Println(err)
		return err
	}

	if CompareAddrPort(currentEndpoint, newEndpoint) != 0 {
		peer.mu.Lock()
		newRemote, err := net.ResolveUDPAddr(conn.UDPType, newEndpoint.String())
		if err != nil {
			log.Println("error updating peer endpoint udp address")
		} else {
			peer.raddr = newRemote
		}
		peer.mu.Unlock()
	}

	//if strings.Compare(currentHostname, info.Hostname) != 0 {
	//	peer.mu.Lock()
	//	peer.Hostname = info.Hostname
	//	peer.mu.Unlock()
	//}

	newKey, err := DecodeBase64Key(info.PublicKey)
	if err != nil {
		log.Println(err)
		//return err
	}

	if subtle.ConstantTimeCompare(currentKey, newKey) != 1 {
		// TODO If the key has changed, we need to stop the peer and clear state,
		// update new key and restart peer completely
		panic("peer key update not yet implemented")
		peer.Stop()
		peer.mu.Lock()
		peer.noise.pubkey = newKey
		peer.mu.Unlock()
		err = peer.Start()
		if err != nil {
			panic(err)
		}
	}

	newIP, err := ParseAddr(info.TunnelIp)
	if err != nil {
		log.Println(err)
		//return err
	}

	if currentIP.Compare(newIP) != 0 {
		peer.mu.Lock()
		peer.IP = newIP
		peer.mu.Unlock()
	}

	return nil
}

func (node *Node) RequestPunch(id uint32) {
	// TODO Fix response for requesting punches
	_, err := node.controller.Punch(context.Background(), &controllerv1.PunchRequest{
		ReqPeerId: node.id,
		DstPeerId: id,
		Endpoint:  node.discoveredEndpoint.String(),
	})
	if err != nil {
		log.Println(err)
		return
	}
}
