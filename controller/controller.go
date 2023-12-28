package controller

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/twitchtv/twirp"

	"github.com/caldog20/overlay/proto"
)

const (
	Subnet            = "100.65.0."
	RemoteAddr string = "remote-address"
)

type Controller struct {
	db      *DB
	ipam    map[netip.Addr]struct{}
	ipCount atomic.Uint64
	updates struct {
		changes []uint32
		deletes []uint32
	}

	punches struct {
		mu      sync.RWMutex
		pending map[uint32]string
	}
}

func WithRemoteAddr(base http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ra := r.RemoteAddr
		ra = strings.Split(ra, ":")[0]
		ctx = context.WithValue(ctx, RemoteAddr, ra)
		r = r.WithContext(ctx)
		base.ServeHTTP(w, r)
	})
}

func NewController() *Controller {
	c := new(Controller)
	c.db = NewDB()
	c.ipam = make(map[netip.Addr]struct{})
	c.ipCount.Store(1)

	c.punches.pending = make(map[uint32]string)

	return c
}

func (c *Controller) DiscoveryServer(ctx context.Context) {
	addr, _ := net.ResolveUDPAddr("udp4", ":7979")
	s, _ := net.ListenUDP("udp4", addr)

	buf := make([]byte, 100)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		s.SetReadDeadline(time.Now().Add(time.Second * 2))
		_, raddr, err := s.ReadFromUDP(buf)
		if err != nil {
			continue
		}
		if binary.BigEndian.Uint32(buf[:4]) != 8675309 {
			continue
		}
		s.WriteToUDP([]byte(raddr.String()), raddr)
	}
}

