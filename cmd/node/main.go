package main

import (
	"context"
	"flag"
	"github.com/caldog20/go-overlay/node"
	"log"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	//f, _ := os.Create("profile.prof")
	//pprof.StartCPUProfile(f)
	node_id := flag.Uint("id", 0, "id for node - unique per node")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		sigchan := make(chan os.Signal, 1)
		signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)
		log.Printf("Received %v signal from channel, shutting down\n", <-sigchan)
		cancel()
	}()

	//go func() {
	//	log.Println(http.ListenAndServe("localhost:6060", nil))
	//}()

	n := node.NewNode(uint32(*node_id))
	n.Run(ctx)
	//pprof.StopCPUProfile()
}
