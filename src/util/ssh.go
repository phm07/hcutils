package util

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"golang.org/x/crypto/ssh"
	"os"
)

func GenerateSSHKeypair() (ssh.Signer, string, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, "", err
	}

	err = privateKey.Validate()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		return nil, "", err
	}

	pub, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, "", err
	}

	pubSSHFormat := ssh.MarshalAuthorizedKey(pub)

	return signer, string(pubSSHFormat), nil
}
