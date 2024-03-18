package discovery

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"

	pb "google.golang.org/protobuf/proto"

	"github.com/caldog20/overlay/pkg/header"
	controllerv1 "github.com/caldog20/overlay/proto/gen/controller/v1"
)

type DiscoveryServer struct {
	conn *net.UDPConn
}

func New(port uint16) (*DiscoveryServer, error) {
	addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		return nil, err
	}

	return &DiscoveryServer{
		conn: conn,
	}, nil
}

func (s *DiscoveryServer) Listen(ctx context.Context) error {
	buf := make([]byte, 1500)
	h := header.NewHeader()
	for {
		select {
		case <-ctx.Done():
			return nil
		default:

			n, raddr, err := s.conn.ReadFromUDP(buf)
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					return nil
				}
				log.Printf("discovery server udp socket error: %s", err)
				continue
			}
			//log.Printf("received discovery packet from %s", raddr.String())

			err = h.Parse(buf)
			if err != nil {
				log.Printf("error parsing discovery header: %s", err)
				continue
			}

			if h.Type != header.Discovery {
				log.Print("message does not have a discovery header")
				continue
			}

			_, err = parseDiscoveryMessage(buf[header.HeaderLen:n])
			if err != nil {
				log.Printf("error parsing discovery message: %s", err)
				continue
			}

			//if msg.Id == 0 {
			//	log.Printf("discovery request ID must not be zero")
			//	continue
			//}
			//_, ok := c.peerChannels.Load(msg.Id)
			//if !ok {
			//	continue
			//}

			out, err := h.Encode(buf[0:], header.Discovery, 0, 1)
			if err != nil {
				log.Printf("error encoding header for discovery reply: %s", err)
				continue
			}

			reply, err := encodeDiscoveryResponse(out, raddr.String())
			if err != nil {
				log.Printf("error encoding discovery reply message: %s", err)
				continue
			}
			_, err = s.conn.WriteToUDP(reply, raddr)
			if err != nil {
				log.Printf(
					"error sending discovery reply message to peer %s : %s",
					raddr.String(),
					err,
				)
			}
		}
	}
}

func (s *DiscoveryServer) Stop() error {
	return s.conn.Close()
}

func parseDiscoveryMessage(b []byte) (*controllerv1.EndpointDiscovery, error) {
	msg := &controllerv1.EndpointDiscovery{}
	err := pb.Unmarshal(b, msg)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func encodeDiscoveryResponse(out []byte, endpoint string) ([]byte, error) {
	msg := &controllerv1.EndpointDiscoveryResponse{Endpoint: endpoint}
	encoded, err := pb.Marshal(msg)
	if err != nil {
		return nil, err
	}
	out = append(out, encoded...)
	return out, nil
}
