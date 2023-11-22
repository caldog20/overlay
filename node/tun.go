package node

import (
	"fmt"
	"github.com/caldog20/go-overlay/header"
	"github.com/rcrowley/go-metrics"
	"github.com/songgao/water"
	"log"
	"net/netip"
	"os/exec"
	"runtime"
)

type TunMetrics struct {
	rx metrics.Counter
	tx metrics.Counter
}

type Tun struct {
	*water.Interface
}

func NewTun() (*Tun, error) {
	ifce, err := water.New(water.Config{DeviceType: water.TUN})
	if err != nil {
		return nil, err
	}

	tun := &Tun{
		ifce,
	}

	return tun, nil
}

type tuncallback func(in []byte, out []byte, h *header.Header, fwpacket *FWPacket, index int)

func (t *Tun) ReadTunPackets(callback tuncallback, index int) {
	in := make([]byte, 1400)
	out := make([]byte, 1400)
	h := &header.Header{}
	fwpacket := &FWPacket{}
	for {
		n, err := t.Read(in)
		if err != nil {
			log.Println(err)
			t.Close()
			return
		}
		callback(in[:n], out, h, fwpacket, index)
	}
}

// TODO: Move route to route handler
func (tun *Tun) ConfigureInterface(ip netip.Addr) error {
	net, _ := ip.Prefix(24)
	switch runtime.GOOS {
	case "linux":
		if err := exec.Command("/sbin/ip", "link", "set", "dev", tun.Name(), "mtu", "1300").Run(); err != nil {
			return fmt.Errorf("ip link error: %w", err)
		}
		if err := exec.Command("/sbin/ip", "addr", "add", ip.String()+"/32", "dev", tun.Name()).Run(); err != nil {
			return fmt.Errorf("ip addr error: %w", err)
		}
		if err := exec.Command("/sbin/ip", "link", "set", "dev", tun.Name(), "up").Run(); err != nil {
			return fmt.Errorf("ip link error: %w", err)
		}
		if err := exec.Command("/sbin/ip", "route", "add", net.String(), "via", ip.String()).Run(); err != nil {
			log.Fatalf("route add error: %v", err)
		}
	case "darwin":
		if err := exec.Command("/sbin/ifconfig", tun.Name(), "mtu", "1300", ip.String(), ip.String(), "up").Run(); err != nil {
			return fmt.Errorf("ifconfig error %v: %w", tun.Name(), err)
		}
		if err := exec.Command("/sbin/route", "-n", "add", "-net", net.String(), ip.String()).Run(); err != nil {
			log.Fatalf("route add error: %v", err)
		}
	default:
		return fmt.Errorf("no tun support for: %v", runtime.GOOS)
	}

	log.Printf("set tunnel IP successful: %v %v", tun.Name(), ip.String()+"/32")
	log.Printf("set route successful: %v via %v dev %v", net.String(), ip.String(), tun.Name())
	return nil
}
