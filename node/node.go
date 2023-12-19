package node

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/caldog20/go-overlay/proto"
	"log"
	"net"
	"net/http"
	"net/netip"
	"os"
	"strings"
	"sync"

	"github.com/flynn/noise"
)

type Node struct {
	conn *net.UDPConn
	id   uint32
	ip   netip.Addr

	maps struct {
		l  sync.RWMutex
		id map[uint32]*Peer     // for RX
		ip map[netip.Addr]*Peer // for TX
	}

	noise struct {
		l       sync.RWMutex
		cipher  noise.CipherSuite
		keyPair noise.DHKey
	}

	controller proto.Controller
}

type Key struct {
	Public  string
	Private string
}

func NewNode(port string, controller string) (*Node, error) {
	node := new(Node)
	node.maps.id = make(map[uint32]*Peer)
	node.maps.ip = make(map[netip.Addr]*Peer)

	node.noise.cipher = noise.NewCipherSuite(noise.DH25519, noise.CipherChaChaPoly, noise.HashBLAKE2s)

	// Try to load key from disk
	keypair, err := LoadKeyFromDisk()
	if err != nil {
		keypair, err = node.noise.cipher.GenerateKeypair(nil)
		err = StoreKeyToDisk(keypair)
		if err != nil {
			log.Fatal("error storing keypair to disk")
		}
	}

	node.noise.keyPair = keypair

	laddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf(":%s", port))
	if err != nil {
		return nil, err
	}

	node.conn, err = net.ListenUDP("udp4", laddr)
	if err != nil {
		return nil, err
	}

	node.controller = proto.NewControllerProtobufClient(controller, &http.Client{})

	return node, nil
}

func (node *Node) Run(ctx context.Context) {
	err := node.Register()
	if err != nil {
		log.Fatal(err)
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
