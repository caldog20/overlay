package node

import (
	"context"
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

	endpoint, _ := node.DiscoverEndpoint()

	pubkey := base64.StdEncoding.EncodeToString(node.noise.keyPair.Public)
	login, err := node.controller.Login(context.TODO(), &proto.LoginRequest{PublicKey: pubkey, Endpoint: &proto.Endpoint{Endpoint: endpoint}})
	if err != nil {
		log.Fatal(err)
	}

	node.id = login.Id
	node.ip = netip.MustParseAddr(login.Config.TunnelIp)

	return nil
}

func (node *Node) DiscoverEndpoint() (string, error) {
	buf := make([]byte, 1500)
	dis := &proto.DiscoverEndpoint{Id: 1}
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
	reply := &proto.Endpoint{}
	err = pb.Unmarshal(buf[:n], reply)
	if err != nil {
		log.Fatal(err)
	}

	return reply.Endpoint, nil
}

//func (node *Node) Register() error {
//	hn, _ := os.Hostname()
//	hostname := strings.Split(hn, ".")[0]
//
//	endpoint, err := node.TempAddrDiscovery()
//
//	// TODO Fix
//	if err != nil {
//		endpoint = fmt.Sprintf(":%d", node.port)
//	}
//
//	registration, err := node.controller.Register(context.Background(), &proto.RegisterRequest{
//		Key:      base64.StdEncoding.EncodeToString(node.noise.keyPair.Public),
//		Hostname: hostname,
//		Endpoint: endpoint,
//	})
//	if err != nil {
//		return errors.New("error registering with controller")
//	}
//
//	node.id = registration.Id
//	node.ip = netip.MustParseAddr(registration.Ip)
//
//	return nil
//}
//
//func (node *Node) UpdateNodes() error {
//	resp, err := node.controller.NodeList(context.Background(), &proto.NodeListRequest{Id: node.id})
//	if err != nil {
//		log.Println(err)
//		return err
//	}
//
//	var new []*proto.Node
//
//	node.maps.l.RLock()
//	for _, n := range resp.Nodes {
//		p, found := node.maps.id[n.Id]
//		if !found {
//			new = append(new, n)
//		} else {
//			p.Update(n)
//		}
//	}
//
//	node.maps.l.RUnlock()
//
//	if len(new) > 0 {
//		for _, n := range new {
//			p, err := node.AddPeer(n)
//			if err != nil {
//				panic(err)
//			}
//			err = p.Start()
//			if err != nil {
//				panic(err)
//			}
//		}
//	}
//
//	log.Println("Update nodes complete")
//	return nil
//}
//
//// TODO Fix variable naming and compares
//func (peer *Peer) Update(info *proto.Node) error {
//	peer.mu.RLock()
//	currentEndpoint := peer.raddr.AddrPort()
//	currentKey := peer.noise.pubkey
//	currentHostname := peer.Hostname
//	currentIP := peer.IP
//	peer.mu.RUnlock()
//
//	// TODO Helper function for parsing IPs
//	newEndpoint, err := ParseAddrPort(info.Endpoint)
//	if err != nil {
//		log.Println(err)
//		return err
//	}
//
//	if CompareAddrPort(currentEndpoint, newEndpoint) != 0 {
//		peer.mu.Lock()
//		newRemote, err := net.ResolveUDPAddr(UDPType, newEndpoint.String())
//		if err != nil {
//			log.Println("error updating peer endpoint udp address")
//		} else {
//			peer.raddr = newRemote
//		}
//		peer.mu.Unlock()
//	}
//
//	if strings.Compare(currentHostname, info.Hostname) != 0 {
//		peer.mu.Lock()
//		peer.Hostname = info.Hostname
//		peer.mu.Unlock()
//	}
//
//	newKey, err := DecodeBase64Key(info.Key)
//	if err != nil {
//		log.Println(err)
//		//return err
//	}
//
//	if subtle.ConstantTimeCompare(currentKey, newKey) != 1 {
//		// TODO If the key has changed, we need to stop the peer and clear state,
//		// update new key and restart peer completely
//		panic("peer key update not yet implemented")
//		peer.Stop()
//		peer.mu.Lock()
//		peer.noise.pubkey = newKey
//		peer.mu.Unlock()
//		err = peer.Start()
//		if err != nil {
//			panic(err)
//		}
//	}
//
//	newIP, err := ParseAddr(info.Ip)
//	if err != nil {
//		log.Println(err)
//		//return err
//	}
//
//	if currentIP.Compare(newIP) != 0 {
//		peer.mu.Lock()
//		peer.IP = newIP
//		peer.mu.Unlock()
//	}
//
//	return nil
//}
//
//func (node *Node) TempAddrDiscovery() (string, error) {
//	b := make([]byte, 4)
//	binary.BigEndian.PutUint32(b, 8675309)
//
//	addr, _, _ := net.SplitHostPort(node.controllerAddr[7:])
//	raddr, _ := net.ResolveUDPAddr(UDPType, addr+":7979")
//
//	node.conn.WriteToUDP(b, raddr)
//	rx := make([]byte, 256)
//
//	node.conn.uc.SetReadDeadline(time.Now().Add(time.Second * 2))
//	n, _, err := node.conn.ReadFromUDP(rx)
//
//	node.conn.uc.SetReadDeadline(time.Time{})
//
//	if err != nil {
//		return "", errors.New("Discovery failed")
//	}
//
//	addrPort, err := netip.ParseAddrPort(string(rx[:n]))
//	if err != nil {
//		return "", errors.New("Parsing AddrPort failed")
//	}
//
//	return addrPort.String(), nil
//}
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
