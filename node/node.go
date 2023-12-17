package node

// test
import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"log"
	"net"
	"net/netip"
	"strings"
	"sync"

	"github.com/flynn/noise"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/caldog20/go-overlay/header"
	"github.com/caldog20/go-overlay/msg"
)

type Node struct {
	mu      sync.RWMutex
	conns   []*Conn
	peermap *PeerMap
	keyPair noise.DHKey
	vpnip   netip.Addr
	api     msg.ControlServiceClient
	gconn   *grpc.ClientConn
	tun     *Tun
	fw      *Firewall
	localID uint32
}

func NewNode(id uint32, port uint16) *Node {
	if id == 0 {
		log.Fatal("id must not be zero")
	}

	kp, _ := TempCS.GenerateKeypair(rand.Reader)
	pmap := NewPeerMap()

	t, err := NewTun()
	if err != nil {
		log.Fatal(err)
	}

	n := &Node{
		conns:   make([]*Conn, 4),
		peermap: pmap,
		keyPair: kp,
		tun:     t,
		fw:      NewFirewall(),
		localID: id,
	}

	c := NewConn(port)
	n.conns[0] = c
	//port := c.GetLocalAddr().String()
	//portint, _ := strconv.Atoi(strings.Split(port, ":")[1])
	//for i := 1; i < 2; i++ {
	//	c := NewConn(uint16(portint))
	//	n.conns[i] = c
	//}

	return n
}

