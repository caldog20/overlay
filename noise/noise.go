package noiseimpl

import "github.com/flynn/noise"

var TempCS = noise.NewCipherSuite(noise.DH25519, noise.CipherAESGCM, noise.HashSHA256)

func NewHandshake(initiator bool, keyPair noise.DHKey, peerStatic []byte) (*noise.HandshakeState, error) {
	var hs *noise.HandshakeState
	var err error
	if initiator {
		hs, err = newInitiatorHS(keyPair, peerStatic)
	} else {
		hs, err = newResponderHS(keyPair)
	}

	if err != nil {
		return nil, err
	}

	return hs, nil
}

func newInitiatorHS(keyPair noise.DHKey, peerStatic []byte) (*noise.HandshakeState, error) {
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

func newResponderHS(keyPair noise.DHKey) (*noise.HandshakeState, error) {
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

//package noise
//
//import (
//	"crypto/rand"
//	"crypto/subtle"
//	"encoding/binary"
//	"errors"
//	"golang.org/x/crypto/blake2s"
//	"golang.org/x/crypto/chacha20poly1305"
//	"golang.org/x/crypto/curve25519"
//	"golang.org/x/crypto/hkdf"
//	"hash"
//	"io"
//	"math"
//)
//
///* ---------------------------------------------------------------- *
// * TYPES                                                            *
// * ---------------------------------------------------------------- */
//
//type Keypair struct {
//	PublicKey  [32]byte
//	PrivateKey [32]byte
//}
//
//type MessageBuffer struct {
//	NE         [32]byte
//	NS         []byte
//	Ciphertext []byte
//}
//
//type cipherstate struct {
//	k [32]byte
//	n uint64
//}
//
//type symmetricstate struct {
//	cs cipherstate
//	ck [32]byte
//	h  [32]byte
//}
//
//type handshakestate struct {
//	ss  symmetricstate
//	s   Keypair
//	e   Keypair
//	rs  [32]byte
//	re  [32]byte
//	psk [32]byte
//}
//
//type NoiseSession struct {
//	hs  handshakestate
//	h   [32]byte
//	cs1 cipherstate
//	cs2 cipherstate
//	mc  uint64
//	i   bool
//}
//
///* ---------------------------------------------------------------- *
// * CONSTANTS                                                        *
// * ---------------------------------------------------------------- */
//
//var emptyKey = [32]byte{
//	0x00, 0x00, 0x00, 0x00,
//	0x00, 0x00, 0x00, 0x00,
//	0x00, 0x00, 0x00, 0x00,
//	0x00, 0x00, 0x00, 0x00,
//	0x00, 0x00, 0x00, 0x00,
//	0x00, 0x00, 0x00, 0x00,
//	0x00, 0x00, 0x00, 0x00,
//	0x00, 0x00, 0x00, 0x00,
//}
//
//var minNonce = uint64(0)
//
///* ---------------------------------------------------------------- *
// * UTILITY FUNCTIONS                                                *
// * ---------------------------------------------------------------- */
//
//func GetPublicKey(kp *Keypair) [32]byte {
//	return kp.PublicKey
//}
//
//func isEmptyKey(k [32]byte) bool {
//	return subtle.ConstantTimeCompare(k[:], emptyKey[:]) == 1
//}
//
//func validatePublicKey(k []byte) bool {
//	forbiddenCurveValues := [12][]byte{
//		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
//		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
//		{224, 235, 122, 124, 59, 65, 184, 174, 22, 86, 227, 250, 241, 159, 196, 106, 218, 9, 141, 235, 156, 50, 177, 253, 134, 98, 5, 22, 95, 73, 184, 0},
//		{95, 156, 149, 188, 163, 80, 140, 36, 177, 208, 177, 85, 156, 131, 239, 91, 4, 68, 92, 196, 88, 28, 142, 134, 216, 34, 78, 221, 208, 159, 17, 87},
//		{236, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 127},
//		{237, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 127},
//		{238, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 127},
//		{205, 235, 122, 124, 59, 65, 184, 174, 22, 86, 227, 250, 241, 159, 196, 106, 218, 9, 141, 235, 156, 50, 177, 253, 134, 98, 5, 22, 95, 73, 184, 128},
//		{76, 156, 149, 188, 163, 80, 140, 36, 177, 208, 177, 85, 156, 131, 239, 91, 4, 68, 92, 196, 88, 28, 142, 134, 216, 34, 78, 221, 208, 159, 17, 215},
//		{217, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
//		{218, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
//		{219, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 25},
//	}
//
//	for _, testValue := range forbiddenCurveValues {
//		if subtle.ConstantTimeCompare(k[:], testValue[:]) == 1 {
//			panic("Invalid public key")
//		}
//	}
//	return true
//}
//
///* ---------------------------------------------------------------- *
// * PRIMITIVES                                                       *
// * ---------------------------------------------------------------- */
//
//func incrementNonce(n uint64) uint64 {
//	return n + 1
//}
//
//func dh(private_key [32]byte, public_key [32]byte) [32]byte {
//	var ss [32]byte
//	curve25519.ScalarMult(&ss, &private_key, &public_key)
//	return ss
//}
//
//func GenerateKeypair() Keypair {
//	var public_key [32]byte
//	var private_key [32]byte
//	_, _ = rand.Read(private_key[:])
//	curve25519.ScalarBaseMult(&public_key, &private_key)
//	if validatePublicKey(public_key[:]) {
//		return Keypair{public_key, private_key}
//	}
//	return GenerateKeypair()
//}
//
//func generatePublicKey(private_key [32]byte) [32]byte {
//	var public_key [32]byte
//	curve25519.ScalarBaseMult(&public_key, &private_key)
//	return public_key
//}
//
//func encrypt(k [32]byte, n uint64, ad []byte, plaintext []byte) []byte {
//	var nonce [12]byte
//	var ciphertext []byte
//	enc, _ := chacha20poly1305.New(k[:])
//	binary.LittleEndian.PutUint64(nonce[4:], n)
//	ciphertext = enc.Seal(nil, nonce[:], plaintext, ad)
//	return ciphertext
//}
//
//func decrypt(k [32]byte, n uint64, ad []byte, ciphertext []byte) (bool, []byte, []byte) {
//	var nonce [12]byte
//	var plaintext []byte
//	enc, err := chacha20poly1305.New(k[:])
//	binary.LittleEndian.PutUint64(nonce[4:], n)
//	plaintext, err = enc.Open(nil, nonce[:], ciphertext, ad)
//	return (err == nil), ad, plaintext
//}
//
//func getHash(a []byte, b []byte) [32]byte {
//	return blake2s.Sum256(append(a, b...))
//}
//
//func hashProtocolName(protocolName []byte) [32]byte {
//	var h [32]byte
//	if len(protocolName) <= 32 {
//		copy(h[:], protocolName)
//	} else {
//		h = getHash(protocolName, []byte{})
//	}
//	return h
//}
//
//func blake2HkdfInterface() hash.Hash {
//	h, _ := blake2s.New256([]byte{})
//	return h
//}
//
//func getHkdf(ck [32]byte, ikm []byte) ([32]byte, [32]byte, [32]byte) {
//	var k1 [32]byte
//	var k2 [32]byte
//	var k3 [32]byte
//	output := hkdf.New(blake2HkdfInterface, ikm[:], ck[:], []byte{})
//	io.ReadFull(output, k1[:])
//	io.ReadFull(output, k2[:])
//	io.ReadFull(output, k3[:])
//	return k1, k2, k3
//}
//
///* ---------------------------------------------------------------- *
// * STATE MANAGEMENT                                                 *
// * ---------------------------------------------------------------- */
//
///* CipherState */
//func initializeKey(k [32]byte) cipherstate {
//	return cipherstate{k, minNonce}
//}
//
//func hasKey(cs *cipherstate) bool {
//	return !isEmptyKey(cs.k)
//}
//
//func setNonce(cs *cipherstate, newNonce uint64) *cipherstate {
//	cs.n = newNonce
//	return cs
//}
//
//func encryptWithAd(cs *cipherstate, ad []byte, plaintext []byte) (*cipherstate, []byte, error) {
//	var err error
//	if cs.n == math.MaxUint64-1 {
//		err = errors.New("encryptWithAd: maximum nonce size reached")
//		return cs, []byte{}, err
//	}
//	e := encrypt(cs.k, cs.n, ad, plaintext)
//	cs = setNonce(cs, incrementNonce(cs.n))
//	return cs, e, err
//}
//
//func decryptWithAd(cs *cipherstate, ad []byte, ciphertext []byte) (*cipherstate, []byte, bool, error) {
//	var err error
//	if cs.n == math.MaxUint64-1 {
//		err = errors.New("decryptWithAd: maximum nonce size reached")
//		return cs, []byte{}, false, err
//	}
//	valid, ad, plaintext := decrypt(cs.k, cs.n, ad, ciphertext)
//	if valid {
//		cs = setNonce(cs, incrementNonce(cs.n))
//	}
//	return cs, plaintext, valid, err
//}
//
//func reKey(cs *cipherstate) *cipherstate {
//	e := encrypt(cs.k, math.MaxUint64, []byte{}, emptyKey[:])
//	copy(cs.k[:], e)
//	return cs
//}
//
///* SymmetricState */
//
//func initializeSymmetric(protocolName []byte) symmetricstate {
//	h := hashProtocolName(protocolName)
//	ck := h
//	cs := initializeKey(emptyKey)
//	return symmetricstate{cs, ck, h}
//}
//
//func mixKey(ss *symmetricstate, ikm [32]byte) *symmetricstate {
//	ck, tempK, _ := getHkdf(ss.ck, ikm[:])
//	ss.cs = initializeKey(tempK)
//	ss.ck = ck
//	return ss
//}
//
//func mixHash(ss *symmetricstate, data []byte) *symmetricstate {
//	ss.h = getHash(ss.h[:], data)
//	return ss
//}
//
//func mixKeyAndHash(ss *symmetricstate, ikm [32]byte) *symmetricstate {
//	var tempH [32]byte
//	var tempK [32]byte
//	ss.ck, tempH, tempK = getHkdf(ss.ck, ikm[:])
//	ss = mixHash(ss, tempH[:])
//	ss.cs = initializeKey(tempK)
//	return ss
//}
//
//func getHandshakeHash(ss *symmetricstate) [32]byte {
//	return ss.h
//}
//
//func encryptAndHash(ss *symmetricstate, plaintext []byte) (*symmetricstate, []byte, error) {
//	var ciphertext []byte
//	var err error
//	if hasKey(&ss.cs) {
//		_, ciphertext, err = encryptWithAd(&ss.cs, ss.h[:], plaintext)
//		if err != nil {
//			return ss, []byte{}, err
//		}
//	} else {
//		ciphertext = plaintext
//	}
//	ss = mixHash(ss, ciphertext)
//	return ss, ciphertext, err
//}
//
//func decryptAndHash(ss *symmetricstate, ciphertext []byte) (*symmetricstate, []byte, bool, error) {
//	var plaintext []byte
//	var valid bool
//	var err error
//	if hasKey(&ss.cs) {
//		_, plaintext, valid, err = decryptWithAd(&ss.cs, ss.h[:], ciphertext)
//		if err != nil {
//			return ss, []byte{}, false, err
//		}
//	} else {
//		plaintext, valid = ciphertext, true
//	}
//	ss = mixHash(ss, ciphertext)
//	return ss, plaintext, valid, err
//}
//
//func split(ss *symmetricstate) (cipherstate, cipherstate) {
//	tempK1, tempK2, _ := getHkdf(ss.ck, []byte{})
//	cs1 := initializeKey(tempK1)
//	cs2 := initializeKey(tempK2)
//	return cs1, cs2
//}
//
///* HandshakeState */
//
//func initializeInitiator(prologue []byte, s Keypair, rs [32]byte, psk [32]byte) handshakestate {
//	var ss symmetricstate
//	var e Keypair
//	var re [32]byte
//	name := []byte("Noise_IK_25519_ChaChaPoly_BLAKE2s")
//	ss = initializeSymmetric(name)
//	mixHash(&ss, prologue)
//	mixHash(&ss, rs[:])
//	return handshakestate{ss, s, e, rs, re, psk}
//}
//
//func initializeResponder(prologue []byte, s Keypair, rs [32]byte, psk [32]byte) handshakestate {
//	var ss symmetricstate
//	var e Keypair
//	var re [32]byte
//	name := []byte("Noise_IK_25519_ChaChaPoly_BLAKE2s")
//	ss = initializeSymmetric(name)
//	mixHash(&ss, prologue)
//	mixHash(&ss, s.PublicKey[:])
//	return handshakestate{ss, s, e, rs, re, psk}
//}
//
//func writeMessageA(hs *handshakestate, payload []byte) (*handshakestate, MessageBuffer, error) {
//	var err error
//	var messageBuffer MessageBuffer
//	ne, ns, ciphertext := emptyKey, []byte{}, []byte{}
//	hs.e = GenerateKeypair()
//	ne = hs.e.PublicKey
//	mixHash(&hs.ss, ne[:])
//	/* No PSK, so skipping mixKey */
//	mixKey(&hs.ss, dh(hs.e.PrivateKey, hs.rs))
//	spk := make([]byte, len(hs.s.PublicKey))
//	copy(spk[:], hs.s.PublicKey[:])
//	_, ns, err = encryptAndHash(&hs.ss, spk)
//	if err != nil {
//		return hs, messageBuffer, err
//	}
//	mixKey(&hs.ss, dh(hs.s.PrivateKey, hs.rs))
//	_, ciphertext, err = encryptAndHash(&hs.ss, payload)
//	if err != nil {
//		return hs, messageBuffer, err
//	}
//	messageBuffer = MessageBuffer{ne, ns, ciphertext}
//	return hs, messageBuffer, err
//}
//
//func writeMessageB(hs *handshakestate, payload []byte) ([32]byte, MessageBuffer, cipherstate, cipherstate, error) {
//	var err error
//	var messageBuffer MessageBuffer
//	ne, ns, ciphertext := emptyKey, []byte{}, []byte{}
//	hs.e = GenerateKeypair()
//	ne = hs.e.PublicKey
//	mixHash(&hs.ss, ne[:])
//	/* No PSK, so skipping mixKey */
//	mixKey(&hs.ss, dh(hs.e.PrivateKey, hs.re))
//	mixKey(&hs.ss, dh(hs.e.PrivateKey, hs.rs))
//	_, ciphertext, err = encryptAndHash(&hs.ss, payload)
//	if err != nil {
//		cs1, cs2 := split(&hs.ss)
//		return hs.ss.h, messageBuffer, cs1, cs2, err
//	}
//	messageBuffer = MessageBuffer{ne, ns, ciphertext}
//	cs1, cs2 := split(&hs.ss)
//	return hs.ss.h, messageBuffer, cs1, cs2, err
//}
//
//func writeMessageRegular(cs *cipherstate, payload []byte) (*cipherstate, MessageBuffer, error) {
//	var err error
//	var messageBuffer MessageBuffer
//	ne, ns, ciphertext := emptyKey, []byte{}, []byte{}
//	cs, ciphertext, err = encryptWithAd(cs, []byte{}, payload)
//	if err != nil {
//		return cs, messageBuffer, err
//	}
//	messageBuffer = MessageBuffer{ne, ns, ciphertext}
//	return cs, messageBuffer, err
//}
//
//func readMessageA(hs *handshakestate, message *MessageBuffer) (*handshakestate, []byte, bool, error) {
//	var err error
//	var plaintext []byte
//	var valid2 bool = false
//	var valid1 bool = true
//	if validatePublicKey(message.NE[:]) {
//		hs.re = message.NE
//	}
//	mixHash(&hs.ss, hs.re[:])
//	/* No PSK, so skipping mixKey */
//	mixKey(&hs.ss, dh(hs.s.PrivateKey, hs.re))
//	_, ns, valid1, err := decryptAndHash(&hs.ss, message.NS)
//	if err != nil {
//		return hs, plaintext, (valid1 && valid2), err
//	}
//	if valid1 && len(ns) == 32 && validatePublicKey(message.NS[:]) {
//		copy(hs.rs[:], ns)
//	}
//	mixKey(&hs.ss, dh(hs.s.PrivateKey, hs.rs))
//	_, plaintext, valid2, err = decryptAndHash(&hs.ss, message.Ciphertext)
//	return hs, plaintext, (valid1 && valid2), err
//}
//
//func readMessageB(hs *handshakestate, message *MessageBuffer) ([32]byte, []byte, bool, cipherstate, cipherstate, error) {
//	var err error
//	var plaintext []byte
//	var valid2 bool = false
//	var valid1 bool = true
//	if validatePublicKey(message.NE[:]) {
//		hs.re = message.NE
//	}
//	mixHash(&hs.ss, hs.re[:])
//	/* No PSK, so skipping mixKey */
//	mixKey(&hs.ss, dh(hs.e.PrivateKey, hs.re))
//	mixKey(&hs.ss, dh(hs.s.PrivateKey, hs.re))
//	_, plaintext, valid2, err = decryptAndHash(&hs.ss, message.Ciphertext)
//	cs1, cs2 := split(&hs.ss)
//	return hs.ss.h, plaintext, (valid1 && valid2), cs1, cs2, err
//}
//
//func readMessageRegular(cs *cipherstate, message *MessageBuffer) (*cipherstate, []byte, bool, error) {
//	var err error
//	var plaintext []byte
//	var valid2 bool = false
//	/* No encrypted keys */
//	_, plaintext, valid2, err = decryptWithAd(cs, []byte{}, message.Ciphertext)
//	return cs, plaintext, valid2, err
//}
//
///* ---------------------------------------------------------------- *
// * PROCESSES                                                        *
// * ---------------------------------------------------------------- */
//
//func InitSession(initiator bool, prologue []byte, s Keypair, rs [32]byte) NoiseSession {
//	var session NoiseSession
//	psk := emptyKey
//	if initiator {
//		session.hs = initializeInitiator(prologue, s, rs, psk)
//	} else {
//		session.hs = initializeResponder(prologue, s, rs, psk)
//	}
//	session.i = initiator
//	session.mc = 0
//	return session
//}
//
//func SendMessage(session *NoiseSession, message []byte) (*NoiseSession, MessageBuffer, error) {
//	var err error
//	var messageBuffer MessageBuffer
//	if session.mc == 0 {
//		_, messageBuffer, err = writeMessageA(&session.hs, message)
//	}
//	if session.mc == 1 {
//		session.h, messageBuffer, session.cs1, session.cs2, err = writeMessageB(&session.hs, message)
//		session.hs = handshakestate{}
//	}
//	if session.mc > 1 {
//		if session.i {
//			_, messageBuffer, err = writeMessageRegular(&session.cs1, message)
//		} else {
//			_, messageBuffer, err = writeMessageRegular(&session.cs2, message)
//		}
//	}
//	session.mc = session.mc + 1
//	return session, messageBuffer, err
//}
//
//func RecvMessage(session *NoiseSession, message *MessageBuffer) (*NoiseSession, []byte, bool, error) {
//	var err error
//	var plaintext []byte
//	var valid bool
//	if session.mc == 0 {
//		_, plaintext, valid, err = readMessageA(&session.hs, message)
//	}
//	if session.mc == 1 {
//		session.h, plaintext, valid, session.cs1, session.cs2, err = readMessageB(&session.hs, message)
//		session.hs = handshakestate{}
//	}
//	if session.mc > 1 {
//		if session.i {
//			_, plaintext, valid, err = readMessageRegular(&session.cs2, message)
//		} else {
//			_, plaintext, valid, err = readMessageRegular(&session.cs1, message)
//		}
//	}
//	session.mc = session.mc + 1
//	return session, plaintext, valid, err
//}
