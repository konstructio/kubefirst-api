/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package aws

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"slices"
	"strconv"
	"time"

	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	route53Types "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/konstructio/kubefirst-api/internal/constants"
	"github.com/konstructio/kubefirst-api/internal/env"
	"github.com/konstructio/kubefirst-api/internal/utils"
	log "github.com/rs/zerolog/log"
)

type TXTRecord struct {
	Name          string
	Value         string
	SetIdentifier *string
	Weight        *int64
	TTL           int64
}

// ARecord stores Route53 A record data
type ARecord struct {
	Name        string
	RecordType  string
	TTL         *int64
	AliasTarget *route53Types.AliasTarget
}

const (
	validationRecordSubdomain string = "kubefirst-liveness-test"
	validationRecordValue     string = "domain record propagated"
)

// New instantiates a new AWS configuration
func New() (*Configuration, error) {
	env, err := env.GetEnv(constants.SilenceGetEnv)
	if err != nil {
		return nil, fmt.Errorf("unable to get environment variables: %w", err)
	}

	awsClient, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion(env.AWSRegion),
		config.WithSharedConfigProfile(env.AWSProfile),
	)
	if err != nil {
		log.Error().Msgf("Could not create AWS config: %s", err.Error())
		return nil, fmt.Errorf("unable to create aws client: %w", err)
	}

	return &Configuration{Config: awsClient}, nil
}

// Route53AlterResourceRecord simplifies manipulation of Route53 records
func (conf *Configuration) Route53AlterResourceRecord(r *Route53AlterResourceRecord) (*route53.ChangeResourceRecordSetsOutput, error) {
	route53Client := route53.NewFromConfig(conf.Config)

	log.Info().Msgf("validating hostedZoneId %q", r.hostedZoneID)
	log.Info().Msgf("validating route53RecordName %q", r.route53RecordName)

	record, err := route53Client.ChangeResourceRecordSets(
		context.Background(),
		r.input)
	if err != nil {
		log.Error().Msgf("error changing resource record sets: %s", err.Error())
		return nil, fmt.Errorf("error changing resource record sets for record: %w", err)
	}

	return record, nil
}

// Route53ListARecords retrieves all DNS A records for a hosted zone
func (conf *Configuration) Route53ListARecords(hostedZoneID string) ([]ARecord, error) {
	route53Client := route53.NewFromConfig(conf.Config)

	recordSets, err := route53Client.ListResourceRecordSets(
		context.Background(),
		&route53.ListResourceRecordSetsInput{HostedZoneId: &hostedZoneID},
	)
	if err != nil {
		return nil, fmt.Errorf("error listing resource record sets for hosted zone ID %q: %w", hostedZoneID, err)
	}

	aRecords := make([]ARecord, 0, len(recordSets.ResourceRecordSets))
	for _, recordSet := range recordSets.ResourceRecordSets {
		if recordSet.Type == route53Types.RRTypeA {
			record := ARecord{
				Name:       *recordSet.Name,
				RecordType: "A",
				AliasTarget: &route53Types.AliasTarget{
					HostedZoneId:         recordSet.AliasTarget.HostedZoneId,
					DNSName:              recordSet.AliasTarget.DNSName,
					EvaluateTargetHealth: true,
				},
			}
			aRecords = append(aRecords, record)
		}
	}
	return aRecords, nil
}

// Route53ListTXTRecords retrieves all DNS TXT record type for a hosted zone
func (conf *Configuration) Route53ListTXTRecords(hostedZoneID string) ([]TXTRecord, error) {
	route53Client := route53.NewFromConfig(conf.Config)

	recordSets, err := route53Client.ListResourceRecordSets(
		context.Background(),
		&route53.ListResourceRecordSetsInput{HostedZoneId: &hostedZoneID},
	)
	if err != nil {
		return nil, fmt.Errorf("error listing resource record sets for hosted zone ID %q: %w", hostedZoneID, err)
	}

	txtRecords := make([]TXTRecord, 0, len(recordSets.ResourceRecordSets))
	for _, recordSet := range recordSets.ResourceRecordSets {
		log.Debug().Msgf("Record Name: %s", *recordSet.Name)

		if recordSet.Type == route53Types.RRTypeTxt {
			for _, resourceRecord := range recordSet.ResourceRecords {
				if recordSet.SetIdentifier != nil && recordSet.Weight != nil {
					record := TXTRecord{
						Name:          *recordSet.Name,
						Value:         *resourceRecord.Value,
						SetIdentifier: recordSet.SetIdentifier,
						TTL:           *recordSet.TTL,
						Weight:        recordSet.Weight,
					}
					txtRecords = append(txtRecords, record)
					continue
				}
				record := TXTRecord{
					Name:  *recordSet.Name,
					Value: *resourceRecord.Value,
					TTL:   *recordSet.TTL,
				}
				txtRecords = append(txtRecords, record)
			}
		}
	}

	return txtRecords, nil
}

