package node

import (
	"context"
	"encoding/base64"
	"errors"
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
		endpoint = ":" + node.port
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
			raddr, err := net.ResolveUDPAddr(UdpType, resp.Remote)
			if err != nil {
				log.Println(err)
				continue
			}
			punch, err := h.Encode(buf, Punch, node.id, 0xff)

			// Send 2 for good measure
			for i := 0; i < 2; i++ {
				node.conn.WriteToUdp(punch, raddr)
			}

			log.Printf("sent punch message to peer: %s", raddr.String())
		}
	}
}
