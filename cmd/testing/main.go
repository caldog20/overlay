package main

import (
	"fmt"
	"log"
	"net"
	"net/netip"
)

func main() {

	addr, _ := net.InterfaceAddrs()
	for _, a := range addr {
		//fmt.Println(a.Network())
		p := netip.MustParsePrefix(a.String())
		ip := p.Addr()
		if ip.Is4() {
			fmt.Println(p.String(), ip.String())
		}
	}

	fmt.Println("Preferred Outbound: ", GetOutboundIP().String())
}

// Get preferred outbound ip of this machine
func GetOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}
