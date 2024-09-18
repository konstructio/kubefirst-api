/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package aws

import (
	"encoding/base64"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/rs/zerolog/log"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/aws-iam-authenticator/pkg/token"
)

func NewClientset(cluster *eks.Cluster) (*kubernetes.Clientset, error) {
	log.Printf("%+v", cluster)
	gen, err := token.NewGenerator(true, false)
	if err != nil {
		return nil, fmt.Errorf("unable to create token generator: %w", err)
	}
	opts := &token.GetTokenOptions{
		ClusterID: aws.StringValue(cluster.Name),
	}
	tok, err := gen.GetWithOptions(opts)
	if err != nil {
		return nil, fmt.Errorf("unable to get token: %w", err)
	}
	ca, err := base64.StdEncoding.DecodeString(aws.StringValue(cluster.CertificateAuthority.Data))
	if err != nil {
		return nil, fmt.Errorf("unable to decode certificate authority: %w", err)
	}
	clientset, err := kubernetes.NewForConfig(
		&rest.Config{
			Host:        aws.StringValue(cluster.Endpoint),
			BearerToken: tok.Token,
			TLSClientConfig: rest.TLSClientConfig{
				CAData: ca,
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create clientset: %w", err)
	}
	return clientset, nil
}

func NewRestConfig(cluster *eks.Cluster) (*rest.Config, error) {
	log.Info().Msgf("cluster info: %#v", cluster)

	gen, err := token.NewGenerator(true, false)
	if err != nil {
		return nil, fmt.Errorf("error creating token generator: %w", err)
	}
	opts := &token.GetTokenOptions{
		ClusterID: aws.StringValue(cluster.Name),
	}
	tok, err := gen.GetWithOptions(opts)
	if err != nil {
		return nil, fmt.Errorf("error getting token: %w", err)
	}
	ca, err := base64.StdEncoding.DecodeString(aws.StringValue(cluster.CertificateAuthority.Data))
	if err != nil {
		return nil, fmt.Errorf("error decoding certificate authority: %w", err)
	}
	restConfig := &rest.Config{
		Host:        aws.StringValue(cluster.Endpoint),
		BearerToken: tok.Token,
		TLSClientConfig: rest.TLSClientConfig{
			CAData: ca,
		},
	}

	return restConfig, nil
}
