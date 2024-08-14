/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
)

func (conf *Configuration) GetIamRole(roleName string) (*iam.GetRoleOutput, error) {
	// fmt.Println("looking up iam role: ", roleName) // todo add helpful logs about if found or not
	iamClient := iam.NewFromConfig(conf.Config)

	role, err := iamClient.GetRole(context.Background(), &iam.GetRoleInput{
		RoleName: aws.String(roleName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get IAM role %q: %w", roleName, err)
	}

	return role, nil
}
