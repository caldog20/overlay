package controller

import (
	"context"

	"github.com/caldog20/overlay/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TODO Login currently registers peer automatically until authentication is implemented
func (c *Controller) Login(ctx context.Context, req *proto.LoginRequest) (*proto.LoginResponse, error) {
	if req.PublicKey == "" {
		return nil, status.Error(codes.InvalidArgument, "public key must not be nil")
	}

	peer, err := c.store.GetPeerByKey(req.PublicKey)
	if peer == nil {
		peer = &Peer{}
		peer.IP = c.AllocateIP().String()
		err = c.store.CreatePeer(peer)
		if err != nil {
			return nil, status.Error(codes.Internal, "error creating peer")
		}
		//return nil, status.Error(codes.NotFound, "peer is not registered")
	}

	peer.Connected = true
	peer.EndPoint = req.Endpoint.Endpoint
	err = c.store.UpdatePeer(peer)
	if err != nil {
		return nil, status.Error(codes.Internal, "error updating peer in database")
	}
	return &proto.LoginResponse{
		Id: peer.ID,
		Config: &proto.PeerConfig{
			TunnelIp: peer.IP,
		},
	}, nil
}
