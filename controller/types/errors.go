package types

import "errors"

var (
	ErrNotFound               = errors.New("object not found")
	ErrAlreadyExists          = errors.New("object already exists")
	ErrCastingObject          = errors.New("error casting object")
	ErrInvalidPeerID          = errors.New("invalid peer id")
	ErrInvalidPublicKey       = errors.New("invalid public key")
	ErrInvalidRegisterKey     = errors.New("invalid register key")
	ErrInvalidEndpoint        = errors.New("invalid endpoint")
	ErrRegisterKeyNotFound    = errors.New("register key now found")
	ErrRegisterKeyAlreadyUsed = errors.New("register key has already been used")
)
