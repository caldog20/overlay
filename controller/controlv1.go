package controller

import (
	"context"
	"errors"
	"log"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	controlv1 "github.com/caldog20/overlay/proto/gen/control/v1"
)

type ControlV1 struct {
	controller *Controller
}

func NewControlV1(controller *Controller) *ControlV1 {
	return &ControlV1{
		controller: controller,
	}
}

func (c *ControlV1) RegisterPeer(
	ctx context.Context,
	req *connect.Request[controlv1.RegisterPeerRequest],
) (*connect.Response[controlv1.RegisterPeerResponse], error) {
	log.Println("Register Request Headers: ", req.Header())
	// TODO: Function to validate/lookup register key in header
	key := req.Header().Get("register-key")
	err := validateRegisterKey(key)
	if err != nil {
		return nil, err
	}

	// Lookup register key and verify using controller

	return connect.NewResponse(&controlv1.RegisterPeerResponse{}), nil
}

func validateRegisterKey(key string) *connect.Error {
	_, err := uuid.Parse(key)
	if err != nil {
		return connect.NewError(
			connect.CodeUnauthenticated,
			errors.New("invalid register key in header"),
		)
	}
	return nil
}
