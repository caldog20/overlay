package controller

import (
	"fmt"
	"log"
	"net"

	"github.com/caldog20/overlay/proto"
	pb "google.golang.org/protobuf/proto"
)

func (c *Controller) StartDiscoveryServer() error {
	laddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf(":%d", c.config.DiscoveryPort))
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp4", laddr)
	if err != nil {
		return err
	}

	c.discovery = conn

	return c.runDiscoveryServer()
}

func (c *Controller) runDiscoveryServer() error {
	buf := make([]byte, 1500)
	conn := c.discovery
	for {
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

func parseDiscoveryMessage(b []byte) (*proto.DiscoverEndpoint, error) {
	msg := &proto.DiscoverEndpoint{}
	err := pb.Unmarshal(b, msg)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func encodeDiscoveryResponse(endpoint string) ([]byte, error) {
	msg := &proto.Endpoint{Endpoint: endpoint}
	return pb.Marshal(msg)
}
