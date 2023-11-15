package node

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/caldog20/go-overlay/header"
	"github.com/flynn/noise"
	"gopkg.in/ini.v1"
	"log"
	"net"
	"net/netip"
)

type Node struct {
	conn    *net.UDPConn
	peermap *PeerMap
	keyPair noise.DHKey
	sender  bool
}

func NewNode(config *ini.File) *Node {
	var kp noise.DHKey
	localSection := config.Section("Local")
	port := localSection.Key("Port").String()
	kp.Private, _ = base64.StdEncoding.DecodeString(localSection.Key("PrivateKey").String())
	kp.Public, _ = base64.StdEncoding.DecodeString(localSection.Key("PublicKey").String())
	//id, _ := localSection.Key("ID").Uint()

	peerSection := config.Section("Peer")
	peerStatic, _ := base64.StdEncoding.DecodeString(peerSection.Key("PublicKey").String())
	peerRemote := peerSection.Key("Remote").String()
	paddr, _ := net.ResolveUDPAddr("udp4", peerRemote)

	p := &Peer{
		remote:  paddr,
		rs:      peerStatic,
		localID: 100,
		ready:   false,
	}

	pmap := NewPeerMap()
	pmap.peers[netip.MustParseAddr("192.168.1.1")] = p

	laddr, err := net.ResolveUDPAddr("udp4", ":"+port)
	if err != nil {
		log.Fatal(err)
	}

	c, err := net.ListenUDP("udp4", laddr)
	if err != nil {
		log.Fatal(err)
	}

	n := &Node{
		conn:    c,
		peermap: pmap,
		keyPair: kp,
		sender:  port == "5555",
	}

	return n
}

func (node *Node) handleInbound() {
	in := make([]byte, 1300)
	out := make([]byte, 1300)
	h := &header.Header{}

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

		log.Printf("Looking up peer with ID %d", h.ID)
		p := node.peermap.ContainsRemoteID(h.ID)

		if p == nil {
			// peer not found, check if handshake messae
			// handle elsewhere
			log.Println("peer is nil")
			if h.Type == header.Handshake {
				if h.SubType == header.Initiator {
					// Host is trying to handshake with us, lets do it
					p = &Peer{localID: 200, remoteID: h.ID, remote: raddr, ready: false}
					p.NewHandshake(false, node.keyPair)
					// need to add methods or lock peer mutex for this stuff later
					// Read handshake message and response
					_, _, _, err = p.hs.ReadMessage(nil, in[header.Len:n])
					if err != nil {
						log.Printf("error reading first handshake message: %v", err)
						continue
					}

					out, _ = h.Encode(out, header.Handshake, header.Responder, p.localID, 2)
					// handshake response gets appended to slice above with header at beginning
					out, rx, tx, err := p.hs.WriteMessage(out, nil)
					if err != nil {
						log.Printf("error writing handshake response: %v", err)
						continue
					}
					log.Printf("PEER STATIC OBTAINED FROM HANDSHAKE: %s", base64.StdEncoding.EncodeToString(p.hs.PeerStatic()))
					p.rx = rx
					p.tx = tx
					n, _ = node.conn.WriteToUDP(out, p.remote)
					log.Printf("wrote handshake response to peer - %d bytes", n)
					p.ready = true
					p.vpnip = netip.MustParseAddr("192.168.1.2")
					node.peermap.AddPeerWithIndices(p)
				}
			}
		} else {
			if !p.ready {
				log.Println("peer not ready...cant read regular data")
				continue
			}

			h.Parse(in[:n])
			// lookup peer
			if p == nil {
				log.Println("cant find peer in peermap by remote index")
				continue
			}

			log.Println("found peer")

			data, err := p.rx.Decrypt(nil, nil, in[header.Len:n])
			if err != nil {
				log.Printf("error decrypting data packet: %v", err)
			}

			fmt.Printf("PAYLOAD: %s\n", string(data))

		}

	}
}

func (node *Node) handleOutbound() {
	in := make([]byte, 1300)
	out := make([]byte, 1300)
	h := &header.Header{}

	// preset peer here since we are testing and know what peer we want to send to

	p := node.peermap.Contains(netip.MustParseAddr("192.168.1.1"))
	if p == nil {
		log.Fatal("cant find predetermined peer")
	}

	p.NewHandshake(true, node.keyPair)

	out, _ = h.Encode(out, header.Handshake, header.Initiator, p.localID, 1)
	log.Println("writing first handshake message")
	var err error
	out, _, _, err = p.hs.WriteMessage(out, nil)
	if err != nil {
		log.Printf("error writing handshake initiating message: %v", err)
		return
	}

	n, _ := node.conn.WriteToUDP(out, p.remote)

	n, err = node.conn.Read(in)
	h.Parse(in[:n])

	if h.Type == header.Handshake && h.SubType == header.Responder {
		_, tx, rx, err := p.hs.ReadMessage(nil, in[header.Len:n])
		if err != nil {
			log.Printf("error reading handshake response: %v", err)
			return
		}
		p.tx = tx
		p.rx = rx
		p.ready = true
		p.remoteID = h.ID
		node.peermap.AddPeerWithIndices(p)
	}

	out, _ = h.Encode(out, header.Data, header.None, p.localID, 3)
	t := []byte("Encrypted Channel Working!!!")
	out, err = p.tx.Encrypt(out, nil, t)
	if err != nil {
		log.Printf("error encrypting regular data packet: %v", err)
		return
	}

	n, _ = node.conn.WriteToUDP(out, p.remote)
	log.Printf("wrote %d bytes to %s", n, p.remote.String())
}

func (node *Node) Run(ctx context.Context) {
	if node.sender {
		node.handleOutbound()
	} else {
		node.handleInbound()
	}
	<-ctx.Done()
}
