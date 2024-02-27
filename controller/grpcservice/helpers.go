package grpcservice

import (
	"encoding/base64"
	"net/netip"

	"github.com/caldog20/overlay/controller/types"
)

func validatePublicKey(key string) error {
	if key == "" {
		return types.ErrInvalidPublicKey
	}

	k, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return types.ErrInvalidPublicKey
	}

	if len(k) != 32 {
		return types.ErrInvalidPublicKey
	}

	return nil
}

func validateID(id uint32) error {
	if id == 0 {
		return types.ErrInvalidPeerID
	}

	return nil
}

func validateEndpoint(endpoint string) error {
	if endpoint == "" {
		return types.ErrInvalidEndpoint
	}
	_, err := netip.ParseAddrPort(endpoint)
	if err != nil {
		return types.ErrInvalidEndpoint
	}
	return nil
}
