package controller

import (
	"context"
	"errors"
	"google.golang.org/protobuf/types/known/emptypb"
	"log"
	"net"
	"runtime"
	"strings"
	"sync"

	"github.com/caldog20/go-overlay/ipam"
	"github.com/caldog20/go-overlay/msg"

	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
)

type client struct {
	Id          string
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

	return &emptypb.Empty{}, nil
}

func (s *ControlServer) PunchSubscriber(req *msg.PunchSubscribe, stream msg.ControlService_PunchSubscriberServer) error {
	log.Printf("vpn ip %s subscribing to punch stream")

	ctx := stream.Context()

	fin := make(chan bool)

	var c *client
	s.clients.Range(func(k, v interface{}) bool {
		cl := v.(*client)
		if cl.VpnIP == req.VpnIp {
			c = cl
			return false
		}
		return true
	})

	if c != nil {
		c.PunchStream = stream
		c.Finished = fin
	} else {
		return errors.New("error finding client requesting stream")
	}

	for {
		select {
		case <-ctx.Done():
			s.clients.Delete(c.Id)
			s.ipman.DeallocateIP((c.VpnIP))
			return nil
		case <-fin:
			s.clients.Delete(c.Id)
			s.ipman.DeallocateIP((c.VpnIP))
			return nil
		}

	}

}

func (s *ControlServer) Register(ctx context.Context, req *msg.RegisterRequest) (*msg.RegisterReply, error) {
	if req.Id == "" {
		return nil, errors.New("id must not be nil")
	}

	p, _ := peer.FromContext(ctx)
	remote := p.Addr.String()

	cip, err := s.ipman.AllocateIP(req.Id)
	if err != nil {
		log.Println(err)
		return nil, errors.New("error allocating IP address")
	}

	newclient := &client{
		Id: req.Id,
		//Hostname: req.Hostname,
		VpnIP:  cip,
		Remote: strings.Split(remote, ":")[0] + ":" + req.Port,
	}

	s.clients.Store(req.Id, newclient)

	return &msg.RegisterReply{
		VpnIp: newclient.VpnIP,
	}, nil
}

func (s *ControlServer) Deregister(ctx context.Context, req *msg.DeregisterRequest) (*emptypb.Empty, error) {
	if req.Id == "" {
		return nil, errors.New("uuid must not be nil")
	}

	//p, _ := peer.FromContext(ctx)
	//remote := p.Addr.String()

	s.clients.Delete(req.Id)

	// change this to return a bool for found
	ip, err := s.ipman.WhoIsByID(req.Id)
	if err != nil {
		log.Printf("ip not found: %v", err)
	}

	err = s.ipman.DeallocateIP(ip)
	if err != nil {
		log.Printf("cannot deallocate ip: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (s *ControlServer) WhoIsIp(ctx context.Context, req *msg.WhoIsIPRequest) (*msg.WhoIsIPReply, error) {
	vpnip := req.GetVpnIp()

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

	return &msg.WhoIsIPReply{
		Remote: &msg.Remote{
			Id:     client.Id,
			VpnIp:  client.VpnIP,
			Remote: client.Remote,
		},
	}, nil
}

func (s *ControlServer) WhoIsID(ctx context.Context, req *msg.WhoIsIDRequest) (*msg.WhoIsIDReply, error) {
	id := req.GetId()

	ip, err := s.ipman.WhoIsByID(id)
	if err != nil {
		return nil, errors.New("client not found")
	}

	//c, ok := s.clients.Load(id)
	//if !ok {
	//	return nil, errors.New("vpn ip not found")
	//}

	// check cast error here
	//client := c.(*client)

	return &msg.WhoIsIDReply{
		Remote: &msg.Remote{
			VpnIp: ip,
		},
	}, nil
}

func (s *ControlServer) RemoteList(ctx context.Context, req *msg.RemoteListRequest) (*msg.RemoteListReply, error) {
	reqid := req.Id
	_, ok := s.clients.Load(reqid)
	if !ok {
		return nil, errors.New("requesting client not registered")
	}

	var rl []*msg.Remote

	s.clients.Range(func(k, v interface{}) bool {
		if k.(string) != reqid {
			r := &msg.Remote{
				Id:     v.(*client).Id,
				VpnIp:  v.(*client).VpnIP,
				Remote: v.(*client).Remote,
			}
			rl = append(rl, r)
		} else {
			log.Printf("not sending requestor its own client info: %v", reqid)
		}

		return true
	})

	return &msg.RemoteListReply{
		Remotes: rl,
	}, nil
}

func RunController(ctx context.Context) {
	runtime.LockOSThread()

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
