package node

import (
	"log"
	"net"
	"time"
)

func (peer *Peer) contextDone() bool {
	select {
	case <-peer.ctx.Done():
		return true
	default:
		return false
	}
}

func (peer *Peer) Inbound() {
	// log.Print("starting inbound routine")

	var err error

	for buffer := range peer.inbound {
		// nil value is signal to exit the routine
		if buffer == nil {
			return
		}
		peer.pendingLock.RLock()
		peer.noise.rx.SetNonce(buffer.header.Counter)
		buffer.packet, err = peer.noise.rx.Decrypt(buffer.packet[:0], nil, buffer.in[HeaderLen:buffer.size])
		if err != nil {
			log.Println("decrypt failed")
			PutInboundBuffer(buffer)
			peer.pendingLock.RUnlock()
			continue
		}

		peer.pendingLock.RUnlock()

		peer.timers.receivedPacket.Reset(TimerRxTimeout)

		if len(buffer.packet) > 0 {
			// TODO: Check source IP here and ensure it matches peer's Ip
			peer.node.tun.Write(buffer.packet)
		}

		peer.UpdateEndpoint(buffer.raddr)

		PutInboundBuffer(buffer)
	}
}

func (peer *Peer) Outbound() {
	for buffer := range peer.outbound {
		// nil value is signal to exit the routine
		if buffer == nil {
			return
		}

		peer.pendingLock.RLock()
		out, err := buffer.header.Encode(buffer.out, Data, peer.node.id, peer.noise.tx.Nonce())
		out, err = peer.noise.tx.Encrypt(out, nil, buffer.packet[:buffer.size])
		if err != nil {
			log.Println("encrypt failed")
			PutOutboundBuffer(buffer)
			peer.pendingLock.RUnlock()
			continue
		}

		peer.timers.keepalive.Reset(TimerKeepalive)
		// peer.timers.sentPacket.Stop()
		// peer.timers.sentPacket.Reset(TimerKeepalive)

		peer.pendingLock.RUnlock()
		// To protect endpoint changes
		peer.mu.RLock()
		peer.node.conn.WriteToUDP(out, peer.raddr)
		peer.mu.RUnlock()
		// log.Printf("Sent data to %s - len: %d", p.remote.String(), elem.size)
		PutOutboundBuffer(buffer)
	}
}

func (peer *Peer) UpdateEndpoint(addr *net.UDPAddr) {
	peer.mu.RLock()
	var paddr *net.UDPAddr
	paddr = peer.raddr
	peer.mu.RUnlock()

	if !paddr.IP.Equal(addr.IP) || paddr.Port != addr.Port {
		peer.mu.Lock()
		*peer.raddr = *addr
		peer.mu.Unlock()
	}
}

func (peer *Peer) RequestPunch() {
	peer.mu.RLock()
	defer peer.mu.RUnlock()
	peer.node.RequestPunch(peer.ID)
}

func (peer *Peer) TrySendHandshake(retry bool) {
	peer.counters.handshakeRetries.Add(1)

	// TODO validate placement of lock here
	if retry {
		peer.RequestPunch()
		attempts := peer.counters.handshakeRetries.Load()
		if attempts > CountHandshakeRetries {
			// Peer never responded to handshakes, so flush all queues, and reset state
			log.Println("peer handshake retries exceeded, resetting peer state to idle")
			peer.timers.handshakeSent.Stop()
			peer.ResetState()
			return
		}
		// TODO Remove this in favor of polling updates from controller
		if attempts > 3 {
			peer.node.UpdateNodes()
		}
		log.Printf("retrying handshake attempt %d", peer.counters.handshakeRetries.Load())
	}

	peer.mu.Lock()
	defer peer.mu.Unlock()
	err := peer.InitHandshake(true)
	if err != nil {
		panic(err)
	}
	buffer := GetOutboundBuffer()
	peer.handshakeP1(buffer)
	peer.timers.handshakeSent.Stop()
	peer.timers.handshakeSent.Reset(time.Second * 3)
}

func (peer *Peer) Handshake() {
	// log.Print("starting handshake routine")
	// TODO handshake completion function
	for buffer := range peer.handshakes {
		// nil value is signal to exit the routine
		if buffer == nil {
			return
		}
		log.Printf("peer %d - received handshake message - remote: %s", peer.ID, peer.raddr.String())
		// received handshake inbound, process
		state := peer.noise.state.Load()
		switch state {
		case 0: // receiving first handshake message as responder
			if buffer.header.Counter != 0 {
				panic("header counter doesnt match state 0")
			}
			err := peer.handshakeP2(buffer)
			if err != nil {
				panic(err)
			}
			peer.noise.state.Store(2)
			peer.inTransport.Store(true)
			peer.pendingLock.Unlock()
			peer.counters.handshakeRetries.Store(0)
			peer.timers.handshakeSent.Stop()
			peer.timers.keepalive.Reset(TimerKeepalive + time.Second*5)
			// Handshake finished
		case 1: // receiving handshake response as initiator
			if buffer.header.Counter != 1 {
				panic("header counter doesnt match state 1")
			}
			err := peer.handshakeP2(buffer)
			if err != nil {
				panic(err)
			}
			peer.noise.state.Store(2)
			peer.inTransport.Store(true)
			peer.pendingLock.Unlock()
			peer.timers.handshakeSent.Stop()
			peer.counters.handshakeRetries.Store(0)
			peer.timers.keepalive.Reset(TimerKeepalive)
		// Handshake finished
		case 2: // Receiving new handshake from peer, lock and consume handshake initiation
			peer.pendingLock.Lock()
			// TODO Do something better here
			// Peer roaming possibly
			peer.UpdateEndpoint(buffer.raddr)
			err := peer.handshakeP2(buffer)
			if err != nil {
				panic(err)
			}
			peer.noise.state.Store(2)
			peer.inTransport.Store(true)
			peer.pendingLock.Unlock()
			peer.timers.handshakeSent.Stop()
			peer.counters.handshakeRetries.Store(0)
			peer.timers.keepalive.Reset(TimerKeepalive)
			peer.timers.receivedPacket.Reset(TimerRxTimeout)
		default:
			panic("out of sequence handshake message received")
		}
	}
}

