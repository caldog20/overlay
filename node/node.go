package node

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"github.com/caldog20/go-overlay/header"
	"github.com/caldog20/go-overlay/msg"
	noiseimpl "github.com/caldog20/go-overlay/noise"
	"github.com/caldog20/go-overlay/tun"
	"github.com/flynn/noise"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"net"
	"net/netip"
)

type Node struct {
	conn    *net.UDPConn
	peermap *PeerMap
	keyPair noise.DHKey
	vpnip   netip.Addr
	api     msg.ControlServiceClient
	gconn   *grpc.ClientConn
	tun     *tun.Tun
	fw      *Firewall
	pchan   chan string
}

func NewNode() *Node {

	kp, _ := noiseimpl.TempCS.GenerateKeypair(rand.Reader)
	pmap := NewPeerMap()

	// Set up UDP Listen Socket
	laddr, err := net.ResolveUDPAddr("udp4", ":4444")
	if err != nil {
		log.Fatal(err)
	}

	c, err := net.ListenUDP("udp4", laddr)
	if err != nil {
		log.Fatal(err)
	}

	t, err := tun.NewTun()
	if err != nil {
		log.Fatal(err)
	}

	n := &Node{
		conn:    c,
		peermap: pmap,
		keyPair: kp,
		tun:     t,
		fw:      NewFirewall(),
		pchan:   make(chan string),
	}

	return n
}

func (node *Node) QueryNewPeer(remoteIP netip.Addr) {
	reply, err := node.api.WhoIsIp(context.TODO(), &msg.WhoIsIPRequest{VpnIp: remoteIP.String()})
	if err != nil {
		log.Println(err)
		return
	}
	peer := &Peer{
		localID:  GenerateID(),
		remoteID: 0,
		ready:    false,
		vpnip:    netip.MustParseAddr(reply.Remote.VpnIp),
		state:    HandshakeNotStarted,
	}

	peer.remote, _ = net.ResolveUDPAddr("udp4", reply.Remote.Remote)
	peer.rs, _ = base64.StdEncoding.DecodeString(reply.Remote.Id)

	err = node.peermap.AddPeer(peer)
	if err != nil {
		log.Println("error adding peer to peermap")
	}
}

