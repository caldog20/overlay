package controller

import (
	"net/netip"
	"strconv"
)

//func (node *Node) Proto() *proto.Node {
//	return &proto.Node{
//		Id:       node.ID,
//		Ip:       node.VpnIP.String(),
//		Hostname: node.Hostname,
//		Endpoint: node.EndPoint.String(),
//		Key:      node.NodeKey,
//	}
//}
//
//func (node *Node) RegisterResponseProto() *proto.RegisterResponse {
//	return &proto.RegisterResponse{
//		Id: node.ID,
//		Ip: node.VpnIP.String(),
//	}
//}
//
//func NodeListProto(count int, nodes []*proto.Node) *proto.NodeListResponse {
//	return &proto.NodeListResponse{
//		Count: uint32(count),
//		Nodes: nodes,
//	}
//}

func (c *Controller) AllocateIP() netip.Addr {
	octet := c.ipCount.Add(1)
	ip := netip.MustParseAddr(Subnet + strconv.FormatUint(octet, 10))

	return ip
}
