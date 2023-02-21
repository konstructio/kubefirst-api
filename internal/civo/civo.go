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
package civo

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/civo/civogo"
	"github.com/kubefirst/kubefirst-api/internal/utils"
	log "github.com/sirupsen/logrus"
)

const (
	validationRecordSubdomain string = "kubefirst-liveness"
	validationRecordValue     string = "domain record propagated"
)

// NewCivo instantiates a new Civo configuration
func NewCivo() civogo.Client {
	region := os.Getenv("CIVO_REGION")

	civoClient, err := civogo.NewClient(os.Getenv("CIVO_TOKEN"), region)
	if err != nil {
		log.Fatal(err.Error())
	}
	return *civoClient
}

// Create a single configuration instance to act as an interface to the Civo client
var Conf CivoConfiguration = CivoConfiguration{
	Config: NewCivo(),
	Region: os.Getenv("CIVO_REGION"),
}

// Route53ListTXTRecords retrieves all DNS TXT record type for a hosted zone
func (conf *CivoConfiguration) DNSListTXTRecords(domainID string) ([]CivoTXTRecord, error) {
	records, err := conf.Config.ListDNSRecords(domainID)
	if err != nil {
		return []CivoTXTRecord{}, err
	}
	var txtRecords []CivoTXTRecord
	for _, r := range records {
		log.Debugf("Record Name: %s", r.Name)
		if r.Type == civogo.DNSRecordTypeTXT {
			record := CivoTXTRecord{
				Name:  r.Name,
				Value: r.Value,
				TTL:   r.TTL,
			}
			txtRecords = append(txtRecords, record)
			continue
		}
	}
	return txtRecords, nil
}

// TestHostedZoneLiveness determines whether or not a target hosted zone is initialized and
// ready to accept records
func (conf *CivoConfiguration) TestHostedZoneLiveness(domainName string) (bool, error) {
	civoRecordName := fmt.Sprintf("%s.%s", validationRecordSubdomain, domainName)
	civoDNSDomain, err := conf.Config.FindDNSDomain(domainName)
	if err != nil {
		return false, err
	}

	// Get all txt records for hosted zone
	records, err := conf.DNSListTXTRecords(civoDNSDomain.ID)
	if err != nil {
		return false, err
	}

	// Construct a []string of record names
	foundRecordNames := make([]string, 0)
	for _, rec := range records {
		foundRecordNames = append(foundRecordNames, rec.Name)
	}

	switch utils.FindStringInSlice(foundRecordNames, civoRecordName) {
	case true:
		log.Infof("record %s exists - zone validated", civoRecordName)
		return true, nil
	case false:
		log.Infof("record %s does not exist, creating...", civoRecordName)
		civoRecordConfig := &civogo.DNSRecordConfig{
			Type:     civogo.DNSRecordTypeTXT,
			Name:     civoRecordName,
			Value:    validationRecordValue,
			Priority: 100,
			TTL:      600,
		}
		record, err := conf.Config.CreateDNSRecord(civoDNSDomain.ID, civoRecordConfig)
		if err != nil {
			log.Warnf("%s", err)
			return false, err
		}
		log.Infof("record created at: %s", record.CreatedAt)

		// Wait for record
		ch := make(chan bool, 1)
		retries := 10
		retryInterval := 10
		duration := (retries * retryInterval)
		log.Infof("waiting on %s domain validation record creation for %v seconds...", civoRecordName, duration)
		go func() {
			for i := 1; i < retries; i++ {
				ips, err := net.LookupTXT(civoRecordName)
				if err != nil {
					ips, err = utils.BackupResolver.LookupTXT(context.Background(), civoRecordName)
				}
				if err != nil {
					log.Warnf("attempt %v of %v resolving %s, retrying in %vs", i, retries, civoRecordName, retryInterval)
					time.Sleep(time.Duration(int32(retryInterval)) * time.Second)
				} else {
					for _, ip := range ips {
						// todo check ip against route53RecordValue in some capacity so we can pivot the value for testing
						log.Infof("%s. in TXT record value: %s", civoRecordName, ip)
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
