package main

import (
	"context"
	"flag"
	"gopkg.in/ini.v1"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/caldog20/go-overlay/node"
)

func main() {
	pa := flag.String("c", "/Users/cyates/projects/go-overlay/cmd/node/config.ini", "")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		sigchan := make(chan os.Signal, 1)
		signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)
		log.Printf("Received %v signal from channel, shutting down\n", <-sigchan)
		cancel()
	}()

	config, err := ini.Load(*pa)
	if err != nil {
		log.Fatal(err)
	}

	n := node.NewNode(config)
	n.Run(ctx)
}
