package main

import (
	"context"
	"crypto/rand"
	"github.com/flynn/noise"
	"log"
	"net"
	"time"
)

type Conn struct {
	conn *net.UDPConn

	in, out   *noise.CipherState
	handshake *noise.HandshakeState
	address   string
	authed    bool
	initiator bool
	key       noise.DHKey
	peerKey   []byte
}

//rng := rand.Reader
//static, _ := cs.GenerateKeypair(rng)

func NewConn(address string, initiator bool, key noise.DHKey, peerKey []byte) *Conn {
	pkey := make([]byte, 32)
	copy(pkey, peerKey)

	c := &Conn{
		handshake: nil,
		address:   address,
		authed:    false,
		initiator: initiator,
		key:       key,
		peerKey:   pkey,
	}

	return c
}

func (c *Conn) Listen() {
	list, _ := net.ListenPacket("udp", c.address)
	c.conn = list.(*net.UDPConn)

	in := make([]byte, 1300)

	for {

		if !c.authed {
			rng := rand.Reader
			cs := noise.NewCipherSuite(noise.DH25519, noise.CipherAESGCM, noise.HashSHA256)
			c.handshake, _ = noise.NewHandshakeState(noise.Config{
				CipherSuite:   cs,
				Random:        rng,
				Pattern:       noise.HandshakeKK,
				Initiator:     c.initiator,
				StaticKeypair: c.key,
				PeerStatic:    c.peerKey,
			})

			log.Println("------HANDSHAKING---------")
			n, remote, _ := c.conn.ReadFromUDP(in)
			_, _, _, err := c.handshake.ReadMessage(nil, in[:n])
			if err != nil {
				log.Print("server")
				log.Fatal(err)
			}
			var out []byte
			out, c.in, c.out, err = c.handshake.WriteMessage(nil, nil)
			if err != nil {
				log.Print("server")
				log.Fatal(err)
			}
			c.conn.WriteToUDP(out, remote)
			c.authed = true
			c.handshake = nil

		} else {
			//os.Exit(1)
			msg := make([]byte, 1300)
			n, err := c.conn.Read(in)
			if err != nil {
				log.Fatal(err)
			}

			msg, err = c.in.Decrypt(msg, nil, in[:n])
			if err != nil {
				log.Printf("server failed %v bytes", n)
				log.Fatal(err)
				c.authed = false
			}

			//log.Println(string(msg))

			//os.Exit(1)
		}
	}
}

func (c *Conn) Dial() {
	conn, _ := net.Dial("udp", c.address)

	in := make([]byte, 1300)
	//out := make([]byte, 1300)

	for {
		if !c.authed {
			rng := rand.Reader
			cs := noise.NewCipherSuite(noise.DH25519, noise.CipherAESGCM, noise.HashSHA256)
			c.handshake, _ = noise.NewHandshakeState(noise.Config{
				CipherSuite:   cs,
				Random:        rng,
				Pattern:       noise.HandshakeKK,
				Initiator:     c.initiator,
				StaticKeypair: c.key,
				PeerStatic:    c.peerKey,
			})

			log.Println("------CLIENT HANDSHAKING---------")
			out, _, _, err := c.handshake.WriteMessage(nil, nil)
			if err != nil {
				log.Print("client")
				log.Fatal(err)
			}

			_, _ = conn.Write(out)

			n, _ := conn.Read(in)

			in, c.out, c.in, err = c.handshake.ReadMessage(nil, in[:n])
			if err != nil {
				log.Print("client")
				log.Fatal(err)
			}
			c.handshake = nil
			c.authed = true
		} else {
			msg := []byte("HelloHello")
			out, err := c.out.Encrypt(nil, nil, msg)
			if err != nil {
				log.Print("client")
				c.authed = false
				log.Fatal(err)
			}

			write, err := conn.Write(out[:])
			if err != nil {
				log.Println(err)
			}
			_ = write
			time.Sleep(time.Second * 1)
		}
	}

}

func main() {
	ctx := context.Background()

	rng := rand.Reader
	cs := noise.NewCipherSuite(noise.DH25519, noise.CipherAESGCM, noise.HashSHA256)
	staticC, _ := cs.GenerateKeypair(rng)
	staticS, _ := cs.GenerateKeypair(rng)

	s := NewConn(":5554", false, staticS, staticC.Public)
	c := NewConn("127.0.01:5554", true, staticC, staticS.Public)

	go func() {
		s.Listen()
	}()

	go func() {
		c.Dial()
	}()

	<-ctx.Done()

	//cs := noise.NewCipherSuite(noise.DH25519, noise.CipherAESGCM, noise.HashSHA256)
	//rng := rand.Reader
	//staticC, _ := cs.GenerateKeypair(rng)
	//staticS, _ := cs.GenerateKeypair(rng)
	//sHS, _ := noise.NewHandshakeState(noise.Config{
	//	CipherSuite:   cs,
	//	Random:        rng,
	//	Pattern:       noise.HandshakeKK,
	//	Initiator:     true,
	//	PeerStatic:    staticC.Public,
	//	StaticKeypair: staticS,
	//})
	//
	//cHS, _ := noise.NewHandshakeState(noise.Config{
	//	CipherSuite:   cs,
	//	Random:        rng,
	//	Pattern:       noise.HandshakeKK,
	//	Initiator:     false,
	//	PeerStatic:    staticS.Public,
	//	StaticKeypair: staticC,
	//})
	//
	//msg, _, _, _ := sHS.WriteMessage(nil, []byte("shs to chs 1"))
	//
	//res, _, _, err := cHS.ReadMessage(nil, msg)
	//log.Println(err)
	//
	//msg, crx, ctx, _ := cHS.WriteMessage(nil, []byte("chs to shs 1"))
	//
	//res, stx, srx, err := sHS.ReadMessage(nil, msg)
	//log.Println(err)
	//
	//_ = res
	//
	//msg1 := []byte("hello server")
	//msg2 := []byte("hello client")
	////ad := make([]byte, 16)
	////stx, srx, err :=
	////out := make([]byte, 1300)
	////in := make([]byte, 1300)
	//
	//out, err := ctx.Encrypt(nil, nil, msg1)
	//log.Println(string(out))
	//in, err := srx.Decrypt(nil, nil, out)
	//log.Println(string(in))
	//
	//out, err = stx.Encrypt(nil, nil, msg2)
	//log.Println(string(out))
	//in, err = crx.Decrypt(nil, nil, out)
	//log.Println(string(in))
}
