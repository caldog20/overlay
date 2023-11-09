package tun

import (
	"fmt"
	"log"
	"net"
	"os/exec"
	"runtime"

	"github.com/rcrowley/go-metrics"
	"github.com/songgao/water"
)

type TunMetrics struct {
	rx metrics.Counter
	tx metrics.Counter
}

type Tun struct {
	*water.Interface
	metrics TunMetrics
}

func (tun *Tun) Read(p []byte) (n int, err error) {
	n, err = tun.Interface.Read(p)
	if err == nil {
		tun.metrics.rx.Inc(int64(n))
	}
	return n, err
}

func (tun *Tun) Write(p []byte) (n int, err error) {
	n, err = tun.Interface.Write(p)
	if err == nil {
		tun.metrics.tx.Inc(int64(n))
	}
	return n, err
}

func NewTun() (*Tun, error) {
	ifce, err := water.New(water.Config{DeviceType: water.TUN})
	if err != nil {
		return nil, err
	}

	tun := &Tun{
		ifce,
		TunMetrics{
			metrics.GetOrRegisterCounter("tun.rx.bytes", nil),
			metrics.GetOrRegisterCounter("tun.tx.bytes", nil),
		},
	}

	return tun, nil
}

// TODO: Move route to route handler
func (tun *Tun) ConfigureInterface(ip net.IP) error {
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
		if err := exec.Command("/sbin/ip", "route", "add", "192.168.1.0/24", "via", ip.String()).Run(); err != nil {
			log.Fatalf("route add error: %v", err)
		}
	case "darwin":
		if err := exec.Command("/sbin/ifconfig", tun.Name(), "mtu", "1300", ip.String(), ip.String(), "up").Run(); err != nil {
			return fmt.Errorf("ifconfig error %v: %w", tun.Name(), err)
		}
		if err := exec.Command("/sbin/route", "-n", "add", "-net", "192.168.1.0/24", ip.String()).Run(); err != nil {
			log.Fatalf("route add error: %v", err)
		}
	default:
		return fmt.Errorf("no tun support for: %v", runtime.GOOS)
	}

	log.Printf("set tunnel IP successful: %v %v", tun.Name(), ip.String()+"/32")
	log.Printf("set route successful: %v via %v dev %v", "192.168.1.0/24", ip.String(), tun.Name())
	return nil
}

func (tun *Tun) PrintMetrics() {
	fmt.Printf("tunnel rx bytes: %v\n", tun.metrics.rx.Count())
	fmt.Printf("tunnel tx bytes: %v\n", tun.metrics.tx.Count())
}
