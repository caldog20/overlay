package main

import (
	"crypto/rand"
	"gitlab.com/yawning/nyquist.git"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/yawning/nyquist.git/cipher"
	"gitlab.com/yawning/nyquist.git/dh"
	"gitlab.com/yawning/nyquist.git/hash"
	"gitlab.com/yawning/nyquist.git/pattern"
)

func TestExample(t *testing.T) {
	require := require.New(t)

	// Protocols can be constructed by parsing a protocol name.
	//protocol, err := nyquist.NewProtocol("Noise_XX_25519_ChaChaPoly_BLAKE2s")
	//require.NoError(err, "NewProtocol")

	// Protocols can also be constructed manually.
	protocol := &nyquist.Protocol{
		Pattern: pattern.KK,
		DH:      dh.X25519,
		Cipher:  cipher.ChaChaPoly,
		Hash:    hash.BLAKE2s,
	}
	//require.Equal(protocol, protocol2)

	// Each side needs a HandshakeConfig, properly filled out.
	aliceStatic, err := protocol.DH.GenerateKeypair(rand.Reader)
	require.NoError(err, "Generate Alice's static keypair")

	bobStatic, err := protocol.DH.GenerateKeypair(rand.Reader)
	require.NoError(err, "Generate Bob's static keypair")

	aliceCfg := &nyquist.HandshakeConfig{
		Protocol:     protocol,
		LocalStatic:  aliceStatic,
		RemoteStatic: bobStatic.Public(),
		IsInitiator:  true,
	}

	bobCfg := &nyquist.HandshakeConfig{
		Protocol:     protocol,
		LocalStatic:  bobStatic,
		RemoteStatic: aliceStatic.Public(),
		IsInitiator:  false,
	}

	// Each side then constructs a HandshakeState.
	aliceHs, err := nyquist.NewHandshake(aliceCfg)
	require.NoError(err, "NewHandshake(aliceCfg)")

	bobHs, err := nyquist.NewHandshake(bobCfg)
	require.NoError(err, "NewHandshake(bobCfg")

	// Ensuring that HandshakeState.Reset() is called, will make sure that
	// the HandshakeState isn't inadvertently reused.
	defer aliceHs.Reset()
	defer bobHs.Reset()

	// The SymmetricState and CipherState objects embedded in the
	// HandshakeState can be accessed while the handshake is in progress,
	// though most users likely will not need to do this.
	aliceSs := aliceHs.SymmetricState()
	require.NotNil(aliceSs, "aliceHs.SymmetricState()")
	aliceCs := aliceSs.CipherState()
	require.NotNil(aliceCs, "aliceSS.CipherState()")

	// Then, each side calls hs.ReadMessage/hs.WriteMessage as appropriate.

	//alicePlaintextE := []byte("alice e plaintext") // Handshake message payloads are optional.

	// Alice -> s
	aliceMsg1, err := aliceHs.WriteMessage(nil, nil)
	require.NoError(err, "alice -> bob e, es, ss") // (alice) -> s (bob)

	// Bob Reads s
	bobRecv, err := bobHs.ReadMessage(nil, aliceMsg1)
	require.NoError(err, "bob read alices e, es, ss")

	bobMsg1, err := bobHs.WriteMessage(nil, nil) // (bob) -> s (alice)
	//require.NoError(err, "bobHS.WriteMessage(bob1)")
	require.Equal(nyquist.ErrDone, err, "bob -> alice e, es, ss - bobs handshake done")

	// Alice Reads s
	aliceRecv, err := aliceHs.ReadMessage(nil, bobMsg1)
	require.Equal(nyquist.ErrDone, err, "alice read bobs e, es ,ss - alice handshake done")

	// Once a handshake is completed, the CipherState objects, handshake hash
	// and various public keys can be pulled out of the HandshakeStatus object.
	aliceStatus := aliceHs.GetStatus()
	bobStatus := bobHs.GetStatus()

	require.Equal(aliceStatus.HandshakeHash, bobStatus.HandshakeHash, "Handshake hashes match")
	require.Equal(aliceStatus.LocalEphemeral.Bytes(), bobStatus.RemoteEphemeral.Bytes())
	require.Equal(bobStatus.LocalEphemeral.Bytes(), aliceStatus.RemoteEphemeral.Bytes())
	require.Equal(aliceStatus.RemoteStatic.Bytes(), bobStatic.Public().Bytes())
	require.Equal(bobStatus.RemoteStatic.Bytes(), aliceStatic.Public().Bytes())

	// Then the CipherState objects can be used to exchange messages.
	aliceTx, aliceRx := aliceStatus.CipherStates[0], aliceStatus.CipherStates[1]
	bobRx, bobTx := bobStatus.CipherStates[0], bobStatus.CipherStates[1] // Reversed from alice!

	// Naturally CipherState.Reset() also exists.
	defer func() {
		aliceTx.Reset()
		aliceRx.Reset()
	}()
	defer func() {
		bobTx.Reset()
		bobRx.Reset()
	}()

	// Alice -> Bob, post-handshake.
	alicePlaintext := []byte("alice transport plaintext")
	aliceMsg3, err := aliceTx.EncryptWithAd(nil, nil, alicePlaintext)
	require.NoError(err, "aliceTx.EncryptWithAd()")

	bobRecv, err = bobRx.DecryptWithAd(nil, nil, aliceMsg3)
	require.NoError(err, "bobRx.DecryptWithAd()")
	require.Equal(alicePlaintext, bobRecv)

	// Bob -> Alice, post-handshake.
	bobPlaintext := []byte("bob transport plaintext")
	bobMsg2, err := bobTx.EncryptWithAd(nil, nil, bobPlaintext)
	require.NoError(err, "bobTx.EncryptWithAd()")

	aliceRecv, err = aliceRx.DecryptWithAd(nil, nil, bobMsg2)
	require.NoError(err, "aliceRx.DecryptWithAd")
	require.Equal(bobPlaintext, aliceRecv)
}
