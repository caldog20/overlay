package node

import "github.com/flynn/noise"

var TempCS = noise.NewCipherSuite(noise.DH25519, noise.CipherChaChaPoly, noise.HashBLAKE2s)

func NewInitiatorHS(keyPair noise.DHKey, peerStatic []byte) (*noise.HandshakeState, error) {
	hs, err := noise.NewHandshakeState(noise.Config{
		CipherSuite:   TempCS,
		Pattern:       noise.HandshakeIK,
		Initiator:     true,
		StaticKeypair: keyPair,
		PeerStatic:    peerStatic,
	})

	if err != nil {
		return nil, err
	}

	return hs, nil
}

func NewResponderHS(keyPair noise.DHKey) (*noise.HandshakeState, error) {
	hs, err := noise.NewHandshakeState(noise.Config{
		CipherSuite:   TempCS,
		Pattern:       noise.HandshakeIK,
		Initiator:     false,
		StaticKeypair: keyPair,
	})

	if err != nil {
		return nil, err
	}

	return hs, nil
}
