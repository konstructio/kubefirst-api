/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package aws

import (
	"context"
	"fmt"
	"strings"

	"github.com/kubefirst/kubefirst-api/internal/types"
	"github.com/kubefirst/runtime/pkg/aws"
	providerConfig "github.com/kubefirst/runtime/pkg/providerConfigs"
	"github.com/rs/zerolog/log"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func BootstrapAWSMgmtCluster(
	clientset *kubernetes.Clientset,
	cl *types.Cluster,
	destinationGitopsRepoURL string,
	awsClient *aws.AWSConfiguration,
) error {
	

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

	//flag out the ecr token
	if cl.ECR {
		ecrToken, err := awsClient.GetECRAuthToken()
		if err != nil {
			return err
		}

		dockerConfigString := fmt.Sprintf(`{"auths": {"%s": {"auth": "%s"}}}`, fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com", cl.AWSAccountId, cl.CloudRegion), ecrToken)
		dockerCfgSecret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "docker-config", Namespace: "argo"},
			Data:       map[string][]byte{"config.json": []byte(dockerConfigString)},
			Type:       "Opaque",
		}
		_, err = clientset.CoreV1().Secrets(dockerCfgSecret.ObjectMeta.Namespace).Create(context.TODO(), dockerCfgSecret, metav1.CreateOptions{})
		if err != nil {
			log.Info().Msgf("error creating kubernetes secret %s/%s: %s", dockerCfgSecret.Namespace, dockerCfgSecret.Name, err)
			return err
		}
	}

	return nil
}
