/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ecr"
)

func (conf *Configuration) GetECRAuthToken() (string, error) {
	fmt.Println("getting ecr auth token")
	ecrClient := ecr.NewFromConfig(conf.Config)

	token, err := ecrClient.GetAuthorizationToken(context.Background(), &ecr.GetAuthorizationTokenInput{})
	if err != nil {
		return "", fmt.Errorf("failed to get ECR authorization token: %w", err)
	}

	return *token.AuthorizationData[0].AuthorizationToken, nil
}
