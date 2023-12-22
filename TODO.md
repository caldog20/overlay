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

- Change peers to start when added
- Lock node keypair when using it for handshake initiation


- Implement peer timers for send/recv packets and timout/dead peer detection
- possibly allow optional keepalives that are tunable 
- after a certain dead period, flush all queues, reset peer state properly to await outbound/inbound packet 
and ensure state is fresh

- validate use of atomics/locks and refactor placement and usage 
- maybe implement lock for noise state separately instead of for entire peer
- figure out a way to stop peers properly if needed outside of runtime quit