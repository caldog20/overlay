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
	defer peer.wg.Done()

	var err error
	for {
		select {
		case <-peer.ctx.Done():
			log.Println("DONE")
			return
		case buffer := <-peer.inbound:
			peer.mu.RLock()
			peer.noise.rx.SetNonce(buffer.header.Counter)
			buffer.packet, err = peer.noise.rx.Decrypt(buffer.packet[:0], nil, buffer.in[HeaderLen:buffer.size])
			if err != nil {
				log.Println("decrypt failed")
				PutInboundBuffer(buffer)
				peer.mu.RUnlock()
				continue
			}
			peer.mu.RUnlock()
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
}

func (peer *Peer) Outbound() {
	defer peer.wg.Done()
	peer.SendPending() // Block here until nothing else on channel to send pending
	for {
		select {
		case <-peer.ctx.Done():
			log.Println("DONE")
			return
		case buffer := <-peer.outbound:
			peer.mu.RLock()
			out, err := buffer.header.Encode(buffer.out, Data, peer.node.id, peer.noise.tx.Nonce())
			out, err = peer.noise.tx.Encrypt(out, nil, buffer.packet[:buffer.size])
			if err != nil {
				log.Println("encrypt failed")
				PutOutboundBuffer(buffer)
				peer.mu.RUnlock()
				continue
			}
			peer.node.conn.WriteToUdp(out, peer.raddr)
			//log.Printf("Sent data to %s - len: %d", p.remote.String(), elem.size)
			PutOutboundBuffer(buffer)
			peer.mu.RUnlock()
		}
	}
}

func (peer *Peer) Handshake(initiate bool) {
	defer peer.wg.Done()

	if initiate {
		peer.mu.Lock()
		peer.InitHandshake(true)
		buffer := GetOutboundBuffer()
		peer.handshakeP1(buffer)
		peer.timers.handshakeSent.Stop()
		peer.timers.handshakeSent.Reset(time.Second * 3)
		//peer.mu.Unlock()
	}

	for {
		select {
		case <-peer.ctx.Done():
			log.Println("DONE")
			return
		case <-peer.timers.handshakeSent.C:
			if peer.noise.state.Load() == 1 {
				peer.cancel()
				peer.mu.Unlock()
				return
			}
		case hs := <-peer.handshakes:
			// received handshake inbound, process
			state := peer.noise.state.Load()
			switch state {
			// receiving first handshake message as responder
			case 0:
				// receiving handshake response as initiator
				if hs.header.Counter != 0 {
					panic("header counter doesnt match state 0")
				}
				peer.handshakeP2(hs)
			case 1:
				if hs.header.Counter != 1 {
					panic("header counter doesnt match state 1")
				}
				peer.handshakeP2(hs)
			default:
				panic("out of sequence handshake message received")
			}
		}

	}
}

func (peer *Peer) SendPending() {
	for {
		select {
		case <-peer.ctx.Done():
			log.Println("DONE")
			return
		case buffer, ok := <-peer.pending:
			if !ok {
				return // channel closed??
			}
			if !peer.inTransport.Load() {
				return
			}
			peer.mu.RLock()
			out, err := buffer.header.Encode(buffer.out, Data, peer.node.id, peer.noise.tx.Nonce())
			out, err = peer.noise.tx.Encrypt(out, nil, buffer.packet[:buffer.size])
			if err != nil {
				// TODO, if encrypt fails then reset state and start over
				// Maybe generalize outbound sending and use here?
				log.Println("encrypt failed for pending packet")
				peer.mu.RUnlock()
				PutOutboundBuffer(buffer)
				continue
			}
			peer.node.conn.WriteToUdp(out, peer.raddr)
			peer.mu.RUnlock()
			PutOutboundBuffer(buffer)
		default:
			return
		}
	}

}

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
func (peer *Peer) handshakeP2(buffer *InboundBuffer) {
	var err error
	log.Printf("received handshake message from peer %s", peer.raddr.String())
	if peer.noise.initiator {
		_, peer.noise.tx, peer.noise.rx, err = peer.noise.hs.ReadMessage(nil, buffer.in[HeaderLen:buffer.size])
		if err != nil {
			panic("error reading handshake message")
		}
		peer.raddr = buffer.raddr
		peer.noise.hs = nil
		peer.noise.state.Store(2)
		peer.mu.Unlock()
		PutInboundBuffer(buffer)
		peer.inTransport.Store(true)
		return
	}

	peer.mu.Lock()

	// Initialze handshake for responder
	err = peer.InitHandshake(false)
	if err != nil {
		panic(err)
	}
	// Read handshake init and response
	_, _, _, err = peer.noise.hs.ReadMessage(nil, buffer.in[HeaderLen:buffer.size])
	if err != nil {
		panic("error reading handshake message")
	}

	outbuf := GetOutboundBuffer()
	final, _ := outbuf.header.Encode(outbuf.out, Handshake, peer.node.id, 1)
	final, peer.noise.rx, peer.noise.tx, err = peer.noise.hs.WriteMessage(final, nil)
	if err != nil {
		panic("error writing handshake response")
	}

	peer.node.conn.WriteToUdp(final, peer.raddr)
	peer.noise.state.Store(2)
	peer.noise.hs = nil
	peer.mu.Unlock()
	PutOutboundBuffer(outbuf)
	peer.inTransport.Store(true)
}
