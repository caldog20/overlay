package node

import (
	"context"
	"errors"
	"log"
	"net"
	"net/netip"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/caldog20/overlay/conn"
	"github.com/caldog20/overlay/tun"
	"github.com/flynn/noise"
	"golang.org/x/net/ipv4"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"github.com/caldog20/overlay/proto"
)

type Node struct {
	conn *conn.Conn // This will change to multiple conns in future
	tun  tun.Tun
	id   uint32
	ip   netip.Prefix

	prefOutboundIP     netip.Addr
	discoveredEndpoint netip.AddrPort

	maps struct {
		l  sync.RWMutex
		id map[uint32]*Peer     // for RX
		ip map[netip.Addr]*Peer // for TX
	}

	noise struct {
		l       sync.RWMutex
		keyPair noise.DHKey
	}

	running atomic.Bool

	controller proto.ControlPlaneClient
	// Temp
	port           uint16
	controllerAddr string
}

func NewNode(port uint16, controller string) (*Node, error) {
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

	if port > 65535 {
		return nil, errors.New("invalid udp port")
	}

	node.conn, err = conn.NewConn(port)
	if err != nil {
		return nil, err
	}

	_, p, err := net.SplitHostPort(node.conn.LocalAddr().String())
	if err != nil {
		return nil, err
	}

	finalPort, err := strconv.ParseUint(p, 10, 16)
	if err != nil {
		return nil, err
	}

	node.port = uint16(finalPort)

	node.tun, err = tun.NewTun()
	if err != nil {
		return nil, err
	}

	// TODO Fix this/move when fixing login/register flow
	ctx, _ := context.WithTimeout(context.Background(), time.Second*10)
	gconn, err := grpc.DialContext(ctx, controller, grpc.WithBlock(), grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		log.Fatal("error connecting to controller grpc: ", err)
	}

	node.controller = proto.NewControlPlaneClient(gconn)

	node.controllerAddr = controller
	return node, nil
}

func (node *Node) Run(ctx context.Context) {
	// Register with controller
	err := node.Login()
	if err != nil {
		s, _ := status.FromError(err)
		if s.Code() == codes.NotFound {
			err = node.Register()
			if err != nil {
				panic(err)
			}
			err = node.Login()
			if err != nil {
				panic(err)
			}
		} else {
			panic(err)
		}
	}

	// Initially set endpoints
	ep, err := node.DiscoverPublicEndpoint()
	if err != nil {
		log.Fatal(err)
	}

	node.SetRemoteEndpoint(ep)

	// Configure tunnel ip/routes
	err = node.tun.ConfigureIPAddress(node.ip)
	if err != nil {
		log.Fatal(err)
	}

	node.StartUpdateStream(ctx)

	go node.ReadUDPPackets(node.OnUDPPacket, 0)
	go node.ReadTunPackets(node.OnTunnelPacket)

	// TODO
	<-ctx.Done()

	node.conn.Close()
	node.tun.Close()

	//for _, peer := range node.maps.id {
	//	if peer.running.Load() {
	//		peer.cancel()
	//	}
	//}
}

func (node *Node) OnUDPPacket(buffer *InboundBuffer, index int) {
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
		//node.UpdateNodes()
		return
	}

	// TODO temporary sanity check if peer is somehow nil
	if peer == nil {
		log.Fatal("Peer is nil")
	}

	buffer.peer = peer
	// Peer found, check message type and handle accordingly
	switch buffer.header.Type {
	// Remote peer sent handshake message
	case Handshake:
		// Callee responsible to returning buffer to pool
		if peer.running.Load() {
			peer.handshakes <- buffer
		}
		return
	// Remote peer sent encrypted data
	case Data:
		// Callee responsible to returning buffer to pool
		peer.InboundPacket(buffer)
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
	if ipHeader.Dst.Equal(node.ip.Addr().AsSlice()) {
		// destination is local tunnel, drop
		PutOutboundBuffer(buffer)
		return
	}

	dst, _ := netip.AddrFromSlice(ipHeader.Dst.To4())
	if !node.ip.Masked().Contains(dst) {
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
		//node.UpdateNodes()
		return
	}

	peer.OutboundPacket(buffer)

	return
}