// TestHostedZoneLivenessWithTxtRecords determines whether or not a target hosted zone is initialized and
// ready to accept records
func (conf *Configuration) TestHostedZoneLivenessWithTxtRecords(hostedZoneName string) (bool, error) {
	// Get hosted zone ID
	hostedZoneID, err := conf.GetHostedZoneID(hostedZoneName)
	if err != nil {
		return false, fmt.Errorf("error getting hosted zone ID for hosted zone name %q: %w", hostedZoneName, err)
	}

	// Format fqdn of target record for validation
	route53RecordName := fmt.Sprintf("%s.%s", validationRecordSubdomain, hostedZoneName)

	// Get all txt records for hosted zone
	records, err := conf.Route53ListTXTRecords(hostedZoneID)
	if err != nil {
		return false, fmt.Errorf("error listing txt records for hosted zone %q: %w", hostedZoneName, err)
	}

	// Construct a []string of record names
	foundRecordNames := make([]string, 0, len(records))
	for _, rec := range records {
		foundRecordNames = append(foundRecordNames, rec.Name)
	}

	// Determine whether or not the record exists, create if it doesn't
	if utils.FindStringInSlice(foundRecordNames, route53RecordName) {
		log.Info().Msgf("record %q exists - zone validated", route53RecordName)
		return true, nil
	}

	log.Info().Msgf("record %q does not exist, creating...", route53RecordName)

	// Construct resource record alter and create record
	alt := Route53AlterResourceRecord{
		hostedZoneName:    hostedZoneName,
		hostedZoneID:      hostedZoneID,
		route53RecordName: route53RecordName,
		input: &route53.ChangeResourceRecordSetsInput{
			ChangeBatch: &route53Types.ChangeBatch{
				Changes: []route53Types.Change{{
					Action: "UPSERT",
					ResourceRecordSet: &route53Types.ResourceRecordSet{
						Name: aws.String(route53RecordName),
						Type: "TXT",
						ResourceRecords: []route53Types.ResourceRecord{{
							Value: aws.String(strconv.Quote(validationRecordValue)),
						}},
						TTL:           aws.Int64(10),
						Weight:        aws.Int64(100),
						SetIdentifier: aws.String("CREATE sanity check for kubefirst installation"),
					},
				}},
				Comment: aws.String("CREATE sanity check dns record."),
			},
			HostedZoneId: aws.String(hostedZoneID),
		},
	}
	record, err := conf.Route53AlterResourceRecord(&alt)
	if err != nil {
		return false, fmt.Errorf("unable to alter resource DNS record for %q: %w", route53RecordName, err)
	}

	log.Info().Msgf("record created and is in state: %s", record.ChangeInfo.Status)

	// Wait for record
	ch := make(chan bool, 1)
	retries := 10
	retryInterval := 10
	duration := (retries * retryInterval)

	go func() {
		log.Info().Msgf("waiting on %s domain validation record creation for %v seconds...", route53RecordName, duration)

		for i := 1; i < retries; i++ {
			ips, err := net.LookupTXT(route53RecordName)

			// If the record was found, return to the caller
			if err == nil {
				log.Info().Msgf("found %q in TXT record values with IP: %v", route53RecordName, ips)
				ch <- true
				break
			}

			// If there was an error looking up the record then retry with a backup resolver
			ips, err = utils.BackupResolver.LookupTXT(context.Background(), route53RecordName)

			// And check too if the record was found
			if err == nil {
				log.Info().Msgf("found %q in TXT record values with IP: %v", route53RecordName, ips)
				ch <- true
				break
			}

			// If the record was not found, log the error and retry
			log.Warn().Msgf("attempt %d of %d resolving %q, retrying in %ds", i, retries, route53RecordName, retryInterval)
			time.Sleep(time.Duration(retryInterval) * time.Second)
		}

		// If the record was not found after all retries, close the channel
		ch <- false
		close(ch)
	}()

	for {
		select {
		case found, ok := <-ch:
			if !ok {
				return found, errors.New("timed out waiting for domain check - check zone for presence of record and retry validation")
			}
			return found, nil
		case <-time.After(time.Duration(duration) * time.Second):
			return false, errors.New("timed out waiting for domain check - check zone for presence of record and retry validation")
		}
	}
}

func NewAwsV2(region string) (aws.Config, error) {
	// todo these should also be supported flags
	profile := os.Getenv("AWS_PROFILE")

	awsClient, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion(region),
		config.WithSharedConfigProfile(profile),
	)
	if err != nil {
		return aws.Config{}, fmt.Errorf("unable to create aws client for region %q: %w", region, err)
	}

	return awsClient, nil
}

