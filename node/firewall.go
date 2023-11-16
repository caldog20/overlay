package node

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"

	"github.com/rcrowley/go-metrics"
	"golang.org/x/net/ipv4"
)

const (
	ProtocolTCP  = 6
	ProtocolUDP  = 17
	ProtocolICMP = 1
)

type FWPacket struct {
	Src      net.IP
	Dst      net.IP
	SrcPort  uint16
	DstPort  uint16
	Protocol int
	// For tracking VPN stuff
	LocalIP  net.IP
	RemoteIP net.IP

	inbound bool
}

type FWMetrics struct {
	packets metrics.Counter
	allowed metrics.Counter
	dropped metrics.Counter
}

type Firewall struct {
	inboundMetrics  FWMetrics
	outboundMetrics FWMetrics
}

func NewFirewall() *Firewall {
	fw := &Firewall{
		inboundMetrics: FWMetrics{
			packets: metrics.GetOrRegisterCounter("firewall.inbound.total_packets", nil),
			allowed: metrics.GetOrRegisterCounter("firewall.inbound.allowed_packets", nil),
			dropped: metrics.GetOrRegisterCounter("firewall.inbound.dropped_packets", nil),
		},
		outboundMetrics: FWMetrics{
			packets: metrics.GetOrRegisterCounter("firewall.outbound.total_packets", nil),
			allowed: metrics.GetOrRegisterCounter("firewall.outbound.allowed_packets", nil),
			dropped: metrics.GetOrRegisterCounter("firewall.outbound.dropped_packets", nil),
		},
	}

	return fw
}

func (fw *Firewall) Parse(packet []byte, inbound bool) (*FWPacket, error) {
	h, err := ipv4.ParseHeader(packet)

	if len(packet) < ipv4.HeaderLen {
		return nil, errors.New("[firewall] packet length < ipv4.HeaderLen")
	}

	if err != nil {
		return nil, errors.New("[firewall] error parsing ipv4 header")
	}

	//if len(packet) < h.TotalLen {
	//	return nil, errors.New(fmt.Sprintf("[firewall] packet length too short: %v - header packet length %v\n", len(packet), h.TotalLen))
	//}

	var srcPort uint16
	var dstPort uint16
	var localIP net.IP
	var remoteIP net.IP

	if h.Protocol == ProtocolICMP {
		srcPort = 0
		dstPort = 0
	} else {
		srcPort = binary.BigEndian.Uint16(packet[h.Len : h.Len+2])
		dstPort = binary.BigEndian.Uint16(packet[h.Len+2 : h.Len+4])
	}

	if inbound {
		remoteIP = h.Src
		localIP = h.Dst
	} else {
		localIP = h.Src
		remoteIP = h.Dst
	}

	fwpacket := &FWPacket{
		Src:      h.Src,
		Dst:      h.Dst,
		SrcPort:  srcPort,
		DstPort:  dstPort,
		Protocol: h.Protocol,
		LocalIP:  localIP,
		RemoteIP: remoteIP,
		inbound:  inbound,
	}

	return fwpacket, nil
}

func (fw *Firewall) Drop(fwpacket *FWPacket) bool {
	// Function here eventually to compare destination with advertised routes
	//if dstNet[0] != "192" {
	//	log.Printf("[inside] packet destination not in vpn routes, dropping packet: src: %v dst: %v\n", ip4Header.Src.String(), ip4Header.Dst.String())
	//	continue
	//}
	if fwpacket == nil {
		return false
	}

	// For now, the firewall only checks to make sure packets come from the nebula vpn subnet
	if fwpacket.inbound {
		fw.inboundMetrics.packets.Inc(1)
		if fwpacket.RemoteIP.To4()[0] != 192 {
			fw.inboundMetrics.dropped.Inc(1)
			//log.Printf("[firewall.inbound.drop] packet not sourced from vpn ip - src %v", fwpacket.RemoteIP.String())
			return true
		}
		if fwpacket.LocalIP.To4()[0] != 192 {
			fw.inboundMetrics.dropped.Inc(1)
			//log.Printf("[firewall.inbound.drop] packet not destined to vpn ip - dst %v", fwpacket.LocalIP.String())
			return true
		}
		fw.inboundMetrics.allowed.Inc(1)
	} else {
		fw.outboundMetrics.packets.Inc(1)
		if fwpacket.LocalIP.To4()[0] != 192 {
			fw.outboundMetrics.dropped.Inc(1)
			//log.Printf("[firewall.outbound.drop] packet not sourced from vpn ip - src %v", fwpacket.LocalIP.String())
			return true
		}
		if fwpacket.RemoteIP.To4()[0] != 192 {
			fw.outboundMetrics.dropped.Inc(1)
			//log.Printf("[firewall.outbound.drop] packet not destined to vpn ip - dst %v", fwpacket.RemoteIP.String())
			return true
		}
		fw.outboundMetrics.allowed.Inc(1)
	}

	return false
}

func (fw *Firewall) PrintMetrics() {
	fmt.Printf("firewall inbound packets total: %v\n", fw.inboundMetrics.packets.Count())
	fmt.Printf("firewall inbound packets allowed: %v\n", fw.inboundMetrics.allowed.Count())
	fmt.Printf("firewall inbound packets dropped: %v\n", fw.inboundMetrics.dropped.Count())

	fmt.Printf("firewall outbound packets total: %v\n", fw.outboundMetrics.packets.Count())
	fmt.Printf("firewall outbound packets allowed: %v\n", fw.outboundMetrics.allowed.Count())
	fmt.Printf("firewall outbound packets dropped: %v\n", fw.outboundMetrics.dropped.Count())
}
