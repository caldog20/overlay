package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/caldog20/go-overlay/firewall"
	"github.com/caldog20/go-overlay/msg"
	"github.com/caldog20/go-overlay/tun"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type GClient struct {
	hosts     sync.Map
	udpcon    *net.UDPConn
	gconn     *grpc.ClientConn
	msgclient msg.ControlServiceClient
	tun       *tun.Tun
	id        string
	vpnip     string
	fw        *firewall.Firewall
}

func RunClient(ctx context.Context, caddr string, hostname string) {
	log.SetPrefix("client: ")

	gclient := &GClient{
		fw: firewall.NewFirewall(),
	}

	t, err := tun.NewTun()
	if err != nil {
		log.Fatal(err)
	}

	gclient.tun = t

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

	err = gclient.Register(ctx, hostname)
	if err != nil {
		log.Fatal(err)
	}

	err = gclient.Subscribe(ctx)

	go gclient.Listen(ctx)

	//pb := []byte("punchout")

	//if doPunch {
	//	client := &Client{}
	//	for {
	//		// Request info about other connected clients
	//		ciresponse, err := gclient.msgclient.ClientInfo(ctx, &msg.ClientInfoRequest{RequesterId: gclient.id, VpnIp: "192.168.1.1"})
	//		if err != nil {
	//			log.Printf("client not found maybe: %v", err)
	//			continue
	//		}
	//
	//		if ciresponse.Tunip != "192.168.1.1" {
	//			log.Printf("got wrong vpnip: %v", ciresponse.Tunip)
	//		}
	//
	//		client.VpnIP = ciresponse.Tunip
	//		client.Remote = ciresponse.Remote
	//		client.Id = uuid.MustParse(ciresponse.Uuid)
	//		gclient.hosts.Store(client.Id.String(), client)
	//		break
	//	}
	//
	//	// Write a few packets out first
	//	log.Printf("requesting punch to remote %v", client.Remote)
	//	raddr, _ := net.ResolveUDPAddr("udp4", client.Remote)
	//
	//	// Send Punch Request to client
	//	_, err = gclient.msgclient.Punch(ctx, &msg.PunchRequest{RequestorId: gclient.id, PuncheeId: client.Id.String()})
	//	if err != nil {
	//		log.Fatal(err)
	//	}
	//	log.Println("sent punch request, starting to write data to remote")
	//	// wait a few seconds
	//	time.Sleep(time.Second * 3)
	//	// Write more data
	//	gclient.udpcon.WriteToUDP([]byte("hello\n"), raddr)
	//	gclient.udpcon.WriteToUDP([]byte("punch worked\n"), raddr)
	//	gclient.udpcon.WriteToUDP([]byte("goodbye\n"), raddr)
	//}

	<-ctx.Done()
	gclient.udpcon.Close()
	gclient.gconn.Close()

}

func (gc *GClient) Register(ctx context.Context, username string) error {
	// Register Client
	reply, err := gc.msgclient.Register(ctx, &msg.RegisterRequest{Hostname: username, Port: "2222"})
	if err != nil {
		log.Printf("error sending/recv message: %v", err)
		return errors.New("failed to register with controller")
	}

	//if !reply.GetSuccess() {
	//	return errors.New("failed to register with controller")
	//}

	gc.vpnip = reply.VpnIp

	log.Println("User registered successfully")
	log.Printf("vpnip: %s", gc.vpnip)

	err = gc.tun.ConfigureInterface(net.ParseIP(gc.vpnip))
	if err != nil {
		log.Fatal(err)
	}

	gc.RunTunnel(ctx)

	return nil
}

func (gc *GClient) Subscribe(ctx context.Context) error {
	// Subscribe to puncher service
	puncher, err := gc.msgclient.PunchNotifier(ctx, &msg.PunchSubscribe{VpnIp: gc.vpnip})
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
			log.Printf("Received punch notification for client: %s", punch.GetRemote())
			gc.Punch(ctx, punch.GetRemote())
		}
	}()

	return nil
}

