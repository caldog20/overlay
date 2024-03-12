package cmd

import (
	"context"
	"log"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/caldog20/overlay/node"
)

var (
	port       uint16
	controller string
	profile    bool
)

func NewRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "runs node service",
		Run: func(cmd *cobra.Command, args []string) {
			cpus := runtime.NumCPU()
			runtime.GOMAXPROCS(cpus)
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
				return localNode.Run(egCtx)
			})

			if profile {
				f, err := os.Create("node.pprof")
				if err != nil {
					log.Fatal(err)
				}
				pprof.StartCPUProfile(f)
				defer pprof.StopCPUProfile()
			}

			if err = eg.Wait(); err != nil {
				log.Fatal(err)
			}

		},
	}

	cmd.PersistentFlags().
		StringVar(&controller, "controller", "127.0.0.1:50000", "controller address in <ip:port> format")
	cmd.PersistentFlags().
		Uint16Var(&port, "port", 0, "listen port for udp socket - defaults to 0 for randomly selected port")
	cmd.PersistentFlags().BoolVar(&profile, "profile", false, "enable pprof profile")

	return cmd
}
