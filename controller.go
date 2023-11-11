package main

import (
	"context"
	"errors"
	"fmt"
	"google.golang.org/grpc/peer"
	"google.golang.org/protobuf/types/known/emptypb"
	"log"
	"net"
	"runtime"
	"strings"
	"sync"

	"github.com/caldog20/go-overlay/ipam"
	"github.com/caldog20/go-overlay/msg"
	"github.com/google/uuid"
	"google.golang.org/grpc"
)

//import (
//	"bufio"
//	"context"
//	"fmt"
//	"log"
//	"net"
//	"sync"
//
//	"github.com/caldog20/go-overlay/msg"
//	"github.com/google/uuid"
//
//	"google.golang.org/protobuf/proto"
//)

type ControlServer struct {
	msg.UnimplementedControlServiceServer
	clients sync.Map
	cipam   *ipam.Ipam
}

func (s *ControlServer) Register(ctx context.Context, req *msg.RegisterRequest) (*msg.RegisterReply, error) {
	if req.User == "" {
		return &msg.RegisterReply{Success: false}, nil
	}

	cid := uuid.New()
	cip, err := s.cipam.AllocateIP(cid.String())
	if err != nil {
		log.Println(err)
	}

	p, _ := peer.FromContext(ctx)
	remote := p.Addr.String()

	newclient := &Client{
		Id:     cid,
		User:   req.User,
		TunIP:  cip,
		Remote: strings.Split(remote, ":")[0] + ":2222",
	}

	s.clients.Store(newclient.Id.String(), newclient)

	return &msg.RegisterReply{
		Success: true,
		Uuid:    newclient.Id.String(),
		Tunip:   newclient.TunIP,
	}, nil
}

func (s *ControlServer) ClientInfo(ctx context.Context, req *msg.ClientInfoRequest) (*msg.ClientInfoReply, error) {
	rid := req.GetRequesterId()
	if rid == "" {
		return nil, errors.New("requestor id must not be nil")
	}

	_, found := s.clients.Load(rid)
	if !found {
		return nil, errors.New("requesting client invalid")
	}

	var clientList []*msg.ClientInfoReply_Client

	if req.Bulk {
		s.clients.Range(func(k, v interface{}) bool {
			cid := fmt.Sprint(k)
			c := v.(*Client)
			if cid != rid {
				clientList = append(clientList, &msg.ClientInfoReply_Client{
					Uuid:   c.Id.String(),
					User:   c.User,
					Tunip:  c.TunIP,
					Remote: c.Remote,
				})
			}
			return true
		})

	} else {
		cquery, found := s.clients.Load(req.ClientId)

		if !found {
			return nil, errors.New("client not found")
		} else {
			result := cquery.(*Client)
			clientList = append(clientList, &msg.ClientInfoReply_Client{
				Uuid:   result.Id.String(),
				User:   result.User,
				Tunip:  result.TunIP,
				Remote: result.Remote,
			})
		}
	}

	return &msg.ClientInfoReply{
		Clients:     clientList,
		ClientCount: uint64(len(clientList)),
	}, nil
}

func (s *ControlServer) PunchNotifier(req *msg.PunchSubscribe, stream msg.ControlService_PunchNotifierServer) error {
	fin := make(chan bool)

	cl, found := s.clients.Load(req.RequestorId)
	if !found {
		return errors.New("requesting client not registered")
	}

	c := cl.(*Client)

	c.PunchStream = stream
	c.Finished = fin

	s.clients.Store(c.Id.String(), c)

	ctx := stream.Context()

	for {
		select {
		case <-fin:
			log.Print("client stream closing")
			s.cipam.DeallocateIP(c.TunIP)
			s.clients.Delete(c.Id.String())
			return nil
		case <-ctx.Done():
			log.Print("client disconnected")
			s.cipam.DeallocateIP(c.TunIP)
			s.clients.Delete(c.Id.String())
			return nil
		}
	}

}

func (s *ControlServer) Punch(ctx context.Context, req *msg.PunchRequest) (*emptypb.Empty, error) {
	rid := req.GetRequestorId()
	if rid == "" {
		return nil, errors.New("requestor id must not be nil")
	}

	_, found := s.clients.Load(req.RequestorId)
	if !found {
		return nil, errors.New("requesting client not registered")
	}

	p, found := s.clients.Load(req.GetPuncheeId())
	if !found {
		return nil, errors.New("punchee client not found")
	}

	punchee := p.(*Client)
	if err := punchee.PunchStream.Send(&msg.PunchNotification{Puncher: rid}); err != nil {
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
		cipam: i,
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
