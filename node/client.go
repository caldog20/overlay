package node

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"net/netip"
	"time"

	"github.com/caldog20/overlay/proto"
	pb "google.golang.org/protobuf/proto"
)

func (node *Node) Login() error {
	node.noise.l.Lock()
	defer node.noise.l.Unlock()

	pubkey := base64.StdEncoding.EncodeToString(node.noise.keyPair.Public)
	login, err := node.controller.LoginPeer(context.TODO(), &proto.LoginRequest{PublicKey: pubkey})
	if err != nil {
		return err
	}

	node.id = login.Config.Id
	node.ip = netip.MustParseAddr(login.Config.TunnelIp)

	return nil
}

func (node *Node) DiscoverEndpoint() (string, error) {
	buf := make([]byte, 1500)
	dis := &proto.EndpointDiscovery{Id: node.id}
	out, err := pb.Marshal(dis)
	if err != nil {
		log.Fatal(err)
	}

	addr, _, _ := net.SplitHostPort(node.controllerAddr)
	ua, _ := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", addr, 5050))
	node.conn.WriteToUDP(out, ua)

	node.conn.uc.SetReadDeadline(time.Now().Add(time.Second * 2))
	n, _, err := node.conn.ReadFromUDP(buf)
	node.conn.uc.SetReadDeadline(time.Time{})

	if err != nil {
		log.Fatal(err)
	}
	reply := &proto.EndpointDiscoveryResponse{}
	err = pb.Unmarshal(buf[:n], reply)
	if err != nil {
		log.Fatal(err)
	}

	return reply.Endpoint, nil
}

func (node *Node) SetRemoteEndpoint(endpoint string) {
	_, err := node.controller.SetPeerEndpoint(context.TODO(), &proto.Endpoint{
		Id:       node.id,
		Endpoint: endpoint,
	})
	if err != nil {
		log.Fatal(err)
	}
}

func (node *Node) Register() error {
	node.noise.l.RLock()
	defer node.noise.l.RUnlock()
	pubkey := base64.StdEncoding.EncodeToString(node.noise.keyPair.Public)

	regmsg := &proto.RegisterRequest{
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
	stream, err := node.controller.Update(context.Background(), &proto.UpdateRequest{
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
	}()
}

func (node *Node) HandleUpdate(update *proto.UpdateResponse) {
	switch update.UpdateType {
	case proto.UpdateResponse_INIT:
		node.handleInitialSync(update)
	case proto.UpdateResponse_CONNECT:
		node.handlePeerConnectUpdate(update)
	case proto.UpdateResponse_DISCONNECT:
		//node.handlePeerDisconnectUpdate(update)
	case proto.UpdateResponse_PUNCH:
		//node.handlePeerPunchRequest(update)
	default:
		log.Println("unmatched update message type")
		return
	}
}

func (node *Node) handleInitialSync(update *proto.UpdateResponse) {
	for _, peer := range update.PeerList.RemotePeer {
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

func (node *Node) handlePeerConnectUpdate(update *proto.UpdateResponse) {
	if update.PeerList.Count < 1 {
		return
	}
	if update.UpdateType != proto.UpdateResponse_CONNECT {
		return
	}

	rp := update.PeerList.RemotePeer[0]

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
func (peer *Peer) Update(info *proto.RemotePeer) error {
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
		newRemote, err := net.ResolveUDPAddr(UDPType, newEndpoint.String())
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

//
//func (node *Node) RequestPunch(id uint32) {
//	// TODO Fix response for requesting punches
//	_, err := node.controller.PunchRequester(context.Background(), &proto.PunchRequest{
//		ReqId:    node.id,
//		RemoteId: id,
//	})
//	if err != nil {
//		log.Println(err)
//		return
//	}
//}
//
//func (node *Node) CheckPunches() {
//	buf := make([]byte, 20)
//	h := NewHeader()
//	for {
//		time.Sleep(time.Second * 2)
//		resp, err := node.controller.PunchChecker(context.Background(), &proto.PunchCheck{
//			ReqId: node.id,
//		})
//
//		if err == nil {
//			raddr, err := net.ResolveUDPAddr(UDPType, resp.Remote)
//			if err != nil {
//				log.Println(err)
//				continue
//			}
//			punch, err := h.Encode(buf, Punch, node.id, 0xff)
//
//			// Send 2 for good measure
//			for i := 0; i < 2; i++ {
//				node.conn.WriteToUDP(punch, raddr)
//			}
//
//			log.Printf("sent punch message to peer: %s", raddr.String())
//		}
//	}
//}
