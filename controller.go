package main

import (
	"context"
	"errors"
	"google.golang.org/grpc/peer"
	"google.golang.org/protobuf/types/known/emptypb"
	"log"
	"net"
	"runtime"
	"strings"
	"sync"

	"github.com/caldog20/go-overlay/ipam"
	"github.com/caldog20/go-overlay/msg"
	"google.golang.org/grpc"
)

type ControlServer struct {
	msg.UnimplementedControlServiceServer
	// Key vpnIP Value *Client
	clients sync.Map
	ipman   *ipam.Ipam
}

func (s *ControlServer) Register(ctx context.Context, req *msg.RegisterRequest) (*msg.RegisterReply, error) {
	if req.Hostname == "" {
		return nil, errors.New("hostname must not be nil")
	}

	cip, err := s.ipman.AllocateIP(req.Hostname)
	if err != nil {
		log.Println(err)
		return nil, errors.New("error allocating IP address")
	}

	p, _ := peer.FromContext(ctx)
	remote := p.Addr.String()

	newclient := &Client{
		Hostname: req.Hostname,
		VpnIP:    cip,
		Remote:   strings.Split(remote, ":")[0] + req.Port,
	}

	s.clients.Store(cip, newclient)

	return &msg.RegisterReply{
		VpnIp: newclient.VpnIP,
	}, nil
}

func (s *ControlServer) WhoIs(ctx context.Context, req *msg.WhoIsIP) (*msg.WhoIsIPReply, error) {
	vpnip := req.GetVpnIp()

	c, ok := s.clients.Load(vpnip)
	if !ok {
		return nil, errors.New("vpn ip not found")
	}

	// check cast error here
	client := c.(*Client)

	return &msg.WhoIsIPReply{
		Remote: client.Remote,
	}, nil
}

func (s *ControlServer) PunchNotifier(req *msg.PunchSubscribe, stream msg.ControlService_PunchNotifierServer) error {
	fin := make(chan bool)

	cl, found := s.clients.Load(req.VpnIp)
	if !found {
		return errors.New("requesting client not registered")
	}

	c := cl.(*Client)

	c.PunchStream = stream
	c.Finished = fin

	s.clients.Store(c.VpnIP, c)

	ctx := stream.Context()

	for {
		select {
		case <-fin:
			log.Print("client stream closing")
			s.ipman.DeallocateIP(c.VpnIP)
			s.clients.Delete(c.VpnIP)
			return nil
		case <-ctx.Done():
			s.clients.Delete(c.VpnIP)
			s.ipman.DeallocateIP(c.VpnIP)
			return nil
		}
	}

}

func (s *ControlServer) Punch(ctx context.Context, req *msg.PunchRequest) (*emptypb.Empty, error) {
	c, found := s.clients.Load(req.SrcVpnIp)
	if !found {
		return nil, errors.New("requesting client not registered")
	}

	p, found := s.clients.Load(req.DstVpnIp)
	if !found {
		return nil, errors.New("punchee client not found")
	}

	preq := c.(*Client)
	punchee := p.(*Client)

	if err := punchee.PunchStream.Send(&msg.PunchNotification{Remote: preq.Remote}); err != nil {
		select {
		case punchee.Finished <- true:
			log.Print("unsubscribed client")
		default:
		}

		// handle unsubscribing and errors
	}

	return &emptypb.Empty{}, nil
}

func RunController(ctx context.Context) {
	runtime.LockOSThread()

	lis, err := net.Listen("tcp4", ":5555")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	i, err := ipam.NewIpam("192.168.1.0/24")
	if err != nil {
		log.Fatal(err)
	}

	cServer := &ControlServer{
		ipman: i,
	}

	grpcServer := grpc.NewServer()
	msg.RegisterControlServiceServer(grpcServer, cServer)

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		<-ctx.Done()
		//grpcServer.GracefulStop()
		grpcServer.Stop()
		wg.Done()
	}()

	log.Printf("starting grpc server on %v", lis.Addr().String())
	err = grpcServer.Serve(lis)
	if err != nil {
		log.Fatalf("grpc serve error: %v", err)
	}
	wg.Wait()
	log.Println("controller shutting down")

}