func (gc *GClient) Punch(ctx context.Context, remote string) {
	//client := &Client{}
	//pc, ok := gc.hosts.Load(id)
	//if !ok {
	//	log.Printf("client to punch to not found, asking server about client: %s", id)
	//	reply, err := gc.msgclient.ClientInfo(ctx, &msg.ClientInfoRequest{RequesterId: gc.id, Uuid: id})
	//	if err != nil {
	//		log.Printf("error asking server about client for punch: %v", err)
	//		return
	//	}
	//	log.Printf("client response id: %s", reply.Uuid)
	//	client.Id, err = uuid.Parse(reply.Uuid)
	//	if err != nil {
	//		log.Fatal(err)
	//	}
	//	client.VpnIP = reply.Tunip
	//	client.Remote = reply.Remote
	//	gc.hosts.Store(client.Id.String(), client)
	//	log.Printf("client info for punch found, storing client: id: %v ip: %v remote: %v", client.Id.String(), client.VpnIP, client.Remote)
	//} else {
	//	client, ok = pc.(*Client)
	//	if !ok {
	//		log.Println("error casting found client to *Client")
	//		return
	//	}
	//}

	//log.Println("client info found - doing punch to remote")
	raddr, _ := net.ResolveUDPAddr("udp4", remote)

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

func (gc *GClient) RunTunnel(ctx context.Context) {

	go func() {
		in := make([]byte, 1308)
		//out := make([]byte, 1308)
		//h := msg.Header{}
		fwpacket := &firewall.FWPacket{}
		for {
			n, err := gc.tun.Read(in)
			log.Printf("read %v bytes from  tunnel", n)
			if err != nil {
				log.Fatal(err)
			}

			fwpacket, err = gc.fw.Parse(in[:n], false)
			if err != nil {
				log.Println(err)
				continue
			}

			drop := gc.fw.Drop(fwpacket)
			if drop {
				log.Println("fw dropping packet")
				continue
			}

			// We want to send the packet
			// check to see if we have active tunnel with this address
			remote, found := gc.hosts.Load(fwpacket.RemoteIP.String())
			if found {
				r := remote.(*net.UDPAddr)
				n, err = gc.udpcon.WriteToUDP(in[:n], r)
				log.Printf("wrote %v bytes to remote: %s", n, r.String())
			} else {
				// query server about client, ask for punch, and store client
				qresp, qerr := gc.msgclient.WhoIs(ctx, &msg.WhoIsIP{VpnIp: fwpacket.RemoteIP.String()})
				if qerr != nil {
					log.Println(qerr)
					continue
				}
				raddr, rerr := net.ResolveUDPAddr("udp4", qresp.Remote)
				if rerr != nil {
					log.Println(rerr)
					continue
				}
				// ask for punch and try again next time
				// Send Punch Request to client
				_, err = gc.msgclient.Punch(ctx, &msg.PunchRequest{SrcVpnIp: gc.vpnip, DstVpnIp: fwpacket.RemoteIP.String()})
				if err != nil {
					log.Println(err)
					continue
				}
				// We sent punch, store client now
				gc.hosts.Store(fwpacket.RemoteIP.String(), raddr)

				select {
				case <-ctx.Done():
					gc.udpcon.Close()
					gc.tun.Close()
					return
				default:
				}
			}
		}
	}()

	go func() {
		in := make([]byte, 1308)
		fwpacket := &firewall.FWPacket{}
		for {
			n, err := gc.udpcon.Read(in)
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("read %v bytes from udp", n)
			fwpacket, err = gc.fw.Parse(in[:n], true)
			drop := gc.fw.Drop(fwpacket)
			if drop {
				log.Println("fw dropping packet")
				continue
			}

			n, err = gc.tun.Write(in[:n])
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("wrote %v bytes to tunnel", n)
		}
	}()

}