func (node *Node) Run(ctx context.Context) {
	log.SetPrefix("node: ")

	var err error
	node.gconn, err = grpc.DialContext(ctx, "10.170.241.1:5555", grpc.WithBlock(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("error connecting to grpc server: %v", err)
	}

	node.api = msg.NewControlServiceClient(node.gconn)

	lport := strings.Split(node.conns[0].GetLocalAddr().String(), ":")[1]

	reply, err := node.api.Register(ctx, &msg.RegisterRequest{
		Id:   node.localID,
		Key:  base64.StdEncoding.EncodeToString(node.keyPair.Public),
		Port: lport,
	})
	if err != nil {
		log.Fatalf("Error registering with controller: %v", err)
	}

	node.vpnip = netip.MustParseAddr(reply.VpnIp)
	log.Printf("Received VPN IP: %s", node.vpnip.String())

	err = node.tun.ConfigureInterface(node.vpnip)
	if err != nil {
		log.Fatal(err)
	}

	// go node.puncher()
	// go node.ListenUDP(ctx)

	// for i := 0; i < 2; i++ {
	go node.conns[0].ReadPackets(node.ReadUDP, 0)
	go node.tun.ReadPackets(node.handleOutbound, 0)
	//}

	// go node.conn.ReadPackets(node.ReadUDP, 0)

	<-ctx.Done()
	//node.api.Deregister(context.TODO(), &msg.DeregisterRequest{
	//	Id: node.localID,
	//})
}

func (node *Node) ReadUDP(elem *Buffer, index int) {
	err := elem.h.Parse(elem.in)
	if err != nil {
		log.Printf("error parsing header: %v", err)
		return
	}

	// Ignore Punch Packets
	if elem.h.Type == header.Punch {
		return
	}

	// Fast path, we have peer and this is a data message. try to process
	if elem.h.Type == header.Data {
		node.peermap.mu.RLock()
		peer, found := node.peermap.peerID[elem.h.ID]
		node.peermap.mu.RUnlock()
		if !found {
			log.Println("received data message for unknown peer")
			return
		}
		// peer.mu.Lock()
		// we have valid peer ready for data
		if peer.ready.Load() {
			peer.inqueue <- elem
		}
		// if peer not ready, drop since this is not a handshake message
		return
	}

	// Handle if this is a handshake message
	if elem.h.Type == header.Handshake {
		node.peermap.mu.RLock()
		peer, found := node.peermap.peerID[elem.h.ID]
		node.peermap.mu.RUnlock()

		if !found {
			peer, err = node.QueryNewPeerID(elem.h.ID)
			if err != nil {
				log.Println(err)
				return
			}
		}
		peer.handshakes <- elem
		if peer.status.Load() == 0 {
			log.Println("Starting peer")
			go peer.Start(false)
		}

	}
}

func (node *Node) handleOutbound(elem *Buffer, index int) {
	// Parse outbound packet with firewall
	var err error
	fwpacket, err := node.fw.Parse(elem.in, false)

	// TODO: Maybe let firewall handle this case for mac, linux kernel handles this for us
	tempIP := net.ParseIP(node.vpnip.String())
	if tempIP.Equal(fwpacket.RemoteIP) {
		// Packet destination IP matches local tunnel IP
		// drop packets
		return
	}

	// TODO: Move this properly checking node ip
	if drop := node.fw.Drop(fwpacket); drop {
		return
	}

	// Lookup peer by destination vpn ip
	// Fix this by updating FW to use netip.Addr
	remoteIP := netip.MustParseAddr(fwpacket.RemoteIP.String())

	node.peermap.mu.RLock()
	peer, found := node.peermap.peerIP[remoteIP]
	node.peermap.mu.RUnlock()

	if !found {
		peer, err = node.QueryNewPeerIP(remoteIP)
		if err != nil {
			log.Println(err)
			return
		}
	}

	// peer.mu.Lock()
	// Peer was found, and is ready, send data
	if peer.ready.Load() {
		peer.outqueue <- elem
		return
	}

	// Pending outbound packets
	peer.pending <- elem
	if peer.status.Load() == 0 {
		log.Println("Starting peer")
		go peer.Start(true)
	}

	//if peer.state == HandshakeInitSent {
	//	log.Printf("already sent handshake - waiting for response for peer %d", peer.remoteID)
	//	peer.mu.Unlock()
	//	return
	//}
	//
	//// Punch
	////_, err = node.api.Punch(context.TODO(), &msg.PunchRequest{SrcVpnIp: node.vpnip.String(), DstVpnIp: peer.vpnip.String()})
	////if err != nil {
	////	log.Printf("error requesting punch before handshake: %v", err)
	////}
	//
	//err = peer.NewHandshake(true, node.keyPair)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//out, _ = h.Encode(out, header.Handshake, header.Initiator, node.localID, 1)
	//out, _, _, err = peer.hs.WriteMessage(out, nil)
	//if err != nil {
	//	log.Printf("error writing handshake initiating message: %v", err)
	//	peer.ready.Store(false)
	//	peer.state = HandshakeNotStarted
	//	peer.hs = nil
	//	peer.rx = nil
	//	peer.tx = nil
	//	peer.mu.Unlock()
	//	return
	//}
	//
	//_, err = node.conns[index].uc.WriteToUDP(out, peer.remote)
	//if err != nil {
	//	peer.ready.Store(false)
	//	peer.state = HandshakeNotStarted
	//	peer.hs = nil
	//	peer.rx = nil
	//	peer.tx = nil
	//	peer.mu.Unlock()
	//	return
	//}
	//peer.state = HandshakeInitSent
	//peer.mu.Unlock()
	return
}

func (node *Node) puncher() {
	puncher, err := node.api.PunchSubscriber(context.TODO(), &msg.PunchSubscribe{Id: node.localID})
	if err != nil {
		log.Fatal(err)
	}

	h := &header.Header{}
	out := make([]byte, header.Len)
	out, err = h.Encode(out, header.Punch, header.None, 0, 0)
	for {
		req, err := puncher.Recv()
		if err != nil {
			puncher = nil
			log.Fatal("error receiving from punch stream...fatal")
			return
		}

		raddr, err := net.ResolveUDPAddr("udp4", req.Remote)
		if err != nil {
			log.Printf("failed to resolve udp address to punch towards: %v", err)
			continue
		}

		if err != nil {
			log.Printf("error encoding header for punch: %v", err)
			continue
		}

		var b int
		for i := 0; i < 3; i++ {
			n, _ := node.conns[0].uc.WriteToUDP(out, raddr)
			b += n
		}

		log.Printf("sent 5 punch packets to %s - %d bytes", req.Remote, b)
	}
}
