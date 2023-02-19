/*
Copyright © 2023 Kubefirst <kubefirst.io>
Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:
The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.
THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
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
