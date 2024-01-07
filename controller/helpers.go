package controller

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