//func (peer *Peer) SendPending() {
//	//peer.pendingLock.Lock()
//	//defer peer.pendingLock.Unlock()
//	//peer.mu.RLock()
//	//peer.mu.RUnlock()
//
//	for {
//		buffer, ok := <-peer.pending
//		if !ok {
//			return
//		}
//		peer.mu.RLock()
//		out, err := buffer.header.Encode(buffer.out, Data, peer.node.id, peer.noise.tx.Nonce())
//		out, err = peer.noise.tx.Encrypt(out, nil, buffer.packet[:buffer.size])
//		if err != nil {
//			// TODO, if encrypt fails then reset state and start over
//			// Maybe generalize outbound sending and use here?
//			log.Println("encrypt failed for pending packet")
//			peer.mu.RUnlock()
//			PutOutboundBuffer(buffer)
//			continue
//		}
//		peer.node.conn.WriteToUdp(out, peer.raddr)
//		peer.mu.RUnlock()
//		PutOutboundBuffer(buffer)
//	}
//}

func (peer *Peer) handshakeP1(buffer *OutboundBuffer) {
	// encode header
	final, _ := buffer.header.Encode(buffer.out, Handshake, peer.node.id, 0)

	final, _, _, err := peer.noise.hs.WriteMessage(final, nil)
	if err != nil {
		panic("error writing first handshake message")
	}
	peer.noise.state.Store(1)
	peer.node.conn.WriteToUDP(final, peer.raddr)
	PutOutboundBuffer(buffer)
	log.Printf("peer %d - sent handshake message - remote: %s", peer.ID, peer.raddr.String())
}

// TODO Refactor this
func (peer *Peer) handshakeP2(buffer *InboundBuffer) error {
	peer.mu.Lock()
	defer peer.mu.Unlock()

	var err error
	if peer.noise.initiator {
		_, peer.noise.tx, peer.noise.rx, err = peer.noise.hs.ReadMessage(nil, buffer.in[HeaderLen:buffer.size])
		if err != nil {
			return err
		}
		peer.raddr = buffer.raddr
		peer.noise.hs = nil
	} else {
		// peer.mu.Lock()
		// Initialze handshake for responder
		err = peer.InitHandshake(false)
		if err != nil {
			return err
		}

		// Read handshake init and response
		_, _, _, err = peer.noise.hs.ReadMessage(nil, buffer.in[HeaderLen:buffer.size])
		if err != nil {
			return err
		}

		outbuf := GetOutboundBuffer()
		final, _ := outbuf.header.Encode(outbuf.out, Handshake, peer.node.id, 1)
		final, peer.noise.rx, peer.noise.tx, err = peer.noise.hs.WriteMessage(final, nil)
		if err != nil {
			return err
		}

		peer.node.conn.WriteToUDP(final, peer.raddr)
		PutOutboundBuffer(outbuf)
	}

	PutInboundBuffer(buffer)
	return nil
}

func (peer *Peer) HandshakeTimeout() {
	if peer.noise.state.Load() > 0 {
		// Handshake response not received, send another handshake
		log.Printf("peer %d handshake response timeout", peer.ID)
		if peer.noise.initiator {
			peer.TrySendHandshake(true)
		}
	}
}

func (peer *Peer) TXTimeout() {
	if len(peer.outbound) == 0 {
		log.Printf("peer %d sending keepalive", peer.ID)
		// Queue up empty packet
		buffer := GetOutboundBuffer()
		buffer.peer = peer
		peer.outbound <- buffer
	}
}

func (peer *Peer) RXTimeout() {
	log.Println("RX TIMEOUT")
	if !peer.inTransport.Load() {
		return
	}

	peer.timers.keepalive.Stop()
	peer.timers.receivedPacket.Stop()

	peer.pendingLock.Lock()
	peer.noise.state.Store(0)

	// TODO Fix this
	peer.mu.RLock()
	initiator := peer.noise.initiator
	peer.mu.RUnlock()

	if !initiator {
		log.Println("RX Timeout but not initiator, resetting peer state")
		peer.timers.receivedPacket.Stop()
		peer.timers.handshakeSent.Stop()
		peer.ResetState()
		return
	}

	peer.TrySendHandshake(true)
}
