package cmd

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/caldog20/overlay/controller"
	"github.com/caldog20/overlay/controller/discovery"
	"github.com/caldog20/overlay/controller/grpcsvc"
	"github.com/caldog20/overlay/controller/store"
	proto "github.com/caldog20/overlay/proto/gen"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
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
			//err = ctrl.CreateAdminUser()
			//if err != nil {
			//	log.Println(err)
			//}

			// GRPC Server
			grpcServer := grpcsvc.NewGRPCServer(ctrl)
			server := grpc.NewServer()
			proto.RegisterControlPlaneServer(server, grpcServer)
			reflection.Register(server)

			eg, egCtx := errgroup.WithContext(ctx)

			eg.Go(func() error {
				conn, err := net.Listen("tcp", ":50000")
				if err != nil {
					return err
				}
				return server.Serve(conn)
			})

			eg.Go(func() error {
				err := discovery.StartDiscoveryServer(egCtx, 5050)
				return err
			})

			// Cleanup
			eg.Go(func() error {
				<-egCtx.Done()
				ctrl.ClosePeerUpdateChannels()

				server.GracefulStop()
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
