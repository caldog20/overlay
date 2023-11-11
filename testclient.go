package main

import (
	"context"
	"errors"
	"log"
	"net"
	"sync"
	"time"

	"github.com/caldog20/go-overlay/firewall"
	"github.com/caldog20/go-overlay/msg"
	"github.com/caldog20/go-overlay/tun"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type remoteHost struct {
	VpnIP  string
	Remote string
	Addr   *net.UDPAddr
	Id     string
}

type GClient struct {
	hosts     sync.Map
	udpcon    *net.UDPConn
	gconn     *grpc.ClientConn
	msgclient msg.ControlServiceClient
	tun       *tun.Tun
	id        string
	vpnip     string
	fw        *firewall.Firewall
	hostname  string
}

func RunClient(ctx context.Context, caddr string, hostname string) {
	log.SetPrefix("client: ")

	t, err := tun.NewTun()
	if err != nil {
		log.Fatal(err)
	}

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

	gclient := &GClient{
		hosts:     sync.Map{},
		udpcon:    uc,
		gconn:     conn,
		msgclient: mc,
		tun:       t,
		id:        "",
		vpnip:     "",
		fw:        firewall.NewFirewall(),
		hostname:  hostname,
	}

	err = gclient.Register(ctx)
	if err != nil {
		log.Fatal(err)
	}

	<-ctx.Done()
	gclient.udpcon.Close()
	gclient.gconn.Close()

}

func (gc *GClient) QueryRemotes(ctx context.Context) {
QUERY:
	for {
		query, err := gc.msgclient.RemoteList(ctx, &msg.RemoteListRequest{
			Uuid: gc.id,
		})
		if err != nil {
			log.Fatal(err)
		}

		remotes := query.Remotes
		if len(remotes) < 1 {
			goto STANDOFF
		}

		var rh *remoteHost
		for _, v := range remotes {
			r, ok := gc.hosts.Load(v.VpnIp)
			if ok {
				rh = r.(*remoteHost)
				rh.VpnIP = v.VpnIp
				rh.Id = v.Uuid
				rh.Remote = v.Remote
				rh.Addr, err = net.ResolveUDPAddr("udp4", rh.Remote)
				if err != nil {
					log.Printf("error resolving raddr for adding host to list: %v", err)
					goto STANDOFF
				}
			} else {
				rh = &remoteHost{}
				rh.VpnIP = v.VpnIp
				rh.Id = v.Uuid
				rh.Remote = v.Remote
				rh.Addr, err = net.ResolveUDPAddr("udp4", rh.Remote)
				log.Printf("error resolving raddr for adding host to list: %v", err)
				if err != nil {
					log.Printf("error resolving raddr for adding host to list: %v", err)
					goto STANDOFF
				}
			}
			count := len(remotes)
			log.Println("updating remote client list - count :%d", count)
			gc.hosts.Store(rh.VpnIP, rh)
			goto STANDOFF
		}
	}

STANDOFF:
	for {
		time.Sleep(time.Second * 5)
		goto QUERY
	}
}

func (gc *GClient) Register(ctx context.Context) error {
	// Register Client
	reply, err := gc.msgclient.Register(ctx, &msg.RegisterRequest{Hostname: gc.hostname, Port: "2222"})
	if err != nil {
		log.Printf("error sending/recv message: %v", err)
		return errors.New("failed to register with controller")
	}

	//if !reply.GetSuccess() {
	//	return errors.New("failed to register with controller")
	//}

	gc.vpnip = reply.VpnIp
	gc.id = reply.Uuid

	log.Println("User registered successfully")
	log.Printf("uuid: %s - vpnip: %s", gc.id, gc.vpnip)

	err = gc.tun.ConfigureInterface(net.ParseIP(gc.vpnip))
	if err != nil {
		log.Fatal(err)
	}

	gc.RunTunnel(ctx)

	return nil
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
				r := remote.(*remoteHost)
				n, err = gc.udpcon.WriteToUDP(in[:n], r.Addr)
				log.Printf("wrote %v bytes to remote: %s", n, r.Addr.String())
			} else {
				// query server about client, ask for punch, and store client
				qresp, qerr := gc.msgclient.WhoIsIp(ctx, &msg.WhoIsIPRequest{VpnIp: fwpacket.RemoteIP.String()})
				if qerr != nil {
					log.Println(qerr)
					continue
				}
				// Got client info back, store and try again next time
				newremote := &remoteHost{}
				newremote.VpnIP = qresp.Remote.VpnIp
				newremote.Id = qresp.Remote.Uuid
				newremote.Remote = qresp.Remote.Remote
				newremote.Addr, err = net.ResolveUDPAddr("udp4", qresp.Remote.Remote)
				if err != nil {
					log.Printf("erroring resolving udp address for new client: %s - %v", newremote.Remote, err)
				} else {
					gc.hosts.Store(newremote.VpnIP, newremote)
				}

				select {
				case <-ctx.Done():
					gc.udpcon.Close()
					gc.tun.Close()
					return
				default:
					continue
				}
			}
		}
	}()

	go func() {
		in := make([]byte, 1308)
		fwpacket := &firewall.FWPacket{}
		for {
			n, raddr, err := gc.udpcon.ReadFromUDP(in)
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("read %v bytes from udp remote: %s", n, raddr.String())
			fwpacket, err = gc.fw.Parse(in[:n], true)
			if err != nil {
				log.Println(err)
				continue
			}
			drop := gc.fw.Drop(fwpacket)
			if drop {
				log.Println("fw dropping packet")
				continue
			}

			// check for real remote here, roaming later

			n, err = gc.tun.Write(in[:n])
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("wrote %v bytes to tunnel", n)
		}
	}()

}
