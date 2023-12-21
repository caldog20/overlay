package node

import (
	"context"
	"encoding/base64"
	"errors"
	"log"
	"net/http"
	"net/netip"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/caldog20/overlay/proto"
	"github.com/flynn/noise"
	"golang.org/x/net/ipv4"
)

type Node struct {
	conn *Conn // This will change to multiple conns in future
	tun  *Tun
	id   uint32
	ip   netip.Addr

	maps struct {
		l  sync.RWMutex
		id map[uint32]*Peer     // for RX
		ip map[netip.Addr]*Peer // for TX
	}

	noise struct {
		l       sync.RWMutex
		keyPair noise.DHKey
	}

	controller proto.Controller
}

func NewNode(port string, controller string) (*Node, error) {
	node := new(Node)
	node.maps.id = make(map[uint32]*Peer)
	node.maps.ip = make(map[netip.Addr]*Peer)

	// Try to load key from disk
	keypair, err := LoadKeyFromDisk()
	if err != nil {
		keypair, err = CipherSuite.GenerateKeypair(nil)
		err = StoreKeyToDisk(keypair)
		if err != nil {
			log.Fatal("error storing keypair to disk")
		}
	}

	node.noise.keyPair = keypair

	listenPort, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		return nil, errors.New("invalid udp port")
	}

	node.conn, err = NewConn(uint16(listenPort))
	if err != nil {
		return nil, err
	}

	node.tun, err = NewTun()
	if err != nil {
		return nil, err
	}

	node.controller = proto.NewControllerProtobufClient(controller, &http.Client{})

	return node, nil
}

func (node *Node) Run(ctx context.Context) {
	// Register with controller
	err := node.Register()
	if err != nil {
		log.Fatal(err)
	}

	node.UpdateNodes()

	// Configure tunnel ip/routes
	err = node.tun.ConfigureInterface(node.ip)
	if err != nil {
		log.Fatal(err)
	}

	go node.conn.ReadPackets(node.OnUdpPacket, 0)
	go node.tun.ReadPackets(node.OnTunnelPacket)

	// TODO
	<-ctx.Done()

	for _, peer := range node.maps.id {
		if peer.running.Load() {
			peer.cancel()
		}
	}
}

func (node *Node) Register() error {
	hn, _ := os.Hostname()
	hostname := strings.Split(hn, ".")[0]

	registration, err := node.controller.Register(context.Background(), &proto.RegisterRequest{
		Key:      base64.StdEncoding.EncodeToString(node.noise.keyPair.Public),
		Hostname: hostname,
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
			_, err := node.AddPeer(n)
			if err != nil {
				panic(err)
			}
		}
	}

	log.Println("Update nodes complete")
	return nil
}

func (node *Node) OnUdpPacket(buffer *InboundBuffer, index int) {
	err := buffer.header.Parse(buffer.in)
	if err != nil {
		log.Println(err)
		PutInboundBuffer(buffer)
		return
	}

	// Lookup Peer based on index
	sender := buffer.header.SenderIndex

	node.maps.l.RLock()
	peer, found := node.maps.id[sender]
	node.maps.l.RUnlock()

	if !found {
		// TODO FIgure out logic when peer not found
		// Peer not found in table, ask for update and try again later?
		PutInboundBuffer(buffer)
		log.Printf("[inbound] peer with index %d not found", sender)
		node.UpdateNodes()
		return
	}

	// TODO temporary sanity check if peer is somehow nil
	if peer == nil {
		log.Fatal("Peer is nil!!!")
	}

	buffer.peer = peer
	// Peer found, check message type and handle accordingly
	switch buffer.header.Type {
	// Remote peer sent handshake message
	case Handshake:
		// Callee responsible to returning buffer to pool
		peer.handshakes <- buffer
		if peer.running.Load() == false {
			go peer.Run(false)
		}
		return
	// Remote peer sent encrypted data
	case Data:
		// Callee responsible to returning buffer to pool
		peer.inbound <- buffer
		return
	// Remote peer sent punch packet
	case Punch:
		log.Printf("[inbound] received punch packet from peer %d", sender)
		PutInboundBuffer(buffer)
		return
	default:
		log.Printf("[inbound] unmatched header: %s", buffer.header.String())
		PutInboundBuffer(buffer)
		return
	}

}

func (node *Node) OnTunnelPacket(buffer *OutboundBuffer) {
	ipHeader, err := ipv4.ParseHeader(buffer.packet)
	if err != nil {
		log.Println("[outbound] failed to parse ipv4 header")
		PutOutboundBuffer(buffer)
		return
	}

	// TODO Move this
	if ipHeader.Dst.Equal(node.ip.AsSlice()) {
		// destination is local tunnel, drop
		PutOutboundBuffer(buffer)
		return
	}

	dst, _ := netip.AddrFromSlice(ipHeader.Dst.To4())
	net, _ := node.ip.Prefix(24)
	if !net.Contains(dst) {
		// destination is not in network, drop
		PutOutboundBuffer(buffer)
		return
	}

	// Lookup peer
	node.maps.l.RLock()
	peer, found := node.maps.ip[dst]
	node.maps.l.RUnlock()
	if !found {
		// peer not found, drop
		log.Printf("[outbound] peer with ip %s not found", dst.String())
		PutOutboundBuffer(buffer)
		node.UpdateNodes()
		return
	}

	peer.OutboundPacket(buffer)

	return
}
