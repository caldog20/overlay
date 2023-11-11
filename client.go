package main

import (
	"github.com/caldog20/go-overlay/msg"
	"net"
)

type Client struct {
	Hostname    string
	VpnIP       string
	Remote      string
	Addr        *net.UDPAddr
	PunchStream msg.ControlService_PunchNotifierServer
	Finished    chan<- bool
}
