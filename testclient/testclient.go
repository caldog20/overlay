package testclient

import (
	"bufio"
	"context"
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
	log.Printf("punching to remote %v", remote)
	laddr, _ := net.ResolveUDPAddr("udp4", ":5050")
	conn, err := net.ListenUDP("udp4", laddr)
	raddr, _ := net.ResolveUDPAddr("udp4", remote+":"+"5050")
	if err != nil {
		log.Fatal(err)
	}
	r := bufio.NewReader(conn)
	conn.WriteToUDP([]byte("punch"), raddr)
	for {
		s, err := r.ReadString(0xff)
		if err != nil {
			log.Fatal(err)
		}
		log.Println(s)
		if s == "goodbye" {
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

	//pb := []byte("punchout")

	if doPunch {
		time.Sleep(time.Second * 5)
		// Write a few packets out first
		remote := strings.Split(clients[0].Remote, ":")[0]
		log.Printf("requesting punch to remote %v", remote)
		laddr, _ := net.ResolveUDPAddr("udp4", ":5050")
		conn, err := net.ListenUDP("udp4", laddr)
		raddr, _ := net.ResolveUDPAddr("udp4", remote+":"+"5050")
		if err != nil {
			log.Fatal(err)
		}

		// Send Punch Request to client
		client.Punch(ctx, &msg.PunchRequest{RequestorId: testclient.Uuid, PuncheeId: clients[0].Uuid})
		log.Println("sent punch request, starting to write data to remote")
		// wait a few seconds
		time.Sleep(time.Second * 3)
		// Write more data
		conn.WriteToUDP([]byte("hellohellohello"), raddr)
		conn.WriteToUDP([]byte{0xff}, raddr)
		conn.WriteToUDP([]byte("hellohellohello"), raddr)
		conn.WriteToUDP([]byte{0xff}, raddr)
		conn.WriteToUDP([]byte("hellohellohello"), raddr)
		conn.WriteToUDP([]byte{0xff}, raddr)
		conn.WriteToUDP([]byte("goodbye"), raddr)
		conn.WriteToUDP([]byte{0xff}, raddr)
	}

	wg.Wait()

	// Deregister and quit

}
