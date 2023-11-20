package node

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/caldog20/go-overlay/header"
	"github.com/caldog20/go-overlay/msg"
	"github.com/caldog20/go-overlay/tun"
	"github.com/flynn/noise"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"net"
	"net/netip"
	"strings"
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
	localID uint32
}

func NewNode(id uint32) *Node {
	if id == 0 {
		log.Fatal("id must not be zero")
	}
	kp, _ := TempCS.GenerateKeypair(rand.Reader)
	pmap := NewPeerMap()

	// Set up UDP Listen Socket
	laddr, err := net.ResolveUDPAddr("udp4", ":0")
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
		localID: id,
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

	lport := strings.Split(node.conn.LocalAddr().String(), ":")[1]

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

	go node.puncher()
	for i := 0; i < 2; i++ {
		go node.ListenUDP(ctx)
		go node.handleOutbound()
	}

	<-ctx.Done()
	node.api.Deregister(context.TODO(), &msg.DeregisterRequest{
		Id: node.localID,
	})
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

func (node *Node) DoDecrypt(peer *Peer, in []byte, fwpacket *FWPacket, counter uint64) error {
	// Lock peer from any changes
	peer.mu.Lock()
	// Set nonce from received header
	peer.rx.SetNonce(counter)
	// try to decrypt data
	data, err := peer.rx.Decrypt(nil, nil, in)
	// Need to close and rehandshake here
	if err != nil {
		peer.ready.Store(false)
		peer.hs = nil
		peer.state = 0
		log.Printf("error decrypting data packet: %v", err)
		peer.mu.Unlock()
		return fmt.Errorf("error decrypting packet from peer id: %d - %v\n", peer.remoteID, err)
	}

	//log.Println("Decrypted Data packet")
	// Decrypted packet, check inner packet
	fwpacket, err = node.fw.Parse(data, true)
	if err != nil {
		log.Println(err)
		peer.mu.Unlock()
		return err
	}

	drop := node.fw.Drop(fwpacket)
	if !drop {
		node.tun.Write(data)
	}

	peer.mu.Unlock()
	return nil
}

func (node *Node) ListenUDP(ctx context.Context) {
	in := make([]byte, 1400)
	out := make([]byte, 1400)
	h := &header.Header{}
	fwpacket := &FWPacket{}

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Read from UDP socket
		n, raddr, err := node.conn.ReadFromUDP(in)
		if err != nil {
			log.Fatal(err)
		}

		// Parse header
		err = h.Parse(in[:n])
		if err != nil {
			log.Println(err)
			continue
		}

		// ignore punch packets
		if h.Type == header.Punch {
			continue
		}

		// if peer not found, query for it
		// this locks peermap

		// Fast path, we have peer and this is a data message. try to process
		if h.Type == header.Data {
			node.peermap.mu.RLock()
			peer, found := node.peermap.peerID[h.ID]
			node.peermap.mu.RUnlock()
			if !found {
				log.Println("received data message for unknown peer")
				continue
			}
			// we have valid peer ready for data
			if peer.ready.Load() {
				err = node.DoDecrypt(peer, in[header.Len:n], fwpacket, h.MsgCounter)
				if err != nil {
					log.Println(err)
				}
			}
			// if peer not ready, drop since this is not a handshake message
			continue
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
					continue
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
						continue
					}
					_, _, _, err = peer.hs.ReadMessage(nil, in[header.Len:n])
					if err != nil {
						log.Printf("error reading first handshake message: %v", err)
						peer.mu.Unlock()
						continue
					}

					// Respond to handshake
					out, _ = h.Encode(out, header.Handshake, header.Responder, node.localID, 2)
					out, peer.rx, peer.tx, err = peer.hs.WriteMessage(out, nil)
					if err != nil {
						log.Printf("error writing handshake response: %v", err)
						peer.mu.Unlock()
						continue
					}

					n, err = node.conn.WriteToUDP(out, peer.remote)
					if err != nil {
						log.Fatal(err)
					}
					peer.state = HandShakeRespSent
					peer.remote = raddr // Update remote endpoint
					peer.ready.Store(true)
					peer.mu.Unlock()
					continue
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
					continue
				}
			}

			// Handshake final response, process
			if h.SubType == header.Responder {
				if peer.state == HandshakeInitSent {
					_, peer.tx, peer.rx, err = peer.hs.ReadMessage(nil, in[header.Len:n])
					if err != nil {
						log.Printf("error reading handshake response: %v - resetting peer state", err)
						peer.ready.Store(false)
						peer.state = HandshakeNotStarted
						peer.hs = nil
						peer.rx = nil
						peer.tx = nil
						peer.mu.Unlock()
						continue
					}
					peer.state = HandshakeDone
					peer.remote = raddr
					peer.ready.Store(true)
					peer.mu.Unlock()
					continue
				}
			}
		} else {
			log.Printf("invalid message type: %s remote: %s", h.TypeName(), raddr.String())
		}
	}
}

