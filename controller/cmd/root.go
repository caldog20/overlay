package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/caldog20/overlay/controller"
	"github.com/caldog20/overlay/controller/discovery"
	"github.com/caldog20/overlay/proto/gen/api/v1/apiv1connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.org/x/sync/errgroup"

	"github.com/caldog20/overlay/controller/store"
	"github.com/spf13/cobra"
)

var (
	storePath string
	autoCert  bool

	rootCmd = &cobra.Command{
		Use:   "controller",
		Short: "Overlay Controller",
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
			err = ctrl.CreateAdminUser()
			if err != nil {
				log.Println(err)
			}

			eg, egCtx := errgroup.WithContext(ctx)

			mux := http.NewServeMux()
			apiv1Path, apiv1Handler := apiv1connect.NewControllerServiceHandler(ctrl)
			mux.Handle(apiv1Path, apiv1Handler)

			srv := &http.Server{Addr: ":8080", Handler: h2c.NewHandler(
				mux, &http2.Server{})}

			// Serve connect http2 server
			eg.Go(func() error {
				err := srv.ListenAndServe()
				if errors.Is(err, http.ErrServerClosed) {
					return nil
				}
				return err
			})

			eg.Go(func() error {
				err := discovery.RunDiscoveryServer(egCtx, 5000)
				return err
			})

			// Cleanup
			eg.Go(func() error {
				<-egCtx.Done()

				//c.ClosePeerUpdateChannels()
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				if err := srv.Shutdown(shutdownCtx); err != nil {
					return fmt.Errorf("Error on http server shutdown: %w", err)
				}
				return nil
			})

			if err = eg.Wait(); err != nil {
				log.Fatal(err)
			}
		},
	}
)

func init() {
	rootCmd.PersistentFlags().StringVar(&storePath, "storepath", "store.db", "file path for controller store persistence")
	rootCmd.PersistentFlags().BoolVar(&autoCert, "autocert", false, "enable autocert for controller")
}

// TODO handle signals and contextual things here
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
