package controller

import (
	"context"
	"encoding/base64"
	"errors"
	"log"
	"net"
	"net/netip"

	"github.com/caldog20/overlay/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	TempRegisterKey = "registermeplz!"
)

type GRPCServer struct {
	proto.UnimplementedControlPlaneServer
	controller *Controller
	server     *grpc.Server
}

func NewGRPCServer(controller *Controller) *GRPCServer {
	grpcServer := new(GRPCServer)
	gserver := grpc.NewServer(grpc.Creds(insecure.NewCredentials()))
	proto.RegisterControlPlaneServer(gserver, grpcServer)
	reflection.Register(gserver)

	grpcServer.controller = controller
	grpcServer.server = gserver

	return grpcServer
}

func (s *GRPCServer) Run() error {
	conn, err := net.Listen("tcp4", ":9000")
	if err != nil {
		return err
	}

	log.Printf("Starting grpc server on port: %d", 9000)
	return s.server.Serve(conn)
}

func (s *GRPCServer) LoginPeer(ctx context.Context, req *proto.LoginRequest) (*proto.LoginResponse, error) {
	err := validatePublicKey(req.PublicKey)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "public key is invalid")
	}

	config, err := s.controller.LoginPeer(req.PublicKey)
	if err != nil {
		if err == ErrNotFound {
			return nil, status.Error(codes.NotFound, "peer not registered")
		} else {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	cfg := config.MarshalPeerConfig()

	return &proto.LoginResponse{Config: cfg}, nil
}

func (s *GRPCServer) RegisterPeer(ctx context.Context, req *proto.RegisterRequest) (*proto.RegisterResponse, error) {
	err := validatePublicKey(req.PublicKey)
	if err != nil {
		return nil, err
	}

	if req.RegisterKey != TempRegisterKey {
		return nil, ErrInvalidRegisterKey
	}

	err = s.controller.RegisterPeer(req.PublicKey)
	if err != nil {
		return nil, err
	}

	return &proto.RegisterResponse{}, nil
}

// TODO Authentication/encryption for messages
func (s *GRPCServer) SetPeerEndpoint(ctx context.Context, endpoint *proto.Endpoint) (*proto.EmptyResponse, error) {
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

func (s *GRPCServer) Update(req *proto.UpdateRequest, stream proto.ControlPlane_UpdateServer) error {
	err := validateID(req.Id)
	if err != nil {
		return err
	}

	// Get the update channel for this peer
	peerChan := s.controller.GetPeerUpdateChan(req.Id)
	if err != nil {
		return err
	}

	err = s.controller.MarkPeerConnected(req.Id)
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
			Count:      uint32(len(peers)),
			RemotePeer: peers,
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
			err := s.controller.MarkPeerDisconnected(req.Id)
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

func (s *GRPCServer) GetInitialPeerList(connectingPeerID uint32) ([]*proto.RemotePeer, error) {
	peers, err := s.controller.GetConnectedPeers()
	if err != nil {
		return nil, err
	}

	var rp []*proto.RemotePeer
	for _, p := range peers {
		if p.ID == connectingPeerID {
			continue
		}
		rp = append(rp, &proto.RemotePeer{
			Id:        p.ID,
			PublicKey: p.PublicKey,
			Endpoint:  p.Endpoint,
			TunnelIp:  p.IP,
		})
	}

	return rp, nil
}

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
