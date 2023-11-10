package testclient

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/caldog20/go-overlay/msg"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var clients []*msg.ClientInfoReply_Client

func udptest(wg *sync.WaitGroup, id string) {
	defer wg.Done()
	remote := strings.Split(clients[0].Remote, ":")[0]
	conn, err := net.Dial("udp4", remote+":"+"5050")
	if err != nil {
		log.Fatal(err)
	}
	conn.Write([]byte("punchout"))
	conn.Write([]byte("punchout"))
	conn.Write([]byte("punchout"))
	for {
		buf := make([]byte, 500)
		n, _ := conn.Read(buf)
		log.Printf("read bytes from socket %v", n)
		log.Print(string(buf[:n]))

		if fmt.Sprint(buf[:n]) == "goodbye" {
			return
		}
	}
}

func Run(ctx context.Context, caddr string, doPunch bool) {
	var wg sync.WaitGroup

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

		time.Sleep(time.Second * 5)
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				punch, err := pclient.Recv()
				if err != nil {
					pclient = nil
					log.Printf("punch client stream read error")
					return
				}
				log.Println("Received punch notification, doing punch out")
				wg.Add(1)
				go udptest(&wg, punch.GetPuncher())
			}
		}()
	}

	for {
		// Request info about other connected clients
		ciresponse, rerr := client.ClientInfo(ctx, &msg.ClientInfoRequest{RequesterId: testclient.Uuid})
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

	pb := []byte("punchout")
	in := []byte("hellohellohello\n")
	if doPunch {
		time.Sleep(time.Second * 5)
		// Write a few packets out first
		remote := strings.Split(clients[0].Remote, ":")[0]
		conn, err := net.Dial("udp4", remote+":"+"5050")
		if err != nil {
			log.Fatal(err)
		}
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

	wg.Wait()

	// Deregister and quit

}
