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
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	route53Types "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/kubefirst/kubefirst-api/internal/constants"
	"github.com/kubefirst/kubefirst-api/internal/env"
	"github.com/kubefirst/kubefirst-api/internal/utils"
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

// Create a single configuration instance to act as an interface to the AWS client
var Conf AWSConfiguration = AWSConfiguration{
	Config: NewAws(),
}

// NewAws instantiates a new AWS configuration
func NewAws() aws.Config {
	env, _ := env.GetEnv(constants.SilenceGetEnv)

	awsClient, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion(env.AWSRegion),
		config.WithSharedConfigProfile(env.AWSProfile),
	)
	if err != nil {
		log.Error().Msgf("Could not create AWS config: %s", err.Error())
	}

	return awsClient
}

// Route53AlterResourceRecord simplifies manipulation of Route53 records
func (conf *AWSConfiguration) Route53AlterResourceRecord(r *AWSRoute53AlterResourceRecord) (*route53.ChangeResourceRecordSetsOutput, error) {
	route53Client := route53.NewFromConfig(conf.Config)

	log.Info().Msgf("validating hostedZoneId %s", r.hostedZoneID)
	log.Info().Msgf("validating route53RecordName %s", r.route53RecordName)
	record, err := route53Client.ChangeResourceRecordSets(
		context.Background(),
		r.input)
	if err != nil {
		log.Warn().Msgf("%s", err)
		return &route53.ChangeResourceRecordSetsOutput{}, err
	}
	return record, nil
}

