package node

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log"
	"net"
	"net/netip"
	"strconv"
	"strings"

	"github.com/caldog20/go-overlay/header"
	"github.com/caldog20/go-overlay/msg"
	"github.com/flynn/noise"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Node struct {
	//conn    *Conn
	conns   []*Conn
	peermap *PeerMap
	keyPair noise.DHKey
	vpnip   netip.Addr
	api     msg.ControlServiceClient
	gconn   *grpc.ClientConn
	tun     *Tun
	fw      *Firewall
	pchan   chan string
	localID uint32
}

func NewNode(id uint32) *Node {
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
		//conn:    NewConn(55555),
		conns:   make([]*Conn, 4),
		peermap: pmap,
		keyPair: kp,
		tun:     t,
		fw:      NewFirewall(),
		pchan:   make(chan string),
		localID: id,
	}

	c := NewConn(0)
	n.conns[0] = c
	port := c.GetLocalAddr().String()
	portint, _ := strconv.Atoi(strings.Split(port, ":")[1])
	for i := 1; i < 2; i++ {
		c := NewConn(uint16(portint))
		n.conns[i] = c
	}

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

	//go node.puncher()
	//go node.ListenUDP(ctx)

	for i := 0; i < 2; i++ {
		go node.conns[i].ReadPackets(node.ReadUDP, i)
		go node.tun.ReadTunPackets(node.handleOutbound, i)
	}

	//go node.conn.ReadPackets(node.ReadUDP, 0)

	<-ctx.Done()
	//node.api.Deregister(context.TODO(), &msg.DeregisterRequest{
	//	Id: node.localID,
	//})
}

func (node *Node) ReadUDP(raddr *net.UDPAddr, in []byte, out []byte, h *header.Header, fwpacket *FWPacket, index int) {
	err := h.Parse(in)
	if err != nil {
		log.Printf("error parsing header: %v", err)
		return
	}

	// Ignore Punch Packets
	if h.Type == header.Punch {
		return
	}

	// Fast path, we have peer and this is a data message. try to process
	if h.Type == header.Data {
		node.peermap.mu.RLock()
		peer, found := node.peermap.peerID[h.ID]
		node.peermap.mu.RUnlock()
		if !found {
			log.Println("received data message for unknown peer")
			return
		}
		peer.mu.Lock()
		// we have valid peer ready for data
		if peer.ready.Load() {
			out, err = node.DoDecrypt(peer, in[header.Len:], out[:0], h.MsgCounter)
			if err != nil {
				log.Println(err)
				peer.mu.Unlock()
				return
			}
			peer.mu.Unlock()

			fwpacket, err = node.fw.Parse(out, true)
			if err != nil {
				log.Println(err)
				return
			}

			drop := node.fw.Drop(fwpacket)
			if !drop {
				_, err = node.tun.Write(out)
				if err != nil {
					log.Fatal(err)
				}
			}

		}
		// if peer not ready, drop since this is not a handshake message
		return
	}

	// Handle if this is a handshake message
	if h.Type == header.Handshake {
		node.peermap.mu.RLock()
		peer, found := node.peermap.peerID[h.ID]
		node.peermap.mu.RUnlock()

		if !found {
			peer, err = node.QueryNewPeerID(h.ID)
			if err != nil {
				log.Println(err)
				return
			}
		}

		// check peer handshake state, lock peer
		peer.mu.Lock()

		// handshake init msg received
		if h.SubType == header.Initiator {
			// peer waiting for handshake message, process it
			if peer.state == HandshakeNotStarted {
				peer.ready.Store(false)
				// create a new handshake and process
				err := peer.NewHandshake(false, node.keyPair)
				if err != nil {
					log.Printf("error creating new handshake state for peer: %d - %v\n", peer.remoteID, err)
					peer.mu.Unlock()
					return
				}
				_, _, _, err = peer.hs.ReadMessage(nil, in[header.Len:])
				if err != nil {
					log.Printf("error reading first handshake message: %v", err)
					peer.mu.Unlock()
					return
				}

				// Respond to handshake
				out, _ = h.Encode(out, header.Handshake, header.Responder, node.localID, 2)
				out, peer.rx, peer.tx, err = peer.hs.WriteMessage(out, nil)
				if err != nil {
					log.Printf("error writing handshake response: %v", err)
					peer.mu.Unlock()
					return
				}

				_, err = node.conns[index].uc.WriteToUDP(out, peer.remote)
				if err != nil {
					log.Fatal(err)
				}
				peer.state = HandShakeRespSent
				peer.remote = raddr // Update remote endpoint
				peer.ready.Store(true)
				peer.mu.Unlock()
				return
			}
			// We already sent handshake response but getting another handshake init message
			if peer.state == HandShakeRespSent {
				peer.ready.Store(false)
				peer.state = 0
				peer.hs = nil
				peer.rx = nil
				peer.tx = nil
				log.Printf("receieved another handshake init for already handshaked peer: %d - resetting peer state", peer.remoteID)
				peer.mu.Unlock()
				return
			}
		}

		// Handshake final response, process
		if h.SubType == header.Responder {
			if peer.state == HandshakeInitSent {
				_, peer.tx, peer.rx, err = peer.hs.ReadMessage(nil, in[header.Len:])
				if err != nil {
					log.Printf("error reading handshake response: %v - resetting peer state", err)
					peer.ready.Store(false)
					peer.state = HandshakeNotStarted
					peer.hs = nil
					peer.rx = nil
					peer.tx = nil
					peer.mu.Unlock()
					return
				}
				peer.state = HandshakeDone
				peer.remote = raddr
				peer.ready.Store(true)
				peer.mu.Unlock()
				return
			}
		}
	} else {
		log.Printf("invalid message type: %s remote: %s", h.TypeName(), raddr.String())
	}

}

