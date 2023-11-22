package node

import (
	"context"
	"fmt"
	"github.com/caldog20/go-overlay/header"
	"golang.org/x/sys/unix"
	"log"
	"net"
	"syscall"
)

const (
	Net = "udp4"
)

type Conn struct {
	uc *net.UDPConn
}

func NewConn(port uint16) *Conn {
	lc := net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			var opErr error
			err := c.Control(func(fd uintptr) {
				opErr = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)
			})
			if err != nil {
				return err
			}
			return opErr
		},
	}

	lp, err := lc.ListenPacket(context.Background(), Net, fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		log.Fatal(err)
	}

	udpconn, ok := lp.(*net.UDPConn)
	if !ok {
		log.Fatal("error casting ListenPacket into UDP Conn")
	}

	conn := &Conn{
		uc: udpconn,
	}

	return conn
}

type udpcallback func(raddr *net.UDPAddr, in []byte, out []byte, h *header.Header, fwpacket *FWPacket, index int)

func (conn *Conn) ReadPackets(callback udpcallback, index int) {
	h := &header.Header{}
	fwpacket := &FWPacket{}
	in := make([]byte, 1400)
	out := make([]byte, 1400)

	for {
		n, raddr, err := conn.uc.ReadFromUDP(in)
		if err != nil {
			log.Println(err)
			conn.uc.Close()
			return
		}
		callback(raddr, in[:n], out, h, fwpacket, index)
	}
}

func (conn *Conn) GetLocalAddr() net.Addr {
	return conn.uc.LocalAddr()
}
