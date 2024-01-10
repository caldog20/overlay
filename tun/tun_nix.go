//go:build darwin || linux || freebsd || netbsd

package tun

import (
	"github.com/songgao/water"
)

// Currently, this is used for Mac/Linux Tunnels
type NixTun struct {
	ifce *water.Interface
}

func NewTun() (Tun, error) {
	ifce, err := water.New(water.Config{DeviceType: water.TUN})
	if err != nil {
		return nil, err
	}

	return &NixTun{ifce: ifce}, nil
}

func (n NixTun) Read(b []byte) (int, error) {
	return n.ifce.Read(b)
}

func (n NixTun) Write(b []byte) (int, error) {
	return n.ifce.Write(b)
}

func (n NixTun) Name() (string, error) {
	return n.ifce.Name(), nil
}

func (n NixTun) Close() error {
	return n.ifce.Close()
}

func (n NixTun) MTU() (int, error) {
	return MTU, nil
}