func (node *Node) handleInbound() {
	in := make([]byte, 1400)
	out := make([]byte, 1400)
	h := &header.Header{}
	fwpacket := &FWPacket{}
	for {
		n, raddr, err := node.conn.ReadFromUDP(in)
		if err != nil {
			log.Fatal(err)
		}

		err = h.Parse(in[:n])
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("[%s] received %d bytes", raddr.String(), n)

		if h.Type == header.Punch {
			// Drop received punch packets
			log.Printf("Received punch packets from %s", raddr.String())
			continue
		}

		log.Printf("Looking up peer with ID %d", h.ID)
		peer := node.peermap.ContainsRemoteID(h.ID)

		if peer == nil && h.Type == header.Handshake {
			// Peer trying to handshake, lets respond
			if h.SubType == header.Initiator {
				peer = &Peer{localID: GenerateID(), remoteID: h.ID, remote: raddr, ready: false, state: HandShakeRespSent}
				peer.NewHandshake(false, node.keyPair)
				// need to add methods or lock peer mutex for this stuff later
				// Read handshake message and response
				_, _, _, err = peer.hs.ReadMessage(nil, in[header.Len:n])
				if err != nil {
					log.Printf("error reading first handshake message: %v", err)
					continue
				}
				// Respond to handshake
				out, _ = h.Encode(out, header.Handshake, header.Responder, peer.localID, 2)
				out, peer.rx, peer.tx, err = peer.hs.WriteMessage(out, nil)
				if err != nil {
					log.Printf("error writing handshake response: %v", err)
					continue
				}

				n, err = node.conn.WriteToUDP(out, peer.remote)
				if err != nil {
					log.Fatal(err)
				}

				// Temporarily query for peer VPN IP
				pid := base64.StdEncoding.EncodeToString(peer.hs.PeerStatic())
				resp, _ := node.api.WhoIsID(context.TODO(), &msg.WhoIsIDRequest{Id: pid})

				if err != nil {
					log.Fatal(err)
				}
				log.Printf("wrote handshake response to peer - %d bytes", n)
				peer.UpdateState(HandshakeDone)
				peer.UpdateStatus(true)
				peer.vpnip = netip.MustParseAddr(resp.Remote.VpnIp)
				node.peermap.AddPeerWithIndices(peer)
				continue
			}

			if h.SubType == header.Responder {
				peer = node.peermap.ContainsRemoteID(h.ID)
				if peer == nil {
					// Search by remote IP in pending
					peer = node.peermap.ContainsPendingRemote(raddr)
					if peer == nil {
						log.Println("no pending peer to complete handshake with")
						continue
					}
				}
				_, peer.tx, peer.rx, err = peer.hs.ReadMessage(nil, in[header.Len:n])
				if err != nil {
					log.Fatalf("error reading handshake response: %v", err)

				}
				peer.UpdateStatus(true)
				peer.remoteID = h.ID
				node.peermap.AddPeerWithIndices(peer)
				node.peermap.DeletePendingPeer(peer)
				log.Printf("handshake completed with peer %s : %s", peer.vpnip.String(), peer.remote.String())
				continue
			}
		}

		if h.Type == header.Data {
			if !peer.isReady() {
				log.Println("peer not ready...cant read regular data")
				continue
			}

			h.Parse(in[:n])
			//peer.rx.SetNonce(22)
			data, err := peer.rx.Decrypt(nil, nil, in[header.Len:n])
			if err != nil {
				peer.UpdateStatus(false)
				log.Printf("error decrypting data packet: %v", err)
				continue
			}

			log.Println("Decrypted Data packet")

			fwpacket, err = node.fw.Parse(data, true)
			if err != nil {
				log.Println(err)
			}
			if drop := node.fw.Drop(fwpacket); drop {
				continue
			}

			node.tun.Write(data)
		}

	}
}

//func (node *Node) pre(raddr *net.UDPAddr) {
//	h := &header.Header{}
//	out := make([]byte, header.Len+4)
//
//	out, _ = h.Encode(out, header.Punch, header.None, 0, 0)
//
//	var b int
//	for i := 0; i < 5; i++ {
//		n, _ := node.conn.WriteToUDP(out, raddr)
//		b += n
//	}
//
//}

