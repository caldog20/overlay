package controller

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"

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
	nextIP  int
}

type Client struct {
	id    uuid.UUID
	user  string
	tunIP string
}

func (s *ControlServer) getNextIP() string {
	allocated := s.nextIP
	s.nextIP += 1
	return fmt.Sprintf("100.65.0.%d", allocated)

}

func (s *ControlServer) Register(ctx context.Context, req *msg.RegisterRequest) (*msg.RegisterReply, error) {
	if req.User == "" {
		return &msg.RegisterReply{Success: false}, nil
	}

	client := &Client{
		id:    uuid.New(),
		user:  req.User,
		tunIP: s.getNextIP(),
	}

	s.clients.Store(client.id, client)

	return &msg.RegisterReply{
		Success: true,
		Uuid:    client.id.String(),
		Tunip:   client.tunIP,
	}, nil
}

func Run(ctx context.Context) {

	lis, err := net.Listen("tcp4", ":9000")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	cServer := &ControlServer{
		nextIP: 1,
	}

	grpcServer := grpc.NewServer()
	msg.RegisterControlServiceServer(grpcServer, cServer)

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		<-ctx.Done()
		grpcServer.GracefulStop()
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
