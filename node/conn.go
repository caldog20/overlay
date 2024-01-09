package node

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/netip"
	"syscall"

	"golang.org/x/sys/unix"
)

const (
	UDPType = "udp4"
)

// Maybe make interface or add more methods for functionality

type Conn struct {
	uc *net.UDPConn
}

func NewConn(port uint16) (*Conn, error) {
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

	lp, err := lc.ListenPacket(context.Background(), UDPType, fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		return nil, err
	}

	udpconn, ok := lp.(*net.UDPConn)
	if !ok {
		return nil, errors.New("error casting ListenPacket into UDP Conn")
	}

	conn := &Conn{
		uc: udpconn,
	}

	return conn, nil
}

type OnUDPPacket func(buffer *InboundBuffer, index int)

func (conn *Conn) ReadPackets(callback OnUDPPacket, index int) {
	for {
		buffer := GetInboundBuffer()
		n, raddr, err := conn.ReadFromUDP(buffer.in)
		if err != nil {
			PutInboundBuffer(buffer)
			log.Println(err)
			continue
		}

		buffer.size = n
		buffer.raddr = raddr
		callback(buffer, index)
	}
}

func (conn *Conn) WriteToUDP(b []byte, addr *net.UDPAddr) (int, error) {
	n, err := conn.uc.WriteToUDP(b, addr)
	return n, err
}

func (conn *Conn) ReadFromUDP(b []byte) (int, *net.UDPAddr, error) {
	n, raddr, err := conn.uc.ReadFromUDP(b)
	return n, raddr, err
}

func (conn *Conn) GetLocalAddr() net.Addr {
	return conn.uc.LocalAddr()
}

func (conn *Conn) Close() error {
	err := conn.uc.Close()
	return err
}

func GetPreferredOutboundAddr() (netip.Addr, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return netip.Addr{}, err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	parsedAddr, err := netip.ParseAddr(localAddr.IP.String())
	if err != nil {
		return netip.Addr{}, err
	}

	return parsedAddr, nil
}