func NewAwsV3(region, accessKeyID, secretAccessKey, sessionToken string) (aws.Config, error) {
	awsClient, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			accessKeyID,
			secretAccessKey,
			sessionToken,
		)),
	)
	if err != nil {
		return aws.Config{}, fmt.Errorf("unable to create aws client for region %q with provided credentials: %w", region, err)
	}

	return awsClient, nil
}

// GetRegions lists all available regions
func (conf *Configuration) GetRegions() ([]string, error) {
	ec2Client := ec2.NewFromConfig(conf.Config)

	regions, err := ec2Client.DescribeRegions(context.Background(), &ec2.DescribeRegionsInput{})
	if err != nil {
		return []string{}, fmt.Errorf("error listing regions: %w", err)
	}

	regionList := make([]string, 0, len(regions.Regions))
	for _, region := range regions.Regions {
		regionList = append(regionList, *region.RegionName)
	}

	return regionList, nil
}

var SSMTypes = map[string]string{
	"AL2_x86_64":                 "/aws/service/eks/optimized-ami/1.31/amazon-linux-2/recommended/image_id",
	"AL2_ARM_64":                 "/aws/service/eks/optimized-ami/1.31/amazon-linux-2-arm64/recommended/image_id",
	"BOTTLEROCKET_ARM_64":        "/aws/service/bottlerocket/aws-k8s-1.31/arm64/latest/image_id",
	"BOTTLEROCKET_x86_64":        "/aws/service/bottlerocket/aws-k8s-1.31/x86_64/latest/image_id",
	"BOTTLEROCKET_ARM_64_NVIDIA": "/aws/service/bottlerocket/aws-k8s-1.31-nvidia/arm64/latest/image_id",
	"BOTTLEROCKET_x86_64_NVIDIA": "/aws/service/bottlerocket/aws-k8s-1.31-nvidia/x86_64/latest/image_id",
}

func (conf *Configuration) ListInstanceSizesForRegion(ctx context.Context, amiType string) ([]string, error) {
	ec2Client := ec2.NewFromConfig(conf.Config)

	ssmClient := ssm.NewFromConfig(conf.Config)
	paginator := ec2.NewDescribeInstanceTypesPaginator(ec2Client, &ec2.DescribeInstanceTypesInput{})

	ssmParameterName, ok := SSMTypes[amiType]
	if !ok {
		return nil, fmt.Errorf("invalid ami type: %s", amiType)
	}

	amiID, err := getLatestAMIFromSSM(ctx, ssmClient, ssmParameterName)
	if err != nil {
		return nil, fmt.Errorf("failed to get AMI ID from SSM: %w", err)
	}

	architecture, err := getAMIArchitecture(ctx, ec2Client, amiID)
	if err != nil {
		return nil, fmt.Errorf("failed to get AMI architecture: %w", err)
	}

	instanceTypes, err := getSupportedInstanceTypes(ctx, paginator, architecture)
	if err != nil {
		return nil, fmt.Errorf("failed to get supported instance types: %w", err)
	}

	return instanceTypes, nil
}

type ssmClienter interface {
	GetParameter(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
}

func getLatestAMIFromSSM(ctx context.Context, ssmClient ssmClienter, parameterName string) (string, error) {
	input := &ssm.GetParameterInput{
		Name: aws.String(parameterName),
	}
	output, err := ssmClient.GetParameter(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failure when fetching parameters: %w", err)
	}

	if output == nil || output.Parameter == nil || output.Parameter.Value == nil {
		return "", fmt.Errorf("invalid parameter value found for %q", parameterName)
	}

	return *output.Parameter.Value, nil
}

type ec2Clienter interface {
	DescribeImages(ctx context.Context, params *ec2.DescribeImagesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeImagesOutput, error)
}

func getAMIArchitecture(ctx context.Context, ec2Client ec2Clienter, amiID string) (string, error) {
	input := &ec2.DescribeImagesInput{
		ImageIds: []string{amiID},
	}
	output, err := ec2Client.DescribeImages(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to describe images: %w", err)
	}

	if len(output.Images) == 0 {
		return "", fmt.Errorf("no images found for AMI ID: %s", amiID)
	}

	return string(output.Images[0].Architecture), nil
}

type paginator interface {
	HasMorePages() bool
	NextPage(ctx context.Context, optFns ...func(*ec2.Options)) (*ec2.DescribeInstanceTypesOutput, error)
}

func getSupportedInstanceTypes(ctx context.Context, p paginator, architecture string) ([]string, error) {
	var instanceTypes []string
	for p.HasMorePages() {
		page, err := p.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to load next pages for instance types: %w", err)
		}

		for _, instanceType := range page.InstanceTypes {
			if slices.Contains(instanceType.ProcessorInfo.SupportedArchitectures, ec2Types.ArchitectureType(architecture)) {
				instanceTypes = append(instanceTypes, string(instanceType.InstanceType))
			}
		}
	}
	return instanceTypes, nil
}
