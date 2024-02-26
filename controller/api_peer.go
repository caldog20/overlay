package controller

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/netip"

	"connectrpc.com/connect"
	"github.com/caldog20/overlay/controller/auth"
	"github.com/caldog20/overlay/controller/types"
	apiv1 "github.com/caldog20/overlay/proto/gen/api/v1"
)

func (c *Controller) RegisterPeer(ctx context.Context, req *connect.Request[apiv1.RegisterPeerRequest]) (*connect.Response[apiv1.RegisterPeerResponse], error) {
	log.Println("register request headers: ", req.Header())
	nodeKey := req.Msg.GetNodeKey()
	token := req.Header().Get("token")

	user, err := auth.ValidateToken(token)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	ip, err := c.AllocatePeerIP()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("error allocating peer ip address"))
	}

	peer := &types.Peer{
		IP:      ip,
		NodeKey: nodeKey,
		User:    user,
	}

	err = c.store.CreatePeer(peer)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&apiv1.RegisterPeerResponse{}), nil
}

func (c *Controller) LoginPeer(ctx context.Context, req *connect.Request[apiv1.LoginPeerRequest]) (*connect.Response[apiv1.LoginPeerResponse], error) {
	log.Println("login request headers: ", req.Header())
	nodeKey := req.Header().Get("node-key")
	endpoint := req.Msg.GetEndpoint()
	publicKey := req.Msg.GetPublicKey()

	if err := validatePublicKey(publicKey); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	peer, err := c.store.GetPeerByNodeKey(nodeKey)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("peer with nodekey %s not found", nodeKey))
	}

	// TODO: Update other peers about this peer login
	peer.PublicKey = publicKey
	peer.Endpoint, err = netip.ParseAddrPort(endpoint)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("error parsing peer endpoint"))
	}

	err = c.store.UpdatePeer(peer)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("error setting peer public key/endpoint on login"))
	}

	config := peer.ProtoConfig()

	return connect.NewResponse(&apiv1.LoginPeerResponse{Config: config}), nil
}

//func (c *Controller) UpdatePeer()
