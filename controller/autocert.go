package controller

import (
	"errors"
	"log"
	"net/http"
	"time"

	"golang.org/x/crypto/acme/autocert"
)

func AutocertHandler(am *autocert.Manager) {

	srv := &http.Server{
		Addr:         ":80",
		Handler:      am.HTTPHandler(nil),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	err := srv.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return
	}
	log.Fatal(err)
}