//func (node *Node) handleInbound() {
//	in := make([]byte, 1400)
//	out := make([]byte, 1400)
//	h := &header.Header{}
//	fwpacket := &FWPacket{}
//	for {
//		n, raddr, err := node.conn.ReadFromUDP(in)
//		if err != nil {
//			log.Fatal(err)
//		}
//
//		err = h.Parse(in[:n])
//		if err != nil {
//			log.Fatal(err)
//		}
//
//		log.Printf("[%s] received %d bytes", raddr.String(), n)
//
//		if h.Type == header.Punch {
//			// Drop received punch packets
//			log.Printf("Received punch packets from %s", raddr.String())
//			continue
//		}
//
//		log.Printf("Looking up peer with ID %d", h.ID)
//		peer := node.peermap.ContainsRemoteID(h.ID)
//
//		if peer == nil && h.Type == header.Handshake {
//			// Peer trying to handshake, lets respond
//			if h.SubType == header.Initiator {
//				peer = &Peer{localID: GenerateID(), remoteID: h.ID, remote: raddr, ready: false, state: HandShakeRespSent}
//				peer.NewHandshake(false, node.keyPair)
//				// need to add methods or lock peer mutex for this stuff later
//				// Read handshake message and response
//				_, _, _, err = peer.hs.ReadMessage(nil, in[header.Len:n])
//				if err != nil {
//					log.Printf("error reading first handshake message: %v", err)
//					continue
//				}
//				// Respond to handshake
//				out, _ = h.Encode(out, header.Handshake, header.Responder, peer.localID, 2)
//				out, peer.rx, peer.tx, err = peer.hs.WriteMessage(out, nil)
//				if err != nil {
//					log.Printf("error writing handshake response: %v", err)
//					continue
//				}
//
//				n, err = node.conn.WriteToUDP(out, peer.remote)
//				if err != nil {
//					log.Fatal(err)
//				}
//
//				// Temporarily query for peer VPN IP
//				pid := base64.StdEncoding.EncodeToString(peer.hs.PeerStatic())
//				resp, _ := node.api.WhoIsID(context.TODO(), &msg.WhoIsIDRequest{Id: pid})
//
//				if err != nil {
//					log.Fatal(err)
//				}
//				log.Printf("wrote handshake response to peer - %d bytes", n)
//				peer.UpdateState(HandshakeDone)
//				peer.UpdateStatus(true)
//				peer.vpnip = netip.MustParseAddr(resp.Remote.VpnIp)
//				node.peermap.AddPeerWithIndices(peer)
//				continue
//			}
//
//			if h.SubType == header.Responder {
//				peer = node.peermap.ContainsRemoteID(h.ID)
//				if peer == nil {
//					// Search by remote IP in pending
//					peer = node.peermap.ContainsPendingRemote(raddr)
//					if peer == nil {
//						log.Println("no pending peer to complete handshake with")
//						continue
//					}
//				}
//				_, peer.tx, peer.rx, err = peer.hs.ReadMessage(nil, in[header.Len:n])
//				if err != nil {
//					log.Fatalf("error reading handshake response: %v", err)
//
//				}
//				peer.UpdateStatus(true)
//				peer.remoteID = h.ID
//				node.peermap.AddPeerWithIndices(peer)
//				node.peermap.DeletePendingPeer(peer)
//				log.Printf("handshake completed with peer %s : %s", peer.vpnip.String(), peer.remote.String())
//				continue
//			}
//		}
//
//		if h.Type == header.Data {
//			if !peer.isReady() {
//				log.Println("peer not ready...cant read regular data")
//				continue
//			}
//
//			err = h.Parse(in[:n])
//			peer.rx.SetNonce(h.MsgCounter)
//			data, err := peer.rx.Decrypt(nil, nil, in[header.Len:n])
//			if err != nil {
//				peer.UpdateStatus(false)
//				log.Printf("error decrypting data packet: %v", err)
//				continue
//			}
//
//			log.Println("Decrypted Data packet")
//
//			fwpacket, err = node.fw.Parse(data, true)
//			if err != nil {
//				log.Println(err)
//			}
//			if drop := node.fw.Drop(fwpacket); drop {
//				continue
//			}
//
//			node.tun.Write(data)
//		}
//
//	}
//}

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

		node.peermap.mu.RLock()
		peer, found := node.peermap.peerIP[remoteIP]
		node.peermap.mu.RUnlock()

		if !found {
			peer, err = node.QueryNewPeerIP(remoteIP)
			if err != nil {
				log.Println(err)
				continue
			}
		}

		peer.mu.Lock()

		// Peer was found, and is ready, send data
		if peer.ready.Load() {
			out, err = h.Encode(out, header.Data, header.None, node.localID, peer.tx.Nonce())
			if err != nil {
				if err == noise.ErrMaxNonce {
					log.Fatal(err)
				}
				log.Printf("error encoding header for data packet: %v", err)
				peer.ready.Store(false)
				peer.state = HandshakeNotStarted
				peer.hs = nil
				peer.rx = nil
				peer.tx = nil
				peer.mu.Unlock()
				continue
			}
			encrypted, err := peer.tx.Encrypt(out, nil, in[:n])
			if err != nil {
				log.Printf("error encryping data packet: %v", err)
				peer.ready.Store(false)
				peer.state = HandshakeNotStarted
				peer.hs = nil
				peer.rx = nil
				peer.tx = nil
				peer.mu.Unlock()
				continue
			}

			n, err = node.conn.WriteToUDP(encrypted, peer.remote)
			if err != nil {
				log.Fatal(err)
			}

			//log.Printf("Wrote %d bytes to peer %s", n, peer.remote.String())
			peer.mu.Unlock()
			continue
		}

		// If we get here, peer was found but not ready, check state and start handshake possibly

		if peer.state == HandshakeInitSent {
			log.Printf("already sent handshake - waiting for response for peer %d", peer.remoteID)
			peer.mu.Unlock()
			continue
		}
		// Punch
		_, err = node.api.Punch(context.TODO(), &msg.PunchRequest{SrcVpnIp: node.vpnip.String(), DstVpnIp: peer.vpnip.String()})
		if err != nil {
			log.Printf("error requesting punch before handshake: %v", err)
		}

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
			continue
		}
		_, err = node.conn.WriteToUDP(out, peer.remote)
		if err != nil {
			peer.ready.Store(false)
			peer.state = HandshakeNotStarted
			peer.hs = nil
			peer.rx = nil
			peer.tx = nil
			peer.mu.Unlock()
			continue
		}
		peer.state = HandshakeInitSent
		peer.mu.Unlock()
		continue

	}
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
			n, _ := node.conn.WriteToUDP(out, raddr)
			b += n
		}

		log.Printf("sent 5 punch packets to %s - %d bytes", req.Remote, b)
	}

}
