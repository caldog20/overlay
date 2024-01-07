package controller

import (
	"errors"
	"net/http"
	"time"

	"golang.org/x/crypto/acme/autocert"
)

func (c *Controller) AutocertHandler(am *autocert.Manager) error {
	srv := &http.Server{
		Addr:         ":80",
		Handler:      am.HTTPHandler(nil),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	err := srv.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