func (node *Node) QueryNewPeerIP(remoteIP netip.Addr) (*Peer, error) {
	reply, err := node.api.WhoIsIp(context.TODO(), &msg.WhoIsIPRequest{VpnIp: remoteIP.String()})
	if err != nil {
		log.Println(err)
		return nil, err
	}
	peer := &Peer{
		remoteID: reply.Id,
		vpnip:    netip.MustParseAddr(reply.VpnIp),
		state:    HandshakeNotStarted,
	}

	peer.remote, _ = net.ResolveUDPAddr("udp4", reply.Remote)
	peer.rs, _ = base64.StdEncoding.DecodeString(reply.Key)

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
		remoteID: reply.Id,
		vpnip:    netip.MustParseAddr(reply.VpnIp),
		state:    HandshakeNotStarted,
	}

	peer.remote, _ = net.ResolveUDPAddr("udp4", reply.Remote)
	peer.rs, err = base64.StdEncoding.DecodeString(reply.Key)
	if err != nil {
		return nil, errors.New("error decoding peer key")
	}
	node.peermap.mu.Lock()
	defer node.peermap.mu.Unlock()

	node.peermap.peerIP[peer.vpnip] = peer
	node.peermap.peerID[peer.remoteID] = peer

	return peer, nil
	//if err != nil {
	//	log.Println("error adding peer to peermap")
	//}
}

func (node *Node) DoDecrypt(peer *Peer, in []byte, out []byte, counter uint64) ([]byte, error) {
	var err error
	out, err = peer.DoDecrypt(out, in, counter)
	return out, err
}

func (node *Node) DoEncrypt(peer *Peer, out []byte, in []byte, h *header.Header) ([]byte, error) {
	var err error
	//peer.mu.Lock()
	out, err = h.Encode(out, header.Data, header.None, node.localID, peer.tx.Nonce())
	//peer.mu.Unlock()
	out, err = peer.DoEncrypt(out, in)
	if err != nil {
		log.Println(err)
	}
	return out, err
}

func (node *Node) handleOutbound(in []byte, out []byte, h *header.Header, fwpacket *FWPacket, index int) {
	// Parse outbound packet with firewall
	var err error
	fwpacket, err = node.fw.Parse(in, false)

	if drop := node.fw.Drop(fwpacket); drop {
		return
	}

	// TODO: Maybe let firewall handle this case for mac, linux kernel handles this for us
	tempIP := net.ParseIP(node.vpnip.String())
	if tempIP.Equal(fwpacket.RemoteIP) {
		// Packet destination IP matches local tunnel IP
		// drop packets
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

	peer.mu.Lock()
	// Peer was found, and is ready, send data
	if peer.ready.Load() {
		out, err = node.DoEncrypt(peer, out, in, h)
		_, err = node.conns[index].uc.WriteToUDP(out, peer.Remote())
		if err != nil {
			log.Fatal(err)
		}
		peer.mu.Unlock()
		//log.Printf("Wrote %d bytes to peer %s", n, peer.remote.String())
		return
	}

	// If we get here, peer was found but not ready, check state and start handshake possibly
	//peer.mu.Lock()

	if peer.state == HandshakeInitSent {
		log.Printf("already sent handshake - waiting for response for peer %d", peer.remoteID)
		peer.mu.Unlock()
		return
	}

	// Punch
	//_, err = node.api.Punch(context.TODO(), &msg.PunchRequest{SrcVpnIp: node.vpnip.String(), DstVpnIp: peer.vpnip.String()})
	//if err != nil {
	//	log.Printf("error requesting punch before handshake: %v", err)
	//}

	err = peer.NewHandshake(true, node.keyPair)
	if err != nil {
		log.Fatal(err)
	}
	out, _ = h.Encode(out, header.Handshake, header.Initiator, node.localID, 1)
	out, _, _, err = peer.hs.WriteMessage(out, nil)
	if err != nil {
		log.Printf("error writing handshake initiating message: %v", err)
		peer.ready.Store(false)
		peer.state = HandshakeNotStarted
		peer.hs = nil
		peer.rx = nil
		peer.tx = nil
		peer.mu.Unlock()
		return
	}

	_, err = node.conns[index].uc.WriteToUDP(out, peer.remote)
	if err != nil {
		peer.ready.Store(false)
		peer.state = HandshakeNotStarted
		peer.hs = nil
		peer.rx = nil
		peer.tx = nil
		peer.mu.Unlock()
		return
	}
	peer.state = HandshakeInitSent
	peer.mu.Unlock()
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
