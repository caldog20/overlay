package main

import (
	"context"
	"github.com/caldog20/overlay/node"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

func httpProfile() {

	go func() {
		http.ListenAndServe(":1234", nil)
	}()

}

func main() {
	cpus := runtime.NumCPU()
	runtime.GOMAXPROCS(cpus)

	httpProfile()
	//f, err := os.Create("p.prof")
	//if err != nil {
	//
	//	fmt.Println(err)
	//	return
	//
	//}

	//pprof.StartCPUProfile(f)
	//defer pprof.StopCPUProfile()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		sigchan := make(chan os.Signal, 1)
		signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)
		log.Printf("Received %v signal from channel, shutting down\n", <-sigchan)
		cancel()
	}()

	node, err := node.NewNode("5555", "http://10.170.241.1:8080")
	if err != nil {
		log.Fatal(err)
	}

	node.Run(ctx)
}
