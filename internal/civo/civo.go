/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package civo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/civo/civogo"
	"github.com/konstructio/kubefirst-api/internal/dns"
	"github.com/konstructio/kubefirst-api/internal/httpCommon"
	"github.com/rs/zerolog/log"
)

// TestDomainLiveness checks Civo DNS for the liveness test record
func (c *Configuration) TestDomainLiveness(domainName, domainID string) bool {
	civoRecordName := fmt.Sprintf("kubefirst-liveness.%s", domainName)
	civoRecordValue := "domain record propagated"

	civoRecordConfig := &civogo.DNSRecordConfig{
		Type:     civogo.DNSRecordTypeTXT,
		Name:     civoRecordName,
		Value:    civoRecordValue,
		Priority: 100,
		TTL:      600,
	}

	log.Info().Msgf("checking to see if record %s exists", domainName)
	log.Info().Msgf("domainId %s", domainID)
	log.Info().Msgf("domainName %s", domainName)

	// check for existing records
	records, err := c.Client.ListDNSRecords(domainID)
	if err != nil {
		log.Warn().Msgf("%s", err)
		return false
	}
	if len(records) > 0 {
		log.Info().Msg("domain record found")
		return true
	}

	// create record if it does not exist
	_, err = c.Client.CreateDNSRecord(domainID, civoRecordConfig)
	if err != nil {
		log.Warn().Msgf("%s", err)
		return false
	}
	log.Info().Msg("domain record created")

	count := 0
	// todo need to exit after n number of minutes and tell them to check ns records
	// todo this logic sucks
	for count <= 100 {
		count++

		log.Info().Msgf("%s", civoRecordName)
		ips, err := net.LookupTXT(civoRecordName)
		if err != nil {
			log.Warn().Msgf("Error lookuping up txt record %s, error: %s", civoRecordName, err)
			ips, err = dns.BackupResolver.LookupTXT(context.Background(), civoRecordName)
		}
		if err != nil {
			log.Warn().Msgf("Could not get record name %s - waiting 10 seconds and trying again", civoRecordName)
			time.Sleep(10 * time.Second)
		} else {
			for _, ip := range ips {
				log.Info().Msgf("%s. in TXT record value: %s", civoRecordName, ip)
				count = 101
			}
		}
		if count == 100 {
			log.Error().Msg("unable to resolve domain dns record. please check your domain registrar")
			return false
		}
	}
	return true
}

// GetDomainApexContent determines whether or not a target domain features
// a host responding at zone apex
func GetDomainApexContent(domainName string) bool {
	client := httpCommon.CustomHTTPClient(false, 5*time.Second)
	exists := false
	for _, proto := range []string{"http", "https"} {
		fqdn := fmt.Sprintf("%s://%s", proto, domainName)
		resp, err := client.Get(fqdn)
		if err != nil {
			log.Warn().Msgf("domain %s has no apex content", fqdn)
			continue
		}
		defer resp.Body.Close()

		log.Info().Msgf("domain %s has apex content", fqdn)
		exists = true
	}

	return exists
}

// GetDNSInfo try to reach the provided domain
func (c *Configuration) GetDNSInfo(domainName string) (string, error) {
	log.Info().Msg("GetDNSInfo (working...)")

	civoDNSDomain, err := c.Client.FindDNSDomain(domainName)
	if err != nil {
		log.Error().Msg(err.Error())
		return "", fmt.Errorf("error getting Civo DNS domain %q: %w", domainName, err)
	}

	return civoDNSDomain.ID, nil
}

// GetDNSDomains lists all available DNS domains
func (c *Configuration) GetDNSDomains() ([]string, error) {
	domains, err := c.Client.ListDNSDomains()
	if err != nil {
		return nil, fmt.Errorf("error listing DNS domains: %w", err)
	}

	domainList := make([]string, 0, len(domains))
	for _, domain := range domains {
		domainList = append(domainList, domain.Name)
	}

	return domainList, nil
}

// GetRegions lists all available regions
func (c *Configuration) GetRegions() ([]string, error) {
	regions, err := c.Client.ListRegions()
	if err != nil {
		return nil, fmt.Errorf("error fetching regions: %w", err)
	}

	regionsList := make([]string, 0, len(regions))
	for _, region := range regions {
		regionsList = append(regionsList, region.Code)
	}

	return regionsList, nil
}

func (c *Configuration) ListInstanceSizes() ([]string, error) {
	resp, err := c.Client.SendGetRequest("/v2/sizes")
	if err != nil {
		return nil, fmt.Errorf("error sending request to list instance sizes: %w", err)
	}

	sizes := make([]civogo.InstanceSize, 0)
	if err := json.NewDecoder(bytes.NewReader(resp)).Decode(&sizes); err != nil {
		return nil, fmt.Errorf("error decoding instance sizes response: %w", err)
	}

	var instanceNames []string
	for _, size := range sizes {
		if size.Type == "Kubernetes" && strings.Contains(size.Name, "kube") {
			instanceNames = append(instanceNames, size.Name)
		}
	}

	return instanceNames, nil
}

func (c *Configuration) GetKubeconfig(clusterName string) (string, error) {
	cluster, err := c.Client.FindKubernetesCluster(clusterName)
	if err != nil {
		return "", fmt.Errorf("error finding Kubernetes cluster %q: %w", clusterName, err)
	}

	return cluster.KubeConfig, nil
}
