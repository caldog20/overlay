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
	//config, err := controller.GetConfig()
	//if err != nil {
	//	log.Fatal(err)
	//}

	store, err := controller.NewSqlStore("data.db")
	if err != nil {
		log.Fatal(err)
	}

	ctrl := controller.NewController(store)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		sigchan := make(chan os.Signal, 1)
		signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)
		log.Printf("Received %v signal, shutting down\n", <-sigchan)
		cancel()
	}()

	go controller.StartDiscoveryServer(ctx)

	err = ctrl.Run(ctx)
	if err != nil {
		log.Fatal(err)
	}
}
