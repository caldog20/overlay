package wintun_test

import (
	"fmt"
	"log"

	"github.com/caldog20/overlay/tun"
)

func main() {
	tun, err := tun.NewTun()
	defer tun.Close()

	if err != nil {
		log.Fatal(err)
	}

	buf := make([]byte, 1400)
	for {
		n, err := tun.Read(buf)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("read packet: %d bytes\n", n)
		fmt.Printf("%s", string(buf[:n]))
	}

}
