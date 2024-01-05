package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/caldog20/overlay/controller"
)

func main() {
	config, err := controller.GetConfig()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		sigchan := make(chan os.Signal, 1)
		signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)
		log.Printf("Received %v signal, shutting down\n", <-sigchan)
		cancel()
	}()

	c, err := controller.NewController(config)
	if err != nil {
		log.Fatal(err)
	}

	c.RunController(ctx)
}
