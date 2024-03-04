package pkg

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"os"

	"github.com/mikesmitty/edkey"
	"golang.org/x/crypto/ssh"
)

func CreateSshKeyPair() (string, string, error) {

	// Generate a key pair
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to generate key pair: %v\n", err)
		return "", "", err
	}

	// Encode private key in PEM format
	pemBlock := &pem.Block{
		Type:  "OPENSSH PRIVATE KEY",
		Bytes: edkey.MarshalED25519PrivateKey(privateKey),
	}
	privKey := pem.EncodeToMemory(pemBlock)

	// Encode public key in OpenSSH authorized_keys format
	authKey, err := ssh.NewPublicKey(publicKey)
	if err != nil {
		return "", "", err
	}
	pubKey := ssh.MarshalAuthorizedKey(authKey)

	return string(privKey), string(pubKey), nil
}
