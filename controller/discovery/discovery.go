package discovery

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	controllerv1 "github.com/caldog20/overlay/proto/gen/controller/v1"
	pb "google.golang.org/protobuf/proto"
)

func StartDiscoveryServer(ctx context.Context, port uint16) error {
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
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		conn.SetReadDeadline(time.Now().Add(time.Second * 2))
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
		//if msg.Id == 0 {
		//	log.Printf("discovery request ID must not be zero")
		//	continue
		//}
		//_, ok := c.peerChannels.Load(msg.Id)
		//if !ok {
		//	continue
		//}

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

func parseDiscoveryMessage(b []byte) (*controllerv1.EndpointDiscovery, error) {
	msg := &controllerv1.EndpointDiscovery{}
	err := pb.Unmarshal(b, msg)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func encodeDiscoveryResponse(endpoint string) ([]byte, error) {
	msg := &controllerv1.EndpointDiscoveryResponse{Endpoint: endpoint}
	return pb.Marshal(msg)
}