func (c *Controller) RunController(ctx context.Context, port string) {
	e := echo.New()
	//e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	server := proto.NewControllerServer(c)
	handler := WithRemoteAddr(server)

	e.Any("/twirp*", echo.WrapHandler(handler))

	go c.DiscoveryServer(ctx)

	go func() {
		if err := e.Start(fmt.Sprintf(":%s", port)); err != nil && err != http.ErrServerClosed {
			e.Logger.Fatal("shutting down the server")
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(shutdownCtx); err != nil {
		e.Logger.Fatal(err)
	}
}

func (c *Controller) Register(ctx context.Context, req *proto.RegisterRequest) (*proto.RegisterResponse, error) {
	key := req.GetKey()
	if key == "" {
		return nil, twirp.InvalidArgumentError("key", "key cannot be nil")
	}

	//if req.Port == 0 || req.Port >= 65535 {
	//	return nil, twirp.InvalidArgumentError("port", "invalid port")
	//}

	// if req.Endpoint == "" {

	//}

	raddr, err := netip.ParseAddrPort(req.Endpoint)
	if err != nil {
		// Discovery failed, we only received port
		// Use RemoteAddr from http request and append port for Endpoint
		ra := ctx.Value("remote-address").(string)
		raddr = netip.MustParseAddrPort(ra + req.Endpoint)
	}

	node, err := c.db.GetNodeByKey(key)
	if err != nil {
		// Node not found
		node = c.NewNode(key, req.Hostname, raddr)
		err = c.db.AddNode(node)
		if err != nil {
			return nil, twirp.InternalError(err.Error())
		}
	}
	node.mu.Lock()
	defer node.mu.Unlock()

	node.EndPoint = raddr
	node.timestamp = time.Now()
	node.Hostname = req.Hostname

	resp := node.RegisterResponseProto()

	log.Printf("registered node: node id: %d - node vpnip: %s - node raddr: %s", node.ID, node.VpnIP.String(), node.EndPoint.String())

	return resp, nil
}

//func (c *Controller) GetUpdate(ctx context.Context, req *proto.UpdateRequest) (*proto.UpdateResponse, error) {
//	if req.Id == 0 {
//		return nil, twirp.InvalidArgumentError("id", "id must not be zero")
//	}
//
//	// Handle updating node endpoint somwhow
//	//node, err := c.db.GetNodeByID(req.Id)
//	//if err != nil {
//	//	return nil, twirp.InternalError("node not registered")
//	//}
//
//	// for now just send node lists
//	c.db.l.RLock()
//	defer c.db.l.RUnlock()
//
//	var nodes []*proto.Node
//	for _, n := range c.db.id {
//		node := &proto.Node{
//			Id:       n.ID,
//			Ip:       n.VpnIP.String(),
//			Hostname: n.Hostname,
//			Endpoint: n.EndPoint.String(),
//			Key:      n.NodeKey,
//		}
//		nodes = append(nodes, node)
//	}
//
//	//resp := &proto.UpdateResponse{
//	//	Type: proto.UpdateType_NODES,
//	//	Update: &proto.UpdateResponse_NodeUpdate{
//	//		&proto.NodeUpdate{
//	//			Nodes: nodes,
//	//		},
//	//	},
//	//}
//
//	return nil, nil
//}

func (c *Controller) NodeList(ctx context.Context, req *proto.NodeListRequest) (*proto.NodeListResponse, error) {
	if req.Id == 0 {
		return nil, twirp.InvalidArgumentError("id", "id must not be zero")
	}

	_, err := c.db.GetNodeByID(req.Id)
	if err != nil {
		return nil, twirp.InternalError("requesting node not found")
	}

	var nodes []*proto.Node
	var count int

	c.db.l.RLock()
	defer c.db.l.RUnlock()
	for id, node := range c.db.id {
		if id == req.Id {
			continue
		}
		nodes = append(nodes, node.Proto())
		count++
	}

	resp := NodeListProto(count, nodes)

	return resp, nil
}

func (c *Controller) NodeQuery(ctx context.Context, req *proto.NodeQueryRequest) (*proto.Node, error) {
	if req.ReqId == 0 {
		return nil, twirp.InvalidArgumentError("id", "id must not be zero")
	}

	_, err := c.db.GetNodeByID(req.ReqId)
	if err != nil {
		return nil, twirp.InternalError("requesting node not found")
	}

	var node *Node
	var resp *proto.Node

	if req.NodeIp != nil {
		ip := req.GetNodeIp()
		node, err = c.db.GetNodeByIP(netip.MustParseAddr(ip))
		if err != nil {
			return nil, twirp.NotFoundError("node IP not found")
		}
	} else if req.NodeId != nil {
		id := req.GetNodeId()
		node, err = c.db.GetNodeByID(id)
		if err != nil {
			return nil, twirp.NotFoundError("node ID not found")
		}
	}

	if node == nil {
		return nil, twirp.InternalError("Error processing request, node is nil")
	}

	resp = node.Proto()

	return resp, nil
}

func (c *Controller) PunchRequester(ctx context.Context, req *proto.PunchRequest) (*proto.PunchReply, error) {
	if req.ReqId == 0 {
		return nil, twirp.InvalidArgumentError("id", "id must not be zero")
	}
	requestor, err := c.db.GetNodeByID(req.ReqId)
	if err != nil {
		return nil, twirp.InternalError("requesting node not found")
	}

	puncher, err := c.db.GetNodeByID(req.RemoteId)
	if err != nil {
		return nil, twirp.InternalError("remote node not found")
	}

	remote := requestor.EndPoint.String()

	c.punches.mu.Lock()
	defer c.punches.mu.Unlock()

	c.punches.pending[puncher.ID] = remote

	return &proto.PunchReply{
		Status: true,
	}, nil
}

func (c *Controller) PunchChecker(ctx context.Context, req *proto.PunchCheck) (*proto.Punch, error) {
	if req.ReqId == 0 {
		return nil, twirp.InvalidArgumentError("id", "id must not be zero")
	}
	_, err := c.db.GetNodeByID(req.ReqId)
	if err != nil {
		return nil, twirp.InternalError("requesting node not found")
	}

	c.punches.mu.RLock()
	remote, found := c.punches.pending[req.ReqId]
	c.punches.mu.RUnlock()

	if !found {
		return nil, twirp.NotFoundError("pending punch for requesting ID not found")
	}

	c.punches.mu.Lock()
	defer c.punches.mu.Unlock()
	delete(c.punches.pending, req.ReqId)

	return &proto.Punch{
		Remote: remote,
	}, nil
}

func (c *Controller) NewNode(key string, hostname string, raddr netip.AddrPort) *Node {
	node := new(Node)
	node.ID = c.db.GenerateID()
	node.VpnIP = c.AllocateIP()
	node.NodeKey = key
	node.Hostname = hostname
	node.EndPoint = raddr

	return node
}

func (c *Controller) AllocateIP() netip.Addr {
	octet := c.ipCount.Load()
	c.ipCount.Add(1)
	ip := netip.MustParseAddr(Subnet + strconv.FormatUint(octet, 10))

	return ip
}
