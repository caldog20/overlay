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

	controller := flag.String("controller", "", "controller address in <http://hostname or ip:port>")

	flag.Parse()

	if *controller == "" {
		log.Fatal("controller argument must not be nil")
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		sigchan := make(chan os.Signal, 1)
		signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)
		log.Printf("Received %v signal from channel, shutting down\n", <-sigchan)
		cancel()
	}()

	//go node.ReportBuffers()

	localNode, err := node.NewNode("5555", *controller)
	if err != nil {
		log.Fatal(err)
	}

	localNode.Run(ctx)
}
