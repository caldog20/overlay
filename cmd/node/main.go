package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/caldog20/overlay/node"
)

func httpProfile() {

	go func() {
		http.ListenAndServe(":1234", nil)
	}()

}

func main() {
	cpus := runtime.NumCPU()
	runtime.GOMAXPROCS(cpus)
	//f, err := os.Create("p.prof")
	//if err != nil {
	//
	//	fmt.Println(err)
	//	return
	//
	//}

	//pprof.StartCPUProfile(f)
	//defer pprof.StopCPUProfile()

	controller := flag.String("controller", "10.170.241.66:9000", "controller address in <http://hostname or ip:port>")
	port := flag.Uint("port", 0, "listen port for udp socket - defaults to 0 for randomly selected port")
	flag.Parse()

	if *controller == "" {
		log.Fatal("controller argument must not be nil")
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		sigchan := make(chan os.Signal, 1)
		signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)
		log.Printf("Received %v signal, shutting down\n", <-sigchan)
		cancel()
	}()

	//go node.ReportBuffers()

	localNode, err := node.NewNode(uint16(*port), *controller)
	if err != nil {
		log.Fatal(err)
	}

	localNode.Run(ctx)
}
