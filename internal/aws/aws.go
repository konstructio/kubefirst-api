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
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	route53Types "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/kubefirst/kubefirst-api/internal/env"
	"github.com/kubefirst/kubefirst-api/internal/utils"
	log "github.com/sirupsen/logrus"
)

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
	env, _ := env.GetEnv()
	
	awsClient, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion(env.AWSRegion),
		config.WithSharedConfigProfile(env.AWSProfile),
	)
	if err != nil {
		log.Errorf("Could not create AWS config: %s", err.Error())
	}

	return awsClient
}

// GetHostedZoneID returns the id of a hosted zone based on its domain (name)
func (conf *AWSConfiguration) GetHostedZoneID(domain string) (string, error) {
	route53Client := route53.NewFromConfig(conf.Config)
	zones, err := route53Client.ListHostedZones(context.Background(), &route53.ListHostedZonesInput{})
	if err != nil {
		return "", err
	}
	for _, zone := range zones.HostedZones {
		if *zone.Name == domain {
			return *zone.Id, nil
		}
	}
	return "", fmt.Errorf(fmt.Sprintf("could not find a hosted zone for: %s", domain))
}

// Route53AlterResourceRecord simplifies manipulation of Route53 records
func (conf *AWSConfiguration) Route53AlterResourceRecord(r *AWSRoute53AlterResourceRecord) (*route53.ChangeResourceRecordSetsOutput, error) {
	route53Client := route53.NewFromConfig(conf.Config)

	log.Infof("validating hostedZoneId %s", r.hostedZoneID)
	log.Infof("validating route53RecordName %s", r.route53RecordName)
	record, err := route53Client.ChangeResourceRecordSets(
		context.Background(),
		r.input)
	if err != nil {
		log.Warnf("%s", err)
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
		log.Debugf("Record Name: %s", *recordSet.Name)
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

// TestHostedZoneLiveness determines whether or not a target hosted zone is initialized and
// ready to accept records
func (conf *AWSConfiguration) TestHostedZoneLiveness(hostedZoneName string) (bool, error) {
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
		log.Infof("record %s exists - zone validated", route53RecordName)
		return true, nil
	case false:
		log.Infof("record %s does not exist, creating...", route53RecordName)

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
		log.Infof("record created and is in state: %s", record.ChangeInfo.Status)

		// Wait for record
		ch := make(chan bool, 1)
		retries := 10
		retryInterval := 10
		duration := (retries * retryInterval)
		log.Infof("waiting on %s domain validation record creation for %v seconds...", route53RecordName, duration)
		go func() {
			for i := 1; i < retries; i++ {
				ips, err := net.LookupTXT(route53RecordName)
				if err != nil {
					ips, err = utils.BackupResolver.LookupTXT(context.Background(), route53RecordName)
				}
				if err != nil {
					log.Warnf("attempt %v of %v resolving %s, retrying in %vs", i, retries, route53RecordName, retryInterval)
					time.Sleep(time.Duration(int32(retryInterval)) * time.Second)
				} else {
					for _, ip := range ips {
						// todo check ip against route53RecordValue in some capacity so we can pivot the value for testing
						log.Infof("%s. in TXT record value: %s", route53RecordName, ip)
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
