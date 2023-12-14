package main

import (
	"context"
	"flag"
	"github.com/caldog20/go-overlay/node"
	"log"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

func main() {
	//f, _ := os.Create("profile.prof")
	//pprof.StartCPUProfile(f)
	cpus := runtime.NumCPU()
	runtime.GOMAXPROCS(cpus)

	node_id := flag.Uint("id", 0, "id for node - unique per node")
	port := flag.Uint("port", 0, "port for listen udp")
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

	n := node.NewNode(uint32(*node_id), uint16(*port))
	n.Run(ctx)
	//pprof.StopCPUProfile()
}
