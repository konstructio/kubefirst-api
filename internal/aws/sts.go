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

	"github.com/YakDriver/regexache"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

func (conf *Configuration) GetCallerIdentity() (*sts.GetCallerIdentityOutput, error) {
	stsClient := sts.NewFromConfig(conf.Config)
	iamCaller, err := stsClient.GetCallerIdentity(
		context.Background(),
		&sts.GetCallerIdentityInput{},
	)
	if err != nil {
		fmt.Printf("error: could not get caller identity %s", err)
	}

	return iamCaller, err
}

// sourceIAMRoleARN Given an STS ARN returns the ARN for the source IAM role
// or returns User's arn
func (conf *Configuration) sourceIAMRoleARN(rawARN string) (string, error) {
	iamClient := iam.NewFromConfig(conf.Config)

	parsedARN, err := arn.Parse(rawARN)

	if err != nil {
		return "", err
	}

	reAssume := regexache.MustCompile(`^assumed-role/.{1,}/.{2,}`)

	if !reAssume.MatchString(parsedARN.Resource) || parsedARN.Service != "sts" {
		return rawARN, nil
	}

	parts := strings.Split(parsedARN.Resource, "/")

	if len(parts) < 3 {
		return "", nil
	}

	iamInput := &iam.GetRoleInput{
		RoleName: aws.String(parts[len(parts)-2]),
	}

	output, err := iamClient.GetRole(context.Background(), iamInput)
	if err != nil {
		return "", err
	}

	return *output.Role.Arn, nil
}
