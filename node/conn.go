package node

import (
	"context"
	"fmt"
	"golang.org/x/sys/unix"
	"log"
	"net"
	"syscall"
)

const (
	Net = "udp4"
)

// Maybe make interface or add more methods for functionality

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

type udpcallback func(elem *Buffer, index int)

func (conn *Conn) ReadPackets(callback udpcallback, index int) {
	for {
		elem := GetBuffer()
		n, raddr, err := conn.uc.ReadFromUDP(elem.in)
		if err != nil {
			log.Println(err)
			PutBuffer(elem)
			conn.uc.Close()
			return
		}
		elem.size = n
		elem.raddr = raddr
		callback(elem, index)
	}
}

func (conn *Conn) GetLocalAddr() net.Addr {
	return conn.uc.LocalAddr()
}
