/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	"context"
	"fmt"

	cloudflare_api "github.com/cloudflare/cloudflare-go"
	"github.com/kubefirst/metrics-client/pkg/telemetry"
	"github.com/kubefirst/runtime/pkg/civo"
	"github.com/kubefirst/runtime/pkg/digitalocean"
	"github.com/kubefirst/runtime/pkg/dns"
	"github.com/kubefirst/runtime/pkg/vultr"
	log "github.com/sirupsen/logrus"
)

// DomainLivenessTest
func (clctrl *ClusterController) DomainLivenessTest() error {
	cl, err := clctrl.MdbCl.GetCluster(clctrl.ClusterName)
	if err != nil {
		return err
	}

	if !cl.DomainLivenessCheck {
		telemetry.SendCountMetric(clctrl.SegmentClient, telemetry.DomainLivenessStarted, err.Error())

		switch clctrl.DnsProvider {
		case "aws":
			domainLiveness := clctrl.AwsClient.TestHostedZoneLiveness(clctrl.DomainName)

			err = clctrl.HandleDomainLiveness(domainLiveness)
			if err != nil {
				return err
			}
		case "civo":
			civoConf := civo.CivoConfiguration{
				Client:  civo.NewCivo(cl.CivoAuth.Token, cl.CloudRegion),
				Context: context.Background(),
			}

			// domain id
			domainId, err := civoConf.GetDNSInfo(clctrl.DomainName, clctrl.CloudRegion)
			if err != nil {
				telemetry.SendCountMetric(clctrl.SegmentClient, telemetry.DomainLivenessFailed, err.Error())
				log.Info(err.Error())
			}

			log.Infof("domainId: %s", domainId)
			domainLiveness := civoConf.TestDomainLiveness(clctrl.DomainName, domainId, clctrl.CloudRegion)

			err = clctrl.HandleDomainLiveness(domainLiveness)
			if err != nil {
				return err
			}
		case "cloudflare":

			_, err := cloudflare_api.NewWithAPIToken(clctrl.CloudflareAuth.APIToken)
			if err != nil {
				return err
			}

			// cloudflareConf := cloudflare.CloudflareConfiguration{
			// 	Client:  client,
			// 	Context: context.Background(),
			// }

			// domainLiveness := cloudflareConf.TestDomainLiveness(clctrl.DomainName)

			// err = clctrl.HandleDomainLiveness(domainLiveness)
			// if err != nil {
			// 	return err
			// }
		case "digitalocean":
			digitaloceanConf := digitalocean.DigitaloceanConfiguration{
				Client:  digitalocean.NewDigitalocean(cl.DigitaloceanAuth.Token),
				Context: context.Background(),
			}

			// domain id
			domainId, err := digitaloceanConf.GetDNSInfo(clctrl.DomainName)
			if err != nil {
				log.Info(err.Error())
			}

			log.Infof("domainId: %s", domainId)
			domainLiveness := digitaloceanConf.TestDomainLiveness(clctrl.DomainName)

			err = clctrl.HandleDomainLiveness(domainLiveness)
			if err != nil {
				return err
			}
		case "vultr":
			vultrConf := vultr.VultrConfiguration{
				Client:  vultr.NewVultr(cl.VultrAuth.Token),
				Context: context.Background(),
			}

			// domain id
			domainId, err := vultrConf.GetDNSInfo(clctrl.DomainName)
			if err != nil {
				log.Info(err.Error())
			}

			// viper values set in above function
			log.Infof("domainId: %s", domainId)
			domainLiveness := vultrConf.TestDomainLiveness(clctrl.DomainName)

			err = clctrl.HandleDomainLiveness(domainLiveness)
			if err != nil {
				return err
			}
		}

		err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "domain_liveness_check", true)
		if err != nil {
			return err
		}

		telemetry.SendCountMetric(clctrl.SegmentClient, telemetry.DomainLivenessCompleted, err.Error())

		log.Infof("domain %s verified", clctrl.DomainName)
	}

	return nil
}

// HandleDomainLiveness
func (clctrl *ClusterController) HandleDomainLiveness(domainLiveness bool) error {
	if !domainLiveness {
		foundRecords, err := dns.GetDomainNSRecords(clctrl.DomainName)
		if err != nil {
			log.Warnf("error attempting to get NS records for domain %s: %s", clctrl.DomainName, err)
		}
		msg := fmt.Sprintf("failed to verify domain liveness for domain %s", clctrl.DomainName)
		if len(foundRecords) != 0 {
			msg = msg + fmt.Sprintf(" - last result: %s - it may be necessary to wait for propagation", foundRecords)
		}
		return fmt.Errorf(msg)
	} else {
		return nil
	}
}
