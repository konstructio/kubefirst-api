package aws

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsinternal "github.com/kubefirst/kubefirst-api/internal/aws"
	"github.com/rs/zerolog/log"
)

func NewEKSServiceAccountClientV1() aws.Config {
	// variables are automatically available in the pod through EKS
	region := os.Getenv("AWS_REGION")
	roleArn := os.Getenv("AWS_ROLE_ARN")
	tokenFilePath := os.Getenv("AWS_WEB_IDENTITY_TOKEN_FILE")

	token, err := ioutil.ReadFile(tokenFilePath)
	if err != nil {
		panic(err.Error())
	}
	fmt.Println(fmt.Sprintf("authenticating as role arn: %s from service account", roleArn))

	awsClient, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			string(token),
			"",
			string(token),
		)),
	)
	if err != nil {
		log.Error().Msg("unable to create aws client")
	}

	return awsClient
}

type AWSConfiguration = awsinternal.AWSConfiguration
type QuotaDetailResponse = awsinternal.QuotaDetailResponse

var NewAwsV2 = awsinternal.NewAwsV2
