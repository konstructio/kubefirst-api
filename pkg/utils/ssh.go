package pkg

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"strings"

	"github.com/kubefirst/kubefirst-api/internal/gitlab"
	"github.com/kubefirst/kubefirst-api/pkg/types"
	"github.com/mikesmitty/edkey"
	log "github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh"
)

func CreateSSHKeyPair() (string, string, error) {
	// Generate a key pair
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate key pair: %w", err)
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
		return "", "", fmt.Errorf("failed to create public key: %w", err)
	}
	pubKey := ssh.MarshalAuthorizedKey(authKey)

	return string(privKey), string(pubKey), nil
}

func EvalSSHKey(req *types.EvalSSHKeyRequest) error {
	// For GitLab, we currently need to add an ssh key to the authenticating user
	if req.GitProvider == "gitlab" {
		gitlabClient, err := gitlab.NewGitLabClient(req.GitToken, req.GitlabGroupFlag)
		if err != nil {
			return fmt.Errorf("error creating gitlab client: %w", err)
		}
		keys, err := gitlabClient.GetUserSSHKeys()
		if err != nil {
			log.Error().Msgf("unable to check for ssh keys in gitlab: %s", err.Error())
			return fmt.Errorf("error checking for ssh keys in gitlab: %w", err)
		}

		keyName := "kbot-ssh-key"
		keyFound := false
		for _, key := range keys {
			if key.Title == keyName {
				if strings.Contains(key.Key, strings.TrimSuffix(viper.GetString("kbot.public-key"), "\n")) {
					log.Info().Msgf("ssh key %s already exists and key is up to date, continuing", keyName)
					keyFound = true
				} else {
					log.Warn().Msgf("ssh key %s already exists and key data has drifted - it will be recreated", keyName)
					err := gitlabClient.DeleteUserSSHKey(keyName)
					if err != nil {
						return fmt.Errorf("error deleting gitlab user ssh key %q: %w", keyName, err)
					}
				}
			}
		}
		if !keyFound {
			log.Info().Msgf("creating ssh key %s...", keyName)
			err := gitlabClient.AddUserSSHKey(keyName, viper.GetString("kbot.public-key"))
			if err != nil {
				log.Error().Msgf("error adding ssh key %s: %s", keyName, err.Error())
				return fmt.Errorf("error adding ssh key %s: %w", keyName, err)
			}
			viper.Set("kbot.gitlab-user-based-ssh-key-title", keyName)
			viper.WriteConfig()
		}
	}

	return nil
}
