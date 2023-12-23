package node

import (
	"log"
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
	//log.Print("starting inbound routine")

	var err error

	for buffer := range peer.inbound {
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
		peer.node.tun.Write(buffer.packet)
		// TODO Fix remote address roaming updates
		//if !peer.raddr.IP.Equal(buffer.raddr.IP) {
		//	peer.mu.Lock()
		//	peer.raddr = buffer.raddr
		//	peer.mu.Unlock()
		//}
		PutInboundBuffer(buffer)
	}
}

func (peer *Peer) Outbound() {
	for buffer := range peer.outbound {
		peer.pendingLock.RLock()
		out, err := buffer.header.Encode(buffer.out, Data, peer.node.id, peer.noise.tx.Nonce())
		out, err = peer.noise.tx.Encrypt(out, nil, buffer.packet[:buffer.size])
		if err != nil {
			log.Println("encrypt failed")
			PutOutboundBuffer(buffer)
			peer.pendingLock.RUnlock()
			continue
		}
		peer.pendingLock.RUnlock()
		peer.node.conn.WriteToUdp(out, peer.raddr)
		//log.Printf("Sent data to %s - len: %d", p.remote.String(), elem.size)
		PutOutboundBuffer(buffer)
	}
}

func (peer *Peer) TrySendHandshake(retry bool) {
	peer.counters.handshakeRetries.Add(1)

	// TODO validate placement of lock here
	if retry {
		attempts := peer.counters.handshakeRetries.Load()
		if attempts > CountHandshakeRetries {
			// Peer never responded to handshakes, so flush all queues, and reset state
			log.Println("peer handshake retries exceeded, resetting peer state to idle")
			peer.timers.handshakeSent.Stop()
			peer.ResetState()
			return
		}
		log.Printf("retrying handshake attempt %d", peer.counters.handshakeRetries.Load())
	}

	peer.mu.Lock()
	defer peer.mu.Unlock()
	peer.InitHandshake(true)
	buffer := GetOutboundBuffer()
	peer.handshakeP1(buffer)
	peer.timers.handshakeSent.Stop()
	peer.timers.handshakeSent.Reset(time.Second * 3)
}

func (peer *Peer) Handshake() {
	//log.Print("starting handshake routine")
	// TODO handshake completion function
	for {
		select {
		case hs := <-peer.handshakes:
			log.Printf("peer %d received handshake message", peer.Id)
			// received handshake inbound, process
			state := peer.noise.state.Load()
			switch state {
			case 0: // receiving first handshake message as responder
				if hs.header.Counter != 0 {
					panic("header counter doesnt match state 0")
				}
				err := peer.handshakeP2(hs)
				if err != nil {
					panic(err)
				}
				peer.noise.state.Store(2)
				peer.inTransport.Store(true)
				peer.pendingLock.Unlock()
				peer.counters.handshakeRetries.Store(0)
				peer.timers.handshakeSent.Stop()
				// Handshake finished
			case 1: // receiving handshake response as initiator
				if hs.header.Counter != 1 {
					panic("header counter doesnt match state 1")
				}
				err := peer.handshakeP2(hs)
				if err != nil {
					panic(err)
				}
				peer.noise.state.Store(2)
				peer.inTransport.Store(true)
				peer.pendingLock.Unlock()
				peer.timers.handshakeSent.Stop()
				peer.counters.handshakeRetries.Store(0)
			// Handshake finished
			case 2: // Receiving new handshake from peer, lock and consume handshake initiation
				peer.pendingLock.Lock()
				err := peer.handshakeP2(hs)
				if err != nil {
					panic(err)
				}
				peer.noise.state.Store(2)
				peer.inTransport.Store(true)
				peer.pendingLock.Unlock()
				peer.timers.handshakeSent.Stop()
				peer.counters.handshakeRetries.Store(0)
			default:
				panic("out of sequence handshake message received")
			}
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
	peer.node.conn.WriteToUdp(final, peer.raddr)
	PutOutboundBuffer(buffer)
	log.Printf("sending p1 to peer %s", peer.raddr.String())
}

// TODO Refactor this
func (peer *Peer) handshakeP2(buffer *InboundBuffer) error {
	peer.mu.Lock()
	defer peer.mu.Unlock()

	var err error
	log.Printf("received handshake message from peer %s", peer.raddr.String())
	if peer.noise.initiator {
		_, peer.noise.tx, peer.noise.rx, err = peer.noise.hs.ReadMessage(nil, buffer.in[HeaderLen:buffer.size])
		if err != nil {
			return err
		}
		peer.raddr = buffer.raddr
		peer.noise.hs = nil
	} else {
		//peer.mu.Lock()
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

		peer.node.conn.WriteToUdp(final, peer.raddr)
		PutOutboundBuffer(outbuf)
	}

	PutInboundBuffer(buffer)
	return nil
}

func (peer *Peer) HandshakeTimeout() {
	if peer.noise.state.Load() > 0 {
		// Handshake response not received, send another handshake
		log.Printf("peer %d handshake response timeout", peer.Id)
		peer.TrySendHandshake(true)
	}
}

func (peer *Peer) RXTimeout() {
	if !peer.inTransport.Load() {
		return
	}

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
