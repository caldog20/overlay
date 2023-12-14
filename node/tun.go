package node

import (
	"fmt"
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

type tuncallback func(elem *Buffer, index int)

func (t *Tun) ReadPackets(callback tuncallback, index int) {
	for {
		elem := GetBuffer()
		n, err := t.Read(elem.in)
		if err != nil {
			log.Println(err)
			t.Close()
			PutBuffer(elem)
			return
		}
		elem.size = n
		callback(elem, index)
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