func (node *Node) handleOutbound() {
	in := make([]byte, 1400)
	out := make([]byte, 1400)
	h := &header.Header{}
	fwpacket := &FWPacket{}

	for {
		// Read from tunnel interface
		n, err := node.tun.Read(in)
		if err != nil {
			log.Fatal(err)
		}

		// Parse outbound packet with firewall
		fwpacket, err = node.fw.Parse(in[:n], false)

		if drop := node.fw.Drop(fwpacket); drop {
			continue
		}

		// TODO: Maybe let firewall handle this case for mac, linux kernel handles this for us
		tempIP := net.ParseIP(node.vpnip.String())
		if tempIP.Equal(fwpacket.RemoteIP) {
			// Packet destination IP matches local tunnel IP
			// drop packets
			continue
		}

		// Lookup peer by destination vpn ip
		// Fix this by updating FW to use netip.Addr
		remoteIP := netip.MustParseAddr(fwpacket.RemoteIP.String())
		peer := node.peermap.Contains(remoteIP)

		// We don't know this peer, so ask about it from server
		// Add to peer list and handle next go around
		if peer == nil {
			_, ok := node.peermap.pending[node.vpnip]
			// peer pending hsake complete, skip query
			if !ok {
				// Not pending, query server
				node.QueryNewPeer(remoteIP)
			}
			continue
		}

		// Peer was found, check state to see if its ready otherwise send it to handshake
		if peer.isReady() != true {
			if peer.state == HandshakeInitSent {
				continue
			}
			_, err = node.api.Punch(context.TODO(), &msg.PunchRequest{SrcVpnIp: node.vpnip.String(), DstVpnIp: peer.vpnip.String()})
			if err != nil {
				log.Printf("error requesting punch before handshake: %v", err)
			}

			err = peer.NewHandshake(true, node.keyPair)
			if err != nil {
				log.Fatal(err)
			}
			out, _ = h.Encode(out, header.Handshake, header.Initiator, peer.localID, 1)
			out, _, _, err = peer.hs.WriteMessage(out, nil)
			if err != nil {
				log.Printf("error writing handshake initiating message: %v", err)
				return
			}
			_, err = node.conn.WriteToUDP(out, peer.remote)
			if err != nil {
				log.Fatal(err)
			}
			peer.UpdateState(HandshakeInitSent)
			node.peermap.AddPeerPending(peer)
			continue
		}

		out, err = h.Encode(out, header.Data, header.None, peer.localID, 0)
		if err != nil {
			log.Printf("error encoding header for data packet: %v", err)
			continue
		}
		//peer.tx.SetNonce(22)
		encrypted, err := peer.tx.Encrypt(out, nil, in[:n])
		if err != nil {
			log.Printf("error encryping data packet: %v", err)
			continue
		}

		n, err = node.conn.WriteToUDP(encrypted, peer.remote)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Wrote %d bytes to peer %s", n, peer.remote.String())
	}

	//// preset peer here since we are testing and know what peer we want to send to
	//
	//p := node.peermap.Contains(netip.MustParseAddr("192.168.1.1"))
	//if p == nil {
	//	log.Fatal("cant find predetermined peer")
	//}
	//
	//p.NewHandshake(true, node.keyPair)
	//
	//out, _ = h.Encode(out, header.Handshake, header.Initiator, p.localID, 1)
	//log.Println("writing first handshake message")
	//
	//out, _, _, err = p.hs.WriteMessage(out, nil)
	//if err != nil {
	//	log.Printf("error writing handshake initiating message: %v", err)
	//	return
	//}
	//
	//n, _ := node.conn.WriteToUDP(out, p.remote)
	//
	//n, err = node.conn.Read(in)
	//h.Parse(in[:n])
	//
	//if h.Type == header.Handshake && h.SubType == header.Responder {
	//	_, tx, rx, err := p.hs.ReadMessage(nil, in[header.Len:n])
	//	if err != nil {
	//		log.Printf("error reading handshake response: %v", err)
	//		return
	//	}
	//	p.tx = tx
	//	p.rx = rx
	//	p.ready = true
	//	p.remoteID = h.ID
	//	node.peermap.AddPeerWithIndices(p)
	//}
	//
	//out, _ = h.Encode(out, header.Data, header.None, p.localID, 3)
	//t := []byte("Encrypted Channel Working!!!")
	//out, err = p.tx.Encrypt(out, nil, t)
	//if err != nil {
	//	log.Printf("error encrypting regular data packet: %v", err)
	//	return
	//}
	//
	//n, _ = node.conn.WriteToUDP(out, p.remote)
	//log.Printf("wrote %d bytes to %s", n, p.remote.String())
}

func (node *Node) puncher() {
	puncher, err := node.api.PunchSubscriber(context.TODO(), &msg.PunchSubscribe{VpnIp: node.vpnip.String()})
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
			n, _ := node.conn.WriteToUDP(out, raddr)
			b += n
		}

		log.Printf("sent 5 punch packets to %s - %d bytes", req.Remote, b)
	}

}

func (node *Node) Run(ctx context.Context) {
	log.SetPrefix("node: ")

	var err error
	node.gconn, err = grpc.DialContext(ctx, "10.170.241.1:5555", grpc.WithBlock(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("error connecting to grpc server: %v", err)
	}

	node.api = msg.NewControlServiceClient(node.gconn)

	reply, err := node.api.Register(ctx, &msg.RegisterRequest{
		Id:   base64.StdEncoding.EncodeToString(node.keyPair.Public),
		Port: "4444",
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

	go node.puncher()
	go node.handleInbound()
	go node.handleOutbound()

	<-ctx.Done()

}
