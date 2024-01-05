package controller

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/caldog20/overlay/proto"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	Subnet = "100.65.0."
)

type Controller struct {
	store           *Store
	grpcServer      *grpc.Server
	config          *Config
	autocertManager *autocert.Manager
	autocertServer  *http.Server
	discovery       *net.UDPConn
	ipCount         atomic.Uint64

	peerChannels sync.Map
	proto.UnimplementedControlPlaneServer
}

func NewController(config *Config) (*Controller, error) {
	c := new(Controller)

	store, err := NewStore(config.DbPath)
	if err != nil {
		return nil, err
	}

	err = store.Migrate()
	if err != nil {
		return nil, err
	}

	autocertManager := &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(config.AutoCert.Domain),
		Cache:      autocert.DirCache(config.AutoCert.CacheDir),
	}

	var creds credentials.TransportCredentials
	if config.AutoCert.Enabled {
		creds = credentials.NewTLS(&tls.Config{GetCertificate: autocertManager.GetCertificate})
	} else {
		creds = insecure.NewCredentials()
	}

	grpcServer := grpc.NewServer(grpc.Creds(creds))
	proto.RegisterControlPlaneServer(grpcServer, c)

	c.store = store
	c.autocertManager = autocertManager
	c.grpcServer = grpcServer
	c.config = config
	c.peerChannels = sync.Map{}

	return c, nil
}

func (c *Controller) RunController(ctx context.Context) {
	eg, egCtx := errgroup.WithContext(ctx)
	if c.config.AutoCert.Enabled {
		log.Println("starting autocert handler")
		eg.Go(func() error { return c.AutocertHandler(c.autocertManager) })
	}

	log.Println("starting discovery server")
	eg.Go(func() error { return c.StartDiscoveryServer() })

	log.Printf("starting grpc server on port %d autocert:%t", c.config.GrpcPort, c.config.AutoCert.Enabled)
	lis, err := net.Listen("tcp4", fmt.Sprintf(":%d", c.config.GrpcPort))
	if err != nil {
		log.Fatal("error starting tcp listener for grpc server")
	}

	eg.Go(func() error { return c.grpcServer.Serve(lis) })

	eg.Go(func() error {
		<-egCtx.Done()
		log.Println("controller shutting down")
		c.discovery.Close()
		c.autocertServer.Shutdown(context.Background())
		c.grpcServer.GracefulStop()
		return nil
	})

	if err := eg.Wait(); err != nil {
		log.Printf("error: %s", err)
	}
}
