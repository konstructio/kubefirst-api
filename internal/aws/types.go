/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package aws

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	route53Types "github.com/aws/aws-sdk-go-v2/service/route53/types"
)

// AWSConfiguration stores session data to organize all AWS functions into a single struct
type AWSConfiguration struct {
	Config aws.Config
}

type AWSRoute53AlterResourceRecord struct {
	hostedZoneName    string
	hostedZoneID      string
	route53RecordName string
	input             *route53.ChangeResourceRecordSetsInput
}

// AWSARecord stores Route53 A record data
type AWSARecord struct {
	Name        string
	RecordType  string
	TTL         *int64
	AliasTarget *route53Types.AliasTarget
}

// AWSTXTRecord stores Route53 TXT record data
type AWSTXTRecord struct {
	Name          string
	Value         string
	SetIdentifier *string
	Weight        *int64
	TTL           int64
}
