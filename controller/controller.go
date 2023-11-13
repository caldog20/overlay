package controller

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/caldog20/go-overlay/ipam"
)

type Node struct {
	Uuid       string    `json:"uuid"`
	Hostname   string    `json:"hostname"`
	VpnIP      string    `json:"vpnip"`
	Remote     string    `json:"remote"`
	LastUpdate time.Time `json:"last-update"`
}

type Controller struct {
	mu    sync.Mutex
	nodes map[string]*Node
	ipman *ipam.Ipam
}

//func (s *ControlServer) Register(ctx context.Context, req *msg.RegisterRequest) (*msg.RegisterReply, error) {
//	if req.Hostname == "" {
//		return nil, errors.New("hostname must not be nil")
//	}
//
//	p, _ := peer.FromContext(ctx)
//	remote := p.Addr.String()
//
//	id, _ := uuid.NewUUID()
//
//	cip, err := s.ipman.AllocateIP(id.String(), req.Hostname)
//	if err != nil {
//		log.Println(err)
//		return nil, errors.New("error allocating IP address")
//	}
//
//	newclient := &client{
//		Id:       id.String(),
//		Hostname: req.Hostname,
//		VpnIP:    cip,
//		Remote:   strings.Split(remote, ":")[0] + ":" + req.Port,
//	}
//
//	s.clients.Store(id.String(), newclient)
//
//	return &msg.RegisterReply{
//		VpnIp: newclient.VpnIP,
//		Uuid:  newclient.Id,
//	}, nil
//}
//
//func (s *ControlServer) Deregister(ctx context.Context, req *msg.DeregisterRequest) (*emptypb.Empty, error) {
//	if req.Uuid == "" {
//		return nil, errors.New("uuid must not be nil")
//	}
//
//	//p, _ := peer.FromContext(ctx)
//	//remote := p.Addr.String()
//
//	s.clients.Delete(req.Uuid)
//
//	// change this to return a bool for found
//	ip, err := s.ipman.WhoIsByID(req.Uuid)
//	if err != nil {
//		log.Printf("ip not found: %v", err)
//	}
//
//	err = s.ipman.DeallocateIP(ip)
//	if err != nil {
//		log.Printf("cannot deallocate ip: %v", err)
//	}
//
//	return &emptypb.Empty{}, nil
//}
//
//func (s *ControlServer) WhoIsIp(ctx context.Context, req *msg.WhoIsIPRequest) (*msg.WhoIsIPReply, error) {
//	vpnip := req.GetVpnIp()
//
//	id, err := s.ipman.WhoIsByIP(vpnip)
//	if err != nil {
//		return nil, errors.New("client not found")
//	}
//
//	c, ok := s.clients.Load(id)
//	if !ok {
//		return nil, errors.New("vpn ip not found")
//	}
//
//	// check cast error here
//	client := c.(*client)
//
//	return &msg.WhoIsIPReply{
//		Remote: &msg.Remote{
//			Uuid:   client.Id,
//			VpnIp:  client.VpnIP,
//			Remote: client.Remote,
//		},
//	}, nil
//}
//
//func (s *ControlServer) RemoteList(ctx context.Context, req *msg.RemoteListRequest) (*msg.RemoteListReply, error) {
//	reqid := req.Uuid
//	_, ok := s.clients.Load(reqid)
//	if !ok {
//		return nil, errors.New("requesting client not registered")
//	}
//
//	var rl []*msg.Remote
//
//	s.clients.Range(func(k, v interface{}) bool {
//		if k.(string) != reqid {
//			r := &msg.Remote{
//				Uuid:   v.(*client).Id,
//				VpnIp:  v.(*client).VpnIP,
//				Remote: v.(*client).Remote,
//			}
//			rl = append(rl, r)
//		} else {
//			log.Printf("not sending requestor its own client info: %v", reqid)
//		}
//
//		return true
//	})
//
//	return &msg.RemoteListReply{
//		Remotes: rl,
//	}, nil
//}

func (c *Controller) addNode(hostname, remote string) (*Node, error) {
	if hostname == "" {
		return nil, errors.New("[add node] hostname must not be nil")
	}

	id := uuid.New().String()

	ip, _ := c.ipman.AllocateIP(id, hostname)

	ctime := time.Now()

	node := &Node{
		id,
		hostname,
		ip,
		remote,
		ctime,
	}

	c.mu.Lock()
	c.nodes[id] = node
	c.mu.Unlock()

	return node, nil
}

func (c *Controller) deleteNode(id string) error {
	if id == "" {
		return errors.New("[delete node] id must not be nil")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	n, found := c.nodes[id]
	if !found {
		return errors.New("[delete node] node not found")
	}

	err := c.ipman.DeallocateIP(n.VpnIP)
	if err != nil {
		return err
	}

	delete(c.nodes, id)

	return nil
}

func (c *Controller) getNodeByIP(ip string) (*Node, error) {
	id, err := c.ipman.WhoIsByIP(ip)
	if err != nil {
		return nil, errors.New("client not found")
	}

	c.mu.Lock()
	n, found := c.nodes[id]
	if !found {
		return nil, errors.New("node not found")
	}

	c.mu.Unlock()
	return n, nil
}

func RunController(ctx context.Context) {
	runtime.LockOSThread()

	i, err := ipam.NewIpam("192.168.1.0/24")
	if err != nil {
		log.Fatal(err)
	}

	cServer := &Controller{
		ipman: i,
		nodes: make(map[string]*Node),
	}

	//wg := sync.WaitGroup{}

	cServer.Serve(ctx)

}
