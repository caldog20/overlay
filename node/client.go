package node

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net"
	"net/netip"
	"os"
	"strings"
	"time"

	"github.com/caldog20/overlay/proto"
)

func (node *Node) Register() error {
	hn, _ := os.Hostname()
	hostname := strings.Split(hn, ".")[0]

	endpoint, err := node.TempAddrDiscovery()

	// TODO Fix
	if err != nil {
		endpoint = fmt.Sprintf(":%d", node.port)
	}

	registration, err := node.controller.Register(context.Background(), &proto.RegisterRequest{
		Key:      base64.StdEncoding.EncodeToString(node.noise.keyPair.Public),
		Hostname: hostname,
		Endpoint: endpoint,
	})
	if err != nil {
		return errors.New("error registering with controller")
	}

	node.id = registration.Id
	node.ip = netip.MustParseAddr(registration.Ip)

	return nil
}

func (node *Node) UpdateNodes() error {
	resp, err := node.controller.NodeList(context.Background(), &proto.NodeListRequest{Id: node.id})
	if err != nil {
		panic(err)
	}

	var new []*proto.Node

	node.maps.l.RLock()
	for _, n := range resp.Nodes {
		_, ok := node.maps.id[n.Id]
		if !ok {
			new = append(new, n)
		}
	}
	node.maps.l.RUnlock()

	if len(new) > 0 {
		for _, n := range new {
			p, err := node.AddPeer(n)
			if err != nil {
				panic(err)
			}
			err = p.Start()
			if err != nil {
				panic(err)
			}
		}
	}

	log.Println("Update nodes complete")
	return nil
}

func (node *Node) TempAddrDiscovery() (string, error) {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, 8675309)

	addr, _, _ := net.SplitHostPort(node.controllerAddr[7:])
	raddr, _ := net.ResolveUDPAddr(UDPType, addr+":7979")

	node.conn.WriteToUDP(b, raddr)
	rx := make([]byte, 256)

	node.conn.uc.SetReadDeadline(time.Now().Add(time.Second * 2))
	n, _, err := node.conn.ReadFromUDP(rx)

	node.conn.uc.SetReadDeadline(time.Time{})

	if err != nil {
		return "", errors.New("Discovery failed")
	}

	addrPort, err := netip.ParseAddrPort(string(rx[:n]))
	if err != nil {
		return "", errors.New("Parsing AddrPort failed")
	}

	return addrPort.String(), nil
}
func (node *Node) RequestPunch(id uint32) {
	// TODO Fix response for requesting punches
	_, err := node.controller.PunchRequester(context.Background(), &proto.PunchRequest{
		ReqId:    node.id,
		RemoteId: id,
	})
	if err != nil {
		log.Println(err)
		return
	}
}

func (node *Node) CheckPunches() {
	buf := make([]byte, 20)
	h := NewHeader()
	for {
		time.Sleep(time.Second * 2)
		resp, err := node.controller.PunchChecker(context.Background(), &proto.PunchCheck{
			ReqId: node.id,
		})

		if err == nil {
			raddr, err := net.ResolveUDPAddr(UDPType, resp.Remote)
			if err != nil {
				log.Println(err)
				continue
			}
			punch, err := h.Encode(buf, Punch, node.id, 0xff)

			// Send 2 for good measure
			for i := 0; i < 2; i++ {
				node.conn.WriteToUDP(punch, raddr)
			}

			log.Printf("sent punch message to peer: %s", raddr.String())
		}
	}
}
