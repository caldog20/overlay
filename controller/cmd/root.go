package cmd

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/caldog20/overlay/controller"
	"github.com/caldog20/overlay/controller/discovery"
	"github.com/caldog20/overlay/controller/grpcsvc"
	"github.com/caldog20/overlay/controller/store"
	controllerv1 "github.com/caldog20/overlay/proto/gen/controller/v1"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var (
	storePath     string
	autoCert      bool
	grpcPort      uint16
	discoveryPort uint16

	rootCmd = &cobra.Command{
		Use:   "controller",
		Short: "Overlay Controller",
		Long:  "",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithCancel(context.Background())
			go func() {
				sigchan := make(chan os.Signal, 1)
				signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)
				log.Printf("Received %v signal, shutting down services\n", <-sigchan)
				cancel()
			}()

			// TODO Implement config stuff/multiple commands
			log.Printf("initializing sqlite store using file: %s", storePath)
			store, err := store.New(storePath)
			if err != nil {
				log.Fatal(err)
			}

			ctrl := controller.NewController(store)
			//err = ctrl.CreateAdminUser()
			//if err != nil {
			//	log.Println(err)
			//}

			// GRPC Server
			grpcServer := grpcsvc.NewGRPCServer(ctrl)
			server := grpc.NewServer()
			controllerv1.RegisterControllerServiceServer(server, grpcServer)
			reflection.Register(server)

			// Discovery Server
			discovery, err := discovery.New(discoveryPort)
			if err != nil {
				log.Fatal(err)
			}

			eg, egCtx := errgroup.WithContext(ctx)

			eg.Go(func() error {
				log.Printf("starting grpc server on port: %d", grpcPort)
				conn, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
				if err != nil {
					return err
				}
				return server.Serve(conn)
			})

			eg.Go(func() error {
				log.Printf("starting discovery server on port: %d", discoveryPort)
				err := discovery.Listen(egCtx)
				return err
			})

			// Cleanup
			eg.Go(func() error {
				<-egCtx.Done()
				ctrl.ClosePeerUpdateChannels()

				server.GracefulStop()
				err := discovery.Stop()
				return err
			})

			// Wait for all errgroup routines to finish before exiting
			if err = eg.Wait(); err != nil {
				log.Fatal(err)
			}
		},
	}
)

func init() {
	rootCmd.PersistentFlags().StringVar(&storePath, "storepath", "store.db", "file path for controller store persistence")
	rootCmd.PersistentFlags().BoolVar(&autoCert, "autocert", false, "enable autocert for controller")
	rootCmd.PersistentFlags().Uint16Var(&grpcPort, "grpcport", 50000, "port to listen for grpc connections")
	rootCmd.PersistentFlags().Uint16Var(&discoveryPort, "discoveryport", 5050, "port to listen for grpc connections")
}

// TODO handle signals and contextual things here
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
