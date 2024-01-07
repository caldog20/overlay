package node

import (
	"fmt"
	"log"
	"net/netip"
	"os/exec"
	"runtime"

	"github.com/rcrowley/go-metrics"
	"github.com/songgao/water"
)

type TunMetrics struct {
	rx metrics.Counter
	tx metrics.Counter
}

// TODO don't anonymously embed this
// TODO Break out into interface to support Wintun and unix/linux tun
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

type OnTunnelPacket func(buffer *OutboundBuffer)

func (tun *Tun) ReadPackets(callback OnTunnelPacket) {
	for {
		buffer := GetOutboundBuffer()
		n, err := tun.Read(buffer.packet)
		if err != nil {
			PutOutboundBuffer(buffer)
			log.Println(err)
			continue
		}

		buffer.size = n
		callback(buffer)
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
		if err := exec.Command("/sbin/ifconfig", tun.Name(), "mtu", "1400", ip.String(), ip.String(), "up").Run(); err != nil {
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
