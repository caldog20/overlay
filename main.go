package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {

	con := flag.Bool("controller", false, "Enable Controller")
	caddr := flag.String("caddr", "localhost:5555", "")
	punch := flag.Bool("punch", false, "")

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
		RunController(ctx)
	} else {
		RunClient(ctx, *caddr, *punch)
	}

}
