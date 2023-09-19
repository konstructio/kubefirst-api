/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package google

import (
	"context"
	"strings"

	"github.com/kubefirst/kubefirst-api/internal/types"
	providerConfig "github.com/kubefirst/runtime/pkg/providerConfigs"
	"github.com/rs/zerolog/log"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func BootstrapGoogleMgmtCluster(
	clientset *kubernetes.Clientset,
	cl *types.Cluster,
	config *providerConfig.ProviderConfig,
) error {

	err := providerConfig.BootstrapMgmtCluster(
		clientset,
		cl.GitProvider,
		cl.GitAuth.User,
		config.DestinationGitopsRepoURL,
		cl.GitProtocol,
		cl.CloudflareAuth.Token,
		cl.GoogleAuth.KeyFile, //AWS has no authentication method because we use roles
		cl.DnsProvider,
		cl.CloudProvider,
	)
	if err != nil {
		log.Fatal().Msgf("error in central function to create secrets: %s", err)
	}
	// Create secrets
	createSecrets := []*v1.Secret{}
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
