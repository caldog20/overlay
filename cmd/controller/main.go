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
	if err != nil {
		log.Fatal("error parsing config: ", err)
	}

	store, err := controller.NewSqlStore(config.DbPath)
	if err != nil {
		log.Fatal(err)
	}

	ctrl := controller.NewController(config, store)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		sigchan := make(chan os.Signal, 1)
		signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)
		log.Printf("Received %v signal, shutting down\n", <-sigchan)
		cancel()
	}()

	go controller.StartDiscoveryServer(ctx, config.DiscoveryPort)

	err = ctrl.Run(ctx)
	if err != nil {
		log.Fatal(err)
	}
}
