/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k3s

import (
	"context"
	"strings"

	providerConfig "github.com/kubefirst/kubefirst-api/pkg/providerConfigs"
	pkgtypes "github.com/kubefirst/kubefirst-api/pkg/types"
	"github.com/rs/zerolog/log"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func BootstrapK3sMgmtCluster(clientset *kubernetes.Clientset, cl *pkgtypes.Cluster, destinationGitopsRepoURL string) error {
	err := providerConfig.BootstrapMgmtCluster(
		clientset,
		cl.GitProvider,
		cl.GitAuth.User,
		destinationGitopsRepoURL,
		cl.GitProtocol,
		cl.CloudflareAuth.Token,
		"",
		cl.DnsProvider,
		cl.CloudProvider,
		cl.GitAuth.Token,
		cl.GitAuth.PrivateKey,
	)
	if err != nil {
		log.Fatal().Msgf("error in central function to create secrets: %s", err)
		return err
	}

	var externalDnsToken string
	switch cl.DnsProvider {
	case "civo":
		externalDnsToken = cl.CivoAuth.Token
	case "vultr":
		externalDnsToken = cl.VultrAuth.Token
	case "digitalocean":
		externalDnsToken = cl.DigitaloceanAuth.Token
	case "aws":
		externalDnsToken = "implement with cluster management"
	case "google":
		externalDnsToken = "implement with cluster management"
	case "cloudflare":
		externalDnsToken = cl.CloudflareAuth.APIToken
	}

	// Create secrets
	createSecrets := []*v1.Secret{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "cloudflare-creds", Namespace: "argo"},
			Data: map[string][]byte{
				"origin-ca-api-key": []byte(cl.CloudflareAuth.OriginCaIssuerKey),
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "cloudflare-creds", Namespace: "atlantis"},
			Data: map[string][]byte{
				"origin-ca-api-key": []byte(cl.CloudflareAuth.OriginCaIssuerKey),
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "cloudflare-creds", Namespace: "chartmuseum"},
			Data: map[string][]byte{
				"origin-ca-api-key": []byte(cl.CloudflareAuth.OriginCaIssuerKey),
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "external-dns-secrets", Namespace: "external-dns"},
			Data: map[string][]byte{
				"token": []byte(externalDnsToken),
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "cloudflare-creds", Namespace: "kubefirst"},
			Data: map[string][]byte{
				"origin-ca-api-key": []byte(cl.CloudflareAuth.OriginCaIssuerKey),
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "cloudflare-creds", Namespace: "vault"},
			Data: map[string][]byte{
				"origin-ca-api-key": []byte(cl.CloudflareAuth.OriginCaIssuerKey),
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "crossplane-secrets", Namespace: "crossplane-system"},
			Data: map[string][]byte{
				"username": []byte(cl.GitAuth.User),
				"password": []byte(cl.GitAuth.Token),
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
