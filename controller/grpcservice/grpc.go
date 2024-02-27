package grpcservice

import (
	"context"
	"errors"
	"log"

	"github.com/caldog20/overlay/controller"
	"github.com/caldog20/overlay/controller/types"
	proto "github.com/caldog20/overlay/proto/gen"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	TempRegisterKey = "registermeplz!"
)

type GRPCServer struct {
	proto.UnimplementedControlPlaneServer
	controller *controller.Controller
}

func NewGRPCServer(controller *controller.Controller) *GRPCServer {
	return &GRPCServer{
		controller: controller,
	}
}

func (s *GRPCServer) LoginPeer(
	ctx context.Context,
	req *proto.LoginRequest,
) (*proto.LoginResponse, error) {
	err := validatePublicKey(req.PublicKey)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "public key is invalid")
	}

	peer, err := s.controller.LoginPeer(req.PublicKey)
	if err != nil {
		if err == types.ErrNotFound {
			return nil, status.Error(codes.NotFound, "peer not registered")
		} else {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	cfg := peer.ProtoConfig()

	return &proto.LoginResponse{Config: cfg}, nil
}

func (s *GRPCServer) RegisterPeer(
	ctx context.Context,
	req *proto.RegisterRequest,
) (*proto.RegisterResponse, error) {
	err := validatePublicKey(req.PublicKey)
	if err != nil {
		return nil, err
	}

	if req.RegisterKey != TempRegisterKey {
		return nil, types.ErrInvalidRegisterKey
	}

	err = s.controller.RegisterPeer(req.PublicKey)
	if err != nil {
		return nil, err
	}

	return &proto.RegisterResponse{}, nil
}

// TODO Authentication/encryption for messages
func (s *GRPCServer) SetPeerEndpoint(
	ctx context.Context,
	endpoint *proto.Endpoint,
) (*proto.EmptyResponse, error) {
	err := validateID(endpoint.Id)
	if err != nil {
		return nil, err
	}

	err = validateEndpoint(endpoint.Endpoint)
	if err != nil {
		return nil, err
	}

	err = s.controller.SetPeerEndpoint(endpoint.Id, endpoint.Endpoint)
	if err != nil {
		return nil, err
	}

	return &proto.EmptyResponse{}, nil
}

func (s *GRPCServer) Update(
	req *proto.UpdateRequest,
	stream proto.ControlPlane_UpdateServer,
) error {
	err := validateID(req.Id)
	if err != nil {
		return err
	}

	// Get the update channel for this peer
	peerChan := s.controller.GetPeerUpdateChan(req.Id)
	if err != nil {
		return err
	}

	err = s.controller.PeerConnected(req.Id)
	if err != nil {
		return err
	}

	// Send initial list of peers
	// TODO Separate this into a function somewhere
	peers, err := s.GetInitialPeerList(req.Id)
	if err != nil {
		return err
	}

	initialSync := &proto.UpdateResponse{
		UpdateType: proto.UpdateResponse_INIT,
		PeerList: &proto.RemotePeerList{
			Count: uint32(len(peers)),
			Peers: peers,
		},
	}

	err = stream.Send(initialSync)
	if err != nil {
		log.Printf("error sending data over stream to peer: %d", req.Id)
		return err
	}

	for {
		select {
		case <-stream.Context().Done():
			// Client disconnected, mark disconnected and send disconnect event to other peers
			err := s.controller.PeerDisconnected(req.Id)
			return err
		case update, ok := <-peerChan:
			if !ok {
				// channel was closed on outside, server forcing peer to disconnect
				return errors.New("server closed connection")
			}
			err = stream.Send(update)
			if err != nil {
				log.Printf("error sending data over stream to peer: %d", req.Id)
			}
		}
	}
}

func (s *GRPCServer) GetInitialPeerList(connectingPeerID uint32) ([]*proto.Peer, error) {
	peers, err := s.controller.GetConnectedPeers()
	if err != nil {
		return nil, err
	}

	var rp []*proto.Peer
	for _, p := range peers {
		if p.ID == connectingPeerID {
			continue
		}
		rp = append(rp, &proto.Peer{
			Id:        p.ID,
			PublicKey: p.PublicKey,
			Endpoint:  p.Endpoint,
			TunnelIp:  p.IP,
		})
	}

	return rp, nil
}

func (s *GRPCServer) Punch(
	ctx context.Context,
	req *proto.PunchRequest,
) (*proto.EmptyResponse, error) {
	err := validateID(req.ReqPeerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, types.ErrInvalidPeerID.Error())
	}

	err = validateID(req.DstPeerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, types.ErrInvalidPeerID.Error())
	}

	err = s.controller.EventPunchRequest(req.DstPeerId, req.Endpoint)
	if err != nil {
		return nil, status.Error(codes.Internal, "error processing punch request")
	}

	return &proto.EmptyResponse{}, nil
}
