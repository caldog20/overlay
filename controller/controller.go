package controller

import (
	"context"
	"errors"
	"google.golang.org/protobuf/types/known/emptypb"
	"log"
	"net"
	"strings"
	"sync"

	"github.com/caldog20/go-overlay/ipam"
	"github.com/caldog20/go-overlay/msg"

	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
)

type client struct {
	DeviceKey   string
	Id          uint32
	Key         string
	Hostname    string
	VpnIP       string
	Remote      string
	PunchStream msg.ControlService_PunchSubscriberServer
	Finished    chan bool
}

type ControlServer struct {
	msg.UnimplementedControlServiceServer
	clients sync.Map
	ipman   *ipam.Ipam
}

func (s *ControlServer) Punch(ctx context.Context, req *msg.PunchRequest) (*emptypb.Empty, error) {
	id, _ := s.ipman.WhoIsByIP(req.SrcVpnIp)
	c, found := s.clients.Load(id)
	if !found {
		return nil, errors.New("requesting client not registered")
	}

	id, _ = s.ipman.WhoIsByIP(req.DstVpnIp)
	p, found := s.clients.Load(id)
	if !found {
		return nil, errors.New("punchee client not found")
	}

	preq := c.(*client)
	punchee := p.(*client)

	if err := punchee.PunchStream.Send(&msg.PunchNotification{Remote: preq.Remote}); err != nil {
		select {
		case punchee.Finished <- true:
			log.Print("unsubscribed client")
		default:
		}

		// handle unsubscribing and errors
	}
	log.Printf("Sent punch request to %s for remote %s", req.DstVpnIp, preq.Remote)

	return &emptypb.Empty{}, nil
}

func (s *ControlServer) PunchSubscriber(req *msg.PunchSubscribe, stream msg.ControlService_PunchSubscriberServer) error {
	if req.Id == 0 {
		return errors.New("id must not be zero")
	}

	ctx := stream.Context()
	p, _ := peer.FromContext(ctx)
	remote := p.Addr.String()
	log.Printf("remote: %s vpn ip %s subscribing to punch stream", remote, req.Id)

	fin := make(chan bool)
	var cl *client
	c, found := s.clients.Load(req.Id)
	if found {
		cl = c.(*client)
		cl.PunchStream = stream
		cl.Finished = fin
	} else {
		return errors.New("error finding client requesting stream")
	}

	for {
		select {
		case <-ctx.Done():
			s.clients.Delete(cl.Id)
			//s.ipman.DeallocateIP((c.VpnIP))
			return nil
		case <-fin:
			s.clients.Delete(cl.Id)
			//s.ipman.DeallocateIP((c.VpnIP))
			return nil
		}

	}
}

func (s *ControlServer) Register(ctx context.Context, req *msg.RegisterRequest) (*msg.RegisterReply, error) {
	if req.Id == 0 {
		return nil, errors.New("id must not be zero")
	}
	if req.Key == "" {
		return nil, errors.New("key must not be nil")
	}
	// Get remote address of peer
	p, _ := peer.FromContext(ctx)
	remote := p.Addr.String()

	// Check to see if IP is already allocated for ID

	cip, err := s.ipman.WhoIsByID(req.Id)
	if err != nil {
		// ID not found, allocate
		cip, err = s.ipman.AllocateIP(req.Id)
		if err != nil {
			log.Println(err)
			return nil, errors.New("error allocating IP address")
		}
	}

	newclient := &client{
		Id:     req.Id,
		Key:    req.Key,
		VpnIP:  cip,
		Remote: strings.Split(remote, ":")[0] + ":" + req.Port,
	}

	s.clients.Store(req.Id, newclient)

	log.Printf("Registered Node - ID: %d - Remote: %s", newclient.Id, newclient.Remote)

	return &msg.RegisterReply{
		VpnIp: newclient.VpnIP,
	}, nil
}

func (s *ControlServer) Deregister(ctx context.Context, req *msg.DeregisterRequest) (*emptypb.Empty, error) {
	if req.Id == 0 {
		return nil, errors.New("id must not be zero")
	}

	s.clients.Delete(req.Id)
	log.Printf("Deregistered node ID: %d", req.Id)
	return &emptypb.Empty{}, nil
}

func (s *ControlServer) WhoIsIp(ctx context.Context, req *msg.WhoIsIPRequest) (*msg.Remote, error) {
	vpnip := req.GetVpnIp()

	if vpnip == "" {
		return nil, errors.New("ip must not be nil")
	}

	id, err := s.ipman.WhoIsByIP(vpnip)
	if err != nil {
		return nil, errors.New("client not found")
	}

	c, ok := s.clients.Load(id)
	if !ok {
		return nil, errors.New("vpn ip not found")
	}

	// check cast error here
	client := c.(*client)

	return &msg.Remote{
		Id:     client.Id,
		Key:    client.Key,
		VpnIp:  client.VpnIP,
		Remote: client.Remote,
	}, nil
}

func (s *ControlServer) WhoIsID(ctx context.Context, req *msg.WhoIsIDRequest) (*msg.Remote, error) {
	id := req.GetId()
	if id == 0 {
		return nil, errors.New("id must not be zero")
	}

	c, found := s.clients.Load(id)
	if !found {
		return nil, errors.New("client not found")
	}

	//c, ok := s.clients.Load(id)
	//if !ok {
	//	return nil, errors.New("vpn ip not found")
	//}

	//check cast error here
	client := c.(*client)

	return &msg.Remote{
		Id:     client.Id,
		Key:    client.Key,
		VpnIp:  client.VpnIP,
		Remote: client.Remote,
	}, nil
}

func (s *ControlServer) RemoteList(ctx context.Context, req *msg.RemoteListRequest) (*msg.RemoteListReply, error) {
	id := req.Id
	if id == 0 {
		return nil, errors.New("id must not be zero")
	}

	_, ok := s.clients.Load(id)
	if !ok {
		return nil, errors.New("requesting client not registered")
	}

	var rl []*msg.Remote

	s.clients.Range(func(k, v interface{}) bool {
		if k.(uint32) != id {
			r := &msg.Remote{
				Id:     v.(*client).Id,
				Key:    v.(*client).Key,
				VpnIp:  v.(*client).VpnIP,
				Remote: v.(*client).Remote,
			}
			rl = append(rl, r)
		} else {
			log.Printf("not sending requestor its own client info: %v", id)
		}

		return true
	})

	return &msg.RemoteListReply{
		Remotes: rl,
	}, nil
}

func RunController(ctx context.Context) {

	lis, err := net.Listen("tcp4", ":5555")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	i, err := ipam.NewIpam("192.168.77.0/24")
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
