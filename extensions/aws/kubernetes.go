/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package aws

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	eksTypes "github.com/aws/aws-sdk-go-v2/service/eks/types"
	"github.com/konstructio/kubefirst-api/internal/k8s"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/aws-iam-authenticator/pkg/token"
)

// CreateEKSKubeconfig
func CreateEKSKubeconfig(awsConfig *aws.Config, clusterName string) *k8s.KubernetesClient {
	eksSvc := eks.NewFromConfig(*awsConfig)

	clusterInput := &eks.DescribeClusterInput{
		Name: aws.String(clusterName),
	}

	eksClusterInfo, err := eksSvc.DescribeCluster(context.Background(), clusterInput)
	if err != nil {
		return &k8s.KubernetesClient{}
	}

	clientset, restConfig, err := newEKSConfig(eksClusterInfo.Cluster)
	if err != nil {
		return &k8s.KubernetesClient{}
	}

	return &k8s.KubernetesClient{
		Clientset:      clientset,
		RestConfig:     restConfig,
		KubeConfigPath: "",
	}
}

// newEKSConfig
func newEKSConfig(cluster *eksTypes.Cluster) (kubernetes.Interface, *rest.Config, error) {
	gen, err := token.NewGenerator(true, false)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating token generator: %w", err)
	}
	opts := &token.GetTokenOptions{
		ClusterID: *aws.String(*cluster.Name),
	}
	tok, err := gen.GetWithOptions(opts)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting token: %w", err)
	}
	ca, err := base64.StdEncoding.DecodeString(*aws.String(*cluster.CertificateAuthority.Data))
	if err != nil {
		return nil, nil, fmt.Errorf("error decoding certificate authority: %w", err)
	}
	clientset, err := kubernetes.NewForConfig(
		&rest.Config{
			Host:        *aws.String(*cluster.Endpoint),
			BearerToken: tok.Token,
			TLSClientConfig: rest.TLSClientConfig{
				CAData: ca,
			},
		},
	)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating clientset: %w", err)
	}

	restConfig := &rest.Config{
		Host:        *aws.String(*cluster.Endpoint),
		BearerToken: tok.Token,
		TLSClientConfig: rest.TLSClientConfig{
			CAData: ca,
		},
	}

	return clientset, restConfig, nil
}