// Route53ListARecords retrieves all DNS A records for a hosted zone
func (conf *AWSConfiguration) Route53ListARecords(hostedZoneId string) ([]AWSARecord, error) {
	route53Client := route53.NewFromConfig(conf.Config)
	recordSets, err := route53Client.ListResourceRecordSets(context.Background(), &route53.ListResourceRecordSetsInput{
		HostedZoneId: &hostedZoneId,
	})
	if err != nil {
		return []AWSARecord{}, err
	}
	var aRecords []AWSARecord
	for _, recordSet := range recordSets.ResourceRecordSets {
		if recordSet.Type == route53Types.RRTypeA {
			record := AWSARecord{
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
func (conf *AWSConfiguration) Route53ListTXTRecords(hostedZoneId string) ([]AWSTXTRecord, error) {
	route53Client := route53.NewFromConfig(conf.Config)
	recordSets, err := route53Client.ListResourceRecordSets(context.Background(), &route53.ListResourceRecordSetsInput{
		HostedZoneId: &hostedZoneId,
	})
	if err != nil {
		return []AWSTXTRecord{}, err
	}
	var txtRecords []AWSTXTRecord
	for _, recordSet := range recordSets.ResourceRecordSets {
		log.Debug().Msgf("Record Name: %s", *recordSet.Name)
		if recordSet.Type == route53Types.RRTypeTxt {
			for _, resourceRecord := range recordSet.ResourceRecords {
				if recordSet.SetIdentifier != nil && recordSet.Weight != nil {
					record := AWSTXTRecord{
						Name:          *recordSet.Name,
						Value:         *resourceRecord.Value,
						SetIdentifier: recordSet.SetIdentifier,
						TTL:           *recordSet.TTL,
						Weight:        recordSet.Weight,
					}
					txtRecords = append(txtRecords, record)
					continue
				}
				record := AWSTXTRecord{
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
func (conf *AWSConfiguration) TestHostedZoneLivenessWithTxtRecords(hostedZoneName string) (bool, error) {
	// Get hosted zone ID
	hostedZoneID, err := conf.GetHostedZoneID(hostedZoneName)
	if err != nil {
		return false, err
	}

	// Format fqdn of target record for validation
	route53RecordName := fmt.Sprintf("%s.%s", validationRecordSubdomain, hostedZoneName)

	// Get all txt records for hosted zone
	records, err := conf.Route53ListTXTRecords(hostedZoneID)
	if err != nil {
		return false, err
	}

	// Construct a []string of record names
	foundRecordNames := make([]string, 0)
	for _, rec := range records {
		foundRecordNames = append(foundRecordNames, rec.Name)
	}

	// Determine whether or not the record exists, create if it doesn't
	switch utils.FindStringInSlice(foundRecordNames, route53RecordName) {
	case true:
		log.Info().Msgf("record %s exists - zone validated", route53RecordName)
		return true, nil
	case false:
		log.Info().Msgf("record %s does not exist, creating...", route53RecordName)

		// Construct resource record alter and create record
		alt := AWSRoute53AlterResourceRecord{
			hostedZoneName:    hostedZoneName,
			hostedZoneID:      hostedZoneID,
			route53RecordName: route53RecordName,
			input: &route53.ChangeResourceRecordSetsInput{
				ChangeBatch: &route53Types.ChangeBatch{
					Changes: []route53Types.Change{
						{
							Action: "UPSERT",
							ResourceRecordSet: &route53Types.ResourceRecordSet{
								Name: aws.String(route53RecordName),
								Type: "TXT",
								ResourceRecords: []route53Types.ResourceRecord{
									{
										Value: aws.String(strconv.Quote(validationRecordValue)),
									},
								},
								TTL:           aws.Int64(10),
								Weight:        aws.Int64(100),
								SetIdentifier: aws.String("CREATE sanity check for kubefirst installation"),
							},
						},
					},
					Comment: aws.String("CREATE sanity check dns record."),
				},
				HostedZoneId: aws.String(hostedZoneID),
			},
		}
		record, err := conf.Route53AlterResourceRecord(&alt)
		if err != nil {
			return false, err
		}
		log.Info().Msgf("record created and is in state: %s", record.ChangeInfo.Status)

		// Wait for record
		ch := make(chan bool, 1)
		retries := 10
		retryInterval := 10
		duration := (retries * retryInterval)
		log.Info().Msgf("waiting on %s domain validation record creation for %v seconds...", route53RecordName, duration)
		go func() {
			for i := 1; i < retries; i++ {
				ips, err := net.LookupTXT(route53RecordName)
				if err != nil {
					ips, err = utils.BackupResolver.LookupTXT(context.Background(), route53RecordName)
				}
				if err != nil {
					log.Warn().Msgf("attempt %v of %v resolving %s, retrying in %vs", i, retries, route53RecordName, retryInterval)
					time.Sleep(time.Duration(int32(retryInterval)) * time.Second)
				} else {
					for _, ip := range ips {
						// todo check ip against route53RecordValue in some capacity so we can pivot the value for testing
						log.Info().Msgf("%s. in TXT record value: %s", route53RecordName, ip)
						ch <- true
					}
				}
			}
		}()
		for {
			select {
			case found, ok := <-ch:
				if !ok {
					return found, errors.New("timed out waiting for domain check - check zone for presence of record and retry validation")
				}
				if ok {
					return found, nil
				}
			case <-time.After(time.Duration(int32(duration)) * time.Second):
				return false, errors.New("timed out waiting for domain check - check zone for presence of record and retry validation")
			}
		}
	}
	return false, err
}

func NewAwsV2(region string) aws.Config {
	// todo these should also be supported flags
	profile := os.Getenv("AWS_PROFILE")

	awsClient, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion(region),
		config.WithSharedConfigProfile(profile),
	)
	if err != nil {
		log.Error().Msg("unable to create aws client")
	}

	return awsClient
}

func NewAwsV3(region string, accessKeyID string, secretAccessKey string, sessionToken string) aws.Config {
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
		log.Error().Msg("unable to create aws client")
	}

	return awsClient
}

// GetRegions lists all available regions
func (conf *AWSConfiguration) GetRegions(region string) ([]string, error) {
	var regionList []string

	ec2Client := ec2.NewFromConfig(conf.Config)

	regions, err := ec2Client.DescribeRegions(context.Background(), &ec2.DescribeRegionsInput{})
	if err != nil {
		return []string{}, fmt.Errorf("error listing regions: %s", err)
	}

	for _, region := range regions.Regions {
		regionList = append(regionList, *region.RegionName)
	}

	return regionList, nil
}

func (conf *AWSConfiguration) ListInstanceSizesForRegion() ([]string, error) {

	ec2Client := ec2.NewFromConfig(conf.Config)

	sizes, err := ec2Client.DescribeInstanceTypeOfferings(context.Background(), &ec2.DescribeInstanceTypeOfferingsInput{})

	if err != nil {
		return nil, err
	}

	var instanceNames []string
	for _, size := range sizes.InstanceTypeOfferings {
		instanceNames = append(instanceNames, string(size.InstanceType))
	}

	return instanceNames, nil
}
