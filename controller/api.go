package controller

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/caldog20/overlay/controller/auth"
	apiv1 "github.com/caldog20/overlay/proto/gen/api/v1"
)

//func (c *Controller) CreateRegisterKey(
//	ctx context.Context,
//	req *connect.Request[apiv1.CreateRegisterKeyRequest],
//) (*connect.Response[apiv1.CreateRegisterKeyResponse], error) {
//	log.Println("Register Request Headers: ", req.Header())
//
//	registerKey := types.NewRegisterKey(req.Msg.GetReusable())
//
//	err := c.store.CreateRegisterKey(registerKey)
//	if err != nil {
//		return nil, connect.NewError(
//			connect.CodeInternal,
//			fmt.Errorf("error creating register key: %w", err),
//		)
//	}
//
//	return connect.NewResponse(&apiv1.CreateRegisterKeyResponse{
//		RegisterKey: registerKey.Key,
//	}), nil
//}

func (c *Controller) GetToken(ctx context.Context, req *connect.Request[apiv1.GetTokenRequest]) (*connect.Response[apiv1.GetTokenResponse], error) {
	username := req.Msg.GetUsername()
	password := req.Msg.GetPassword()

	user, err := c.store.GetUser(username)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	if !user.ComparePassword(password) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid password"))
	}

	token, err := auth.GenerateToken(username)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&apiv1.GetTokenResponse{Token: token}), nil
}
