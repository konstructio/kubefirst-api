/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	"context"
	"fmt"

	"github.com/kubefirst/kubefirst-api/internal/telemetryShim"
	"github.com/kubefirst/runtime/pkg/civo"
	"github.com/kubefirst/runtime/pkg/digitalocean"
	"github.com/kubefirst/runtime/pkg/segment"
	"github.com/kubefirst/runtime/pkg/vultr"
	log "github.com/sirupsen/logrus"
)

// DomainLivenessTest
func (clctrl *ClusterController) DomainLivenessTest() error {
	cl, err := clctrl.MdbCl.GetCluster(clctrl.ClusterName)
	if err != nil {
		return err
	}

	// Telemetry handler
	segmentClient, err := telemetryShim.SetupTelemetry(cl)
	if err != nil {
		return err
	}
	defer segmentClient.Client.Close()

	if !cl.DomainLivenessCheck {
		telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricDomainLivenessStarted, "")

		switch clctrl.CloudProvider {
		case "aws":
			domainLiveness := clctrl.AwsClient.TestHostedZoneLiveness(clctrl.DomainName)
			if !domainLiveness {
				return fmt.Errorf("failed to verify domain liveness for domain %s", clctrl.DomainName)
			}
		case "civo":
			// domain id
			domainId, err := civo.GetDNSInfo(clctrl.DomainName, clctrl.CloudRegion)
			if err != nil {
				telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricDomainLivenessFailed, "domain liveness test failed")
				log.Info(err.Error())
			}

			log.Infof("domainId: %s", domainId)
			domainLiveness := civo.TestDomainLiveness(clctrl.DomainName, domainId, clctrl.CloudRegion)
			if !domainLiveness {
				return fmt.Errorf("failed to verify domain liveness for domain %s", clctrl.DomainName)
			}
		case "digitalocean":
			digitaloceanConf := digitalocean.DigitaloceanConfiguration{
				Client:  digitalocean.NewDigitalocean(),
				Context: context.Background(),
			}

			// domain id
			domainId, err := digitaloceanConf.GetDNSInfo(clctrl.DomainName)
			if err != nil {
				log.Info(err.Error())
			}

			log.Infof("domainId: %s", domainId)
			domainLiveness := digitaloceanConf.TestDomainLiveness(clctrl.DomainName)
			if !domainLiveness {
				return fmt.Errorf("failed to verify domain liveness for domain %s", clctrl.DomainName)
			}
		case "vultr":
			vultrConf := vultr.VultrConfiguration{
				Client:  vultr.NewVultr(),
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
			if !domainLiveness {
				return fmt.Errorf("failed to verify domain liveness for domain %s", clctrl.DomainName)
			}
		}

		err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "domain_liveness_check", true)
		if err != nil {
			return err
		}

		telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricDomainLivenessCompleted, "")

		log.Infof("domain %s verified", clctrl.DomainName)
	}

	return nil
}
