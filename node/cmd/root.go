package cmd

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/caldog20/overlay/node"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var (
	port       uint16
	controller string

	rootCmd = &cobra.Command{
		Use:   "node",
		Short: "Overlay Node",
		Long:  "",
		Run: func(cmd *cobra.Command, args []string) {

			ctx, cancel := context.WithCancel(context.Background())
			go func() {
				sigchan := make(chan os.Signal, 1)
				signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)
				log.Printf("Received %v signal, shutting down\n", <-sigchan)
				cancel()
			}()

			eg, egCtx := errgroup.WithContext(ctx)

			localNode, err := node.NewNode(port, controller)
			if err != nil {
				log.Fatal(err)
			}

			eg.Go(func() error {
				// TODO: Change to return error
				return localNode.Run(egCtx)
			})

			if err = eg.Wait(); err != nil {
				log.Fatal(err)
			}

		},
	}
)

func init() {
	rootCmd.PersistentFlags().StringVar(&controller, "controller", "127.0.0.1:50000", "controller address in <ip:port> format")
	rootCmd.PersistentFlags().Uint16Var(&port, "port", 0, "listen port for udp socket - defaults to 0 for randomly selected port")
}

// TODO handle signals and contextual things here
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
