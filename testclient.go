package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"log"
	"net"
	"sync"
	"time"

	"github.com/caldog20/go-overlay/msg"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type GClient struct {
	hosts     sync.Map
	udpcon    *net.UDPConn
	gconn     *grpc.ClientConn
	msgclient msg.ControlServiceClient

	id    string
	tunip string
}

func RunClient(ctx context.Context, caddr string, doPunch bool) {
	log.SetPrefix("client: ")

	gclient := &GClient{}

	conn, err := grpc.DialContext(ctx, caddr, grpc.WithBlock(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("error connecting to grpc server: %v", err)
	}
	defer conn.Close()

	udpcon, err := net.ListenPacket("udp4", ":2222")
	if err != nil {
		log.Fatalf("error listening on udp socket: %v", err)
	}

	uc, ok := udpcon.(*net.UDPConn)
	if !ok {
		log.Fatal("error casting connection to net.udpConn")
	}

	mc := msg.NewControlServiceClient(conn)

	gclient.udpcon = uc
	gclient.gconn = conn
	gclient.msgclient = mc

	err = gclient.Register(ctx, "")
	if err != nil {
		log.Fatal(err)
	}

	err = gclient.Subscribe(ctx)

	go gclient.Listen(ctx)

	//pb := []byte("punchout")

	if doPunch {

		var clients []*msg.ClientInfoReply_Client

		for {
			// Request info about other connected clients
			ciresponse, rerr := gclient.msgclient.ClientInfo(ctx, &msg.ClientInfoRequest{RequesterId: gclient.id, Bulk: true})
			if rerr != nil {
				log.Printf("error sending/recv message: %v", rerr)
				return
			}

			clients = ciresponse.Clients

			if len(clients) == 0 {
				log.Println("no clients yet")
			} else {
				break
			}

		}

		// Write a few packets out first
		remote := clients[0].Remote
		log.Printf("requesting punch to remote %v", remote)
		raddr, _ := net.ResolveUDPAddr("udp4", remote)

		// Send Punch Request to client
		_, err = gclient.msgclient.Punch(ctx, &msg.PunchRequest{RequestorId: gclient.id, PuncheeId: clients[0].Uuid})
		if err != nil {
			log.Fatal(err)
		}
		log.Println("sent punch request, starting to write data to remote")
		// wait a few seconds
		time.Sleep(time.Second * 3)
		// Write more data
		gclient.udpcon.WriteToUDP([]byte("hello\n"), raddr)
		gclient.udpcon.WriteToUDP([]byte("punch worked\n"), raddr)
		gclient.udpcon.WriteToUDP([]byte("goodbye\n"), raddr)
	}

	<-ctx.Done()
	gclient.udpcon.Close()
	gclient.gconn.Close()

}

func (gc *GClient) Register(ctx context.Context, username string) error {
	// Register Client
	reply, err := gc.msgclient.Register(ctx, &msg.RegisterRequest{User: ""})
	if err != nil {
		log.Printf("error sending/recv message: %v", err)
		return errors.New("failed to register with controller")
	}

	if !reply.GetSuccess() {
		return errors.New("failed to register with controller")
	}

	gc.id = reply.GetUuid()
	gc.tunip = reply.GetTunip()

	log.Println("User registered successfully")
	log.Printf("uuid: %s - tunip: %s", reply.GetUuid(), reply.GetTunip())

	return nil
}

func (gc *GClient) Subscribe(ctx context.Context) error {
	// Subscribe to puncher service
	puncher, err := gc.msgclient.PunchNotifier(ctx, &msg.PunchSubscribe{RequestorId: gc.id})
	if err != nil {
		return err
	}

	log.Println("Starting puncher routine")
	go func() {
		for {
			punch, err := puncher.Recv()
			if err != nil {
				puncher = nil
				log.Printf("punch client stream read error")
				return
			}
			log.Printf("Received punch notification for client: %s", punch.GetPuncher())
			gc.Punch(ctx, punch.GetPuncher())
		}
	}()

	return nil
}

func (gc *GClient) Punch(ctx context.Context, id string) {
	var client *Client
	pc, ok := gc.hosts.Load(id)
	if !ok {
		log.Printf("client to punch to not found, asking server about client: %s", id)
		reply, err := gc.msgclient.ClientInfo(ctx, &msg.ClientInfoRequest{RequesterId: gc.id, Bulk: false, ClientId: id})
		if err != nil {
			log.Printf("error asking server about client for punch: %v", err)
			return
		}
		client.Id, err = uuid.Parse(reply.Clients[0].Uuid)
		client.TunIP = reply.Clients[0].Tunip
		client.Remote = reply.Clients[0].Remote
		gc.hosts.Store(client.Id.String(), client)
		log.Printf("client info for punch found, storing client: id: %v ip: %v remote: %v", client.Id.String(), client.TunIP, client.Remote)
	} else {
		client, ok = pc.(*Client)
		if !ok {
			log.Println("error casting found client to *Client")
			return
		}
	}

	log.Println("client info found - doing punch to remote")
	raddr, _ := net.ResolveUDPAddr("udp4", client.Remote)

	for i := 0; i < 3; i++ {
		gc.udpcon.WriteToUDP([]byte("punch"), raddr)
	}

	log.Println("punch completed")
}

func (gc *GClient) Listen(ctx context.Context) {
	rdr := bufio.NewScanner(gc.udpcon)

	for {
		rdr.Scan()
		fmt.Println(rdr.Text())
	}
}
