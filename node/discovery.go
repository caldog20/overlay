package node

import (
	"errors"
	"log"
	"net/netip"

	pb "google.golang.org/protobuf/proto"

	"github.com/caldog20/overlay/pkg/header"
	controllerv1 "github.com/caldog20/overlay/proto/gen/controller/v1"
)

func (node *Node) SendDiscoveryRequest() error {
	h := header.NewHeader()
	buf := make([]byte, 1300)
	encoded, err := h.Encode(buf, header.Discovery, node.id, 0)
	if err != nil {
		return err
	}

	dis := &controllerv1.EndpointDiscovery{Id: node.id}
	out, err := pb.Marshal(dis)
	encoded = append(encoded, out...)
	if err != nil {
		return err
	}

	n, err := node.conn.WriteToUDP(encoded, node.controllerDiscoveryEndpoint)
	if err != nil {
		return err
	}
	log.Printf(
		"sent discovery packet to controller - %d bytes - address :%s",
		n,
		node.controllerDiscoveryEndpoint.String(),
	)
	return nil
}

func (node *Node) HandleDiscoveryResponse(buffer *InboundBuffer) error {
	reply := &controllerv1.EndpointDiscoveryResponse{}
	err := pb.Unmarshal(buffer.in[header.HeaderLen:buffer.size], reply)
	if err != nil {
		return err
	}
	log.Printf("received discovery response from controller: %s", reply.Endpoint)
	// TODO fix this
	endpoint, err := netip.ParseAddrPort(reply.Endpoint)
	if err != nil {
		return err
	}

	return node.maybeUpdateEndpoint(endpoint)
}

func (node *Node) maybeUpdateEndpoint(endpoint netip.AddrPort) error {
	if node.discoveredEndpoint.Compare(endpoint) != 0 {
		err := node.SetRemoteEndpoint(endpoint.String())
		if err != nil {
			return errors.New("error updating node endpoint to controller")
		}
		node.discoveredEndpoint = endpoint
	}
	return nil
}
