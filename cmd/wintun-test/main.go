package main

import (
	"log"
	"net/netip"

	"github.com/caldog20/overlay/tun"
)

func main() {
	tun, err := tun.NewTun()
	defer tun.Close()

	if err != nil {
		log.Fatal(err)
	}

	err = tun.ConfigureIPAddress(netip.MustParsePrefix("100.70.0.10/24"))
	if err != nil {
		log.Fatal(err)
	}

	for {
	}
	//buf := make([]byte, 1400)
	//for {
	//	n, err := tun.Read(buf)
	//	if err != nil {
	//		log.Fatal(err)
	//	}
	//
	//	fmt.Printf("read packet: %d bytes\n", n)
	//	fmt.Printf("%s", string(buf[:n]))
	//}

}
