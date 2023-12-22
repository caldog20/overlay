package main

import (
	"context"
	"flag"

	"github.com/caldog20/overlay/controller"
)

func main() {
	port := flag.String("port", "8080", "port for http server")
	flag.Parse()
	//ctx, cancel := context.WithCancel(context.Background())
	//go func() {
	//	sigchan := make(chan os.Signal, 1)
	//	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)
	//	log.Printf("Received %v signal from channel, shutting down\n", <-sigchan)
	//	cancel()
	//}()

	c := controller.NewController()
	c.RunController(context.TODO(), *port)
}
