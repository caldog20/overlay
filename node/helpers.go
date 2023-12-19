package node

import (
	"encoding/base64"
	"encoding/gob"
	"errors"
	"github.com/flynn/noise"
	"os"
)

func LoadKeyFromDisk() (noise.DHKey, error) {
	var key Key
	var noise noise.DHKey

	keyfile, err := os.Open("~/overlay.keypair")
	if err != nil {
		return noise, errors.New("File not found")
	}

	err = gob.NewDecoder(keyfile).Decode(&key)
	if err != nil {
		return noise, errors.New("error decoding file")
	}

	priv, err := base64.StdEncoding.DecodeString(key.Private)
	if err != nil {
		return noise, errors.New("error decoding private key")
	}
	pub, err := base64.StdEncoding.DecodeString(key.Public)
	if err != nil {
		return noise, errors.New("error decoding public key")
	}

	noise.Public = pub
	noise.Private = priv

	return noise, nil
}

func StoreKeyToDisk(keyPair noise.DHKey) error {
	var key Key

	keyfile, err := os.Create("~/.overlay.keypair")
	if err != nil {
		return err
	}
	keyfile.Seek(0, 0)

	key.Private = base64.StdEncoding.EncodeToString(keyPair.Private)
	key.Public = base64.StdEncoding.EncodeToString(keyPair.Public)

	err = gob.NewEncoder(keyfile).Encode(key)
	if err != nil {
		return err
	}

	return nil
}
