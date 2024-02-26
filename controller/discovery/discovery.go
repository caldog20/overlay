package discovery

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	apiv1 "github.com/caldog20/overlay/proto/gen/api/v1"
	pb "google.golang.org/protobuf/proto"
)

func StartDiscoveryServer(ctx context.Context, port uint16) error {
	log.Printf("starting discovery listener on port: %d", port)
	laddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp4", laddr)
	if err != nil {
		return err
	}

	buf := make([]byte, 1500)

	for {
		if ctxDone(ctx) {
			return nil
		}

		conn.SetReadDeadline(time.Now().Add(time.Second * 3))
		n, raddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			return nil
		}
		log.Printf("received discovery packet from %s", raddr.String())

		_, err = parseDiscoveryMessage(buf[:n])
		if err != nil {
			log.Printf("error parsing discovery message: %s", err)
			continue
		}

		reply, err := encodeDiscoveryResponse(raddr.String())
		if err != nil {
			log.Printf("error encoding discovery reply message: %s", err)
			continue
		}
		_, err = conn.WriteToUDP(reply, raddr)
		if err != nil {
			log.Printf("error sending discovery reply message to peer %s : %s", raddr.String(), err)
		}
	}
}

func parseDiscoveryMessage(b []byte) (*apiv1.EndpointDiscovery, error) {
	msg := &apiv1.EndpointDiscovery{}
	err := pb.Unmarshal(b, msg)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func encodeDiscoveryResponse(endpoint string) ([]byte, error) {
	msg := &apiv1.EndpointDiscoveryResponse{Endpoint: endpoint}
	return pb.Marshal(msg)
}

func ctxDone(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}
