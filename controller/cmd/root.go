package cmd

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/caldog20/overlay/controller"
	"github.com/caldog20/overlay/controller/store"
	"github.com/spf13/cobra"
)

var (
	storePath string
	autotls   bool

	rootCmd = &cobra.Command{
		Use:   "controller",
		Short: "ZeroMesh Overlay Controller",
		Long:  "",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithCancel(context.Background())
			go func() {
				sigchan := make(chan os.Signal, 1)
				signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)
				log.Printf("Received %v signal, shutting down\n", <-sigchan)
				cancel()
			}()

			// TODO Implement config stuff/multiple commands

			store, err := store.NewSqlStore(storePath)
			if err != nil {
				log.Fatal(err)
			}

			ctrl := controller.NewController(store)
			log.Fatal(ctrl.Run(ctx))
		},
	}
)

func init() {
	rootCmd.PersistentFlags().StringVar(&storePath, "storepath", "store.db", "file path for controller store persistence")
	rootCmd.PersistentFlags().BoolVar(&autotls, "autotls", false, "enable autotls for controller")
}

// TODO handle signals and contextual things here
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
