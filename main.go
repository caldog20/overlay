package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/caldog20/go-overlay/controller"
	"github.com/caldog20/go-overlay/testclient"
)

func main() {

	con := flag.Bool("controller", true, "Enable Controller")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		sigchan := make(chan os.Signal, 1)
		signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)
		log.Printf("Received %v signal from channel, shutting down\n", <-sigchan)
		//time.Sleep(3 * time.Second)
		//fmt.Print("Shutting down context")
		cancel()
	}()

	if *con {
		go controller.Run(ctx)
	}

	testclient.Run(ctx)

}
