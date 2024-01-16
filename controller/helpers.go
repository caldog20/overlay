package controller

import (
	"encoding/base64"
	"net/netip"
)

type Error string

func (e Error) Error() string { return string(e) }

const (
	ErrNotFound           = Error("object not found")
	ErrAlreadyExists      = Error("object already exists")
	ErrCastingObject      = Error("error casting object")
	ErrInvalidPeerID      = Error("invalid peer id")
	ErrInvalidPublicKey   = Error("invalid public key")
	ErrInvalidRegisterKey = Error("invalid register key")
	ErrInvalidEndpoint    = Error("invalid endpoint")
)

func validatePublicKey(key string) error {
	if key == "" {
		return ErrInvalidPublicKey
	}

	k, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return ErrInvalidPublicKey
	}

	if len(k) != 32 {
		return ErrInvalidPublicKey
	}

	return nil
}

func validateID(id uint32) error {
	if id == 0 {
		return ErrInvalidPeerID
	}

	return nil
}

func validateEndpoint(endpoint string) error {
	if endpoint == "" {
		return ErrInvalidEndpoint
	}
	_, err := netip.ParseAddrPort(endpoint)
	if err != nil {
		return ErrInvalidEndpoint
	}
	return nil
}
