/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package civo

import (
	"context"
	"fmt"
	"strings"

	"github.com/kubefirst/kubefirst-api/internal/types"
	config "github.com/kubefirst/runtime/pkg/providerConfigs"
	"github.com/rs/zerolog/log"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func BootstrapCivoMgmtCluster(clientset *kubernetes.Clientset, cl *types.Cluster, config *config.ProviderConfig) error {

	secretData := map[string][]byte{}

	if cl.GitProtocol == "https" {
		// http basic auth
		secretData = map[string][]byte{
			"type":     []byte("git"),
			"name":     []byte(fmt.Sprintf("%s-gitops", cl.GitUser)),
			"url":      []byte(config.DestinationGitopsRepoURL),
			"username": []byte(cl.GitUser),
			"password": []byte([]byte(fmt.Sprintf(cl.GitToken))),
		}
	} else {
		// ssh
		secretData = map[string][]byte{
			"type":          []byte("git"),
			"name":          []byte(fmt.Sprintf("%s-gitops", cl.GitUser)),
			"url":           []byte(config.DestinationGitopsRepoURL),
			"sshPrivateKey": []byte(cl.PrivateKey),
		}
	}

	// Create secrets
	createSecrets := []*v1.Secret{
		// argocd
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "repo-credentials-template",
				Namespace:   "argocd",
				Annotations: map[string]string{"managed-by": "argocd.argoproj.io"},
				Labels:      map[string]string{"argocd.argoproj.io/secret-type": "repository"},
			},
			Data: secretData,
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "civo-creds", Namespace: "external-dns"},
			Data: map[string][]byte{
				"civo-token":   []byte(cl.CivoAuth.Token),
				"cf-api-token": []byte(cl.CloudflareApiToken),
			},
		},
	}
	for _, secret := range createSecrets {
		_, err := clientset.CoreV1().Secrets(secret.ObjectMeta.Namespace).Get(context.TODO(), secret.ObjectMeta.Name, metav1.GetOptions{})
		if err == nil {
			log.Info().Msgf("kubernetes secret %s/%s already created - skipping", secret.Namespace, secret.Name)
		} else if strings.Contains(err.Error(), "not found") {
			_, err = clientset.CoreV1().Secrets(secret.ObjectMeta.Namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
			if err != nil {
				log.Fatal().Msgf("error creating kubernetes secret %s/%s: %s", secret.Namespace, secret.Name, err)
			}
			log.Info().Msgf("created kubernetes secret: %s/%s", secret.Namespace, secret.Name)
		}
	}

	return nil
}
