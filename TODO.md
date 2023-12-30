## TODO

When a new peer is added, it's routines should be started and waiting
The routines should be able to block/unblock as needed if no packets are being sent

The tunnel/udp receive should pass the buffer to the peer if the peer is found and running/runnable
the peer should determine whether a new handshake is needed on consumption of buffer from tunnel/udp

Things need refactored within the peer to make it possible to test the peer logic decoupled from the node

This may mean custom types are required to wrap tunnel/udp types to make test types to pass to the peer

This may also mean a manager type is needed to wrap tunnel/udp connections from the host in order to mock it for testing
and pass it to the peer
or make a mock node possible

I'm rambling now.. ^.^

- Change peers to start when added - done
- Lock node keypair when using it for handshake initiation - done


- Implement peer timers for send/recv packets and timout/dead peer detection - done
- possibly allow optional keepalives that are tunable
- after a certain dead period, flush all queues, reset peer state properly to await outbound/inbound packet 
and ensure state is fresh - done

- validate use of atomics/locks and refactor placement and usage 
- maybe implement lock for noise state separately instead of for entire peer
- figure out a way to stop peers properly if needed outside of runtime quit
- eventually figure out how to manage noise cipherstate pairs and rekeying
- track nonces for out of order packets?

TODO: Fix restarting keepalive timer when responder is completing handshake. 
Wait until first successful encrypted packet is received to ensure handshake is proper before restarting TX/keepalive timer


- Punching when not receiving handshake message, maybe another timer triggered after first attempt shorter than handshake timeout
- For now, request punch on each handshake attempt whether I like it or not
- Eventually support ipv6 inside and outside the tunnel

- Protect peer endpoint better with synchronization


- Define rules for peers and peer updates
  - What updates should be received from controller after initial state pull?
    - How do these updates affect active peers?
  - How can we track connected/disconnected state and should peers track this state for remote peers
  - Add an ability similar to ICE for peers to negotiate ip:port pairs to connect to each other with
    - This should solve peers on the same local network as well
  - Initial handshake on outbound traffic
  - After so many keepalives without any real data, close the peer

- Peer Timers and handshake initiation rules: (inspired by WireGuard protocol)
  - A handshake is only initiated when outbound traffic is queued for a remote peer
  - if handshake needs to retry after session was alive, retry x times then reset peer state and idle, unless more outbound data is queued
  - If no data has been sent in x time, and we haven't received a keepalive, send keepalive
  - if no data has been receieved after x time, and we haven't received a keepalive, send keepalive