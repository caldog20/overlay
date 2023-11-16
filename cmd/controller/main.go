package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/caldog20/go-overlay/controller"
)

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		sigchan := make(chan os.Signal, 1)
		signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)
		log.Printf("Received %v signal from channel, shutting down\n", <-sigchan)
		cancel()
	}()

	controller.RunController(ctx)
}
