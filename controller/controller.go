package controller

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"time"

	"github.com/caldog20/overlay/controller/store"
	"github.com/caldog20/overlay/controller/types"
	"github.com/caldog20/overlay/proto/gen/control/v1/controlv1connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.org/x/sync/errgroup"
)

const (
	Prefix = "100.70.0.0/24"
)

type Controller struct {
	store  store.Store
	prefix netip.Prefix
	config *types.Config
}

func NewController(store store.Store) *Controller {
	c := &Controller{
		store: store,
	}

	return c
}

func (c *Controller) Run(ctx context.Context) error {
	eg, egCtx := errgroup.WithContext(ctx)
	controlv1 := NewControlV1(c)

	mux := http.NewServeMux()
	cv1Path, cv1Handler := controlv1connect.NewControllerServiceHandler(controlv1)
	mux.Handle(cv1Path, cv1Handler)

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

	return eg.Wait()
}

//func (c *Controller) AllocatePeerIP() (string, error) {
//	usedIPs, err := c.store.GetAllocatedIPs()
//	if err != nil {
//		return "", err
//	}
//
//
//	return "", nil
//}
