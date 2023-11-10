package testclient

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/caldog20/go-overlay/msg"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var clients []*msg.ClientInfoReply_Client

func udptest(id string) {
	remote := strings.Split(clients[0].Remote, ":")[0]
	conn, _ := net.Dial("udp4", remote+":"+"5050")
	conn.Write([]byte("punchout"))
	conn.Write([]byte("punchout"))
	conn.Write([]byte("punchout"))
	for {
		buf := make([]byte, 500)
		n, _ := conn.Read(buf)
		s := fmt.Sprintf(string(buf[:n]))
		if s == "goodbye" {
			return
		}
	}
}

func Run(ctx context.Context, caddr string, doPunch bool) {
	log.SetPrefix("client: ")
	conn, err := grpc.DialContext(ctx, caddr, grpc.WithBlock(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("error connecting to grpc server: %v", err)
	}
	defer conn.Close()

	client := msg.NewControlServiceClient(conn)

	// Register Client
	testclient, rerr := client.Register(ctx, &msg.RegisterRequest{User: "simp1"})
	if rerr != nil {
		log.Printf("error sending/recv message: %v", rerr)
		return
	}
	log.Println(testclient)

	// Subscribe to puncher service
	if !doPunch {
		pclient, perr := client.PunchNotifier(ctx, &msg.PunchSubscribe{RequestorId: testclient.Uuid})
		if perr != nil {
			log.Fatal(perr)
		}

		go func() {
			for {
				punch, err := pclient.Recv()
				if err != nil {
					pclient = nil
					log.Printf("punch client stream read error")
					return
				}
				log.Println("Received punch notification, doing punch out")
				go udptest(punch.GetPuncher())
			}
		}()
	}

	// Request info about other connected clients
	ciresponse, rerr := client.ClientInfo(ctx, &msg.ClientInfoRequest{RequesterId: testclient.Uuid})
	if rerr != nil {
		log.Printf("error sending/recv message: %v", rerr)
		return
	}

	clients = ciresponse.Clients

	pb := []byte("punchout")
	in := []byte("hellohellohello\n")
	if doPunch {
		// Write a few packets out first
		remote := strings.Split(clients[0].Remote, ":")[0]
		conn, _ := net.Dial("udp4", remote+":"+"5050")
		conn.Write(pb)
		// Send Punch Request to client
		client.Punch(ctx, &msg.PunchRequest{RequestorId: testclient.Uuid, PuncheeId: clients[0].Uuid})
		log.Println("sent punch request, starting to write data to remote")
		// wait a few seconds
		time.Sleep(time.Second * 3)
		// Write more data
		conn.Write(in)
		conn.Write(in)
		conn.Write(in)
		conn.Write(in)
		conn.Write(in)
		conn.Write(in)
		conn.Write(in)
		conn.Write(in)
		conn.Write([]byte("goodbye"))
	}

	// Deregister and quit

}
