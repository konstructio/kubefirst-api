/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	"context"
	"errors"
	"fmt"

	cloudflare_api "github.com/cloudflare/cloudflare-go"
	"github.com/konstructio/kubefirst-api/internal/civo"
	"github.com/konstructio/kubefirst-api/internal/cloudflare"
	"github.com/konstructio/kubefirst-api/internal/digitalocean"
	"github.com/konstructio/kubefirst-api/internal/dns"
	"github.com/konstructio/kubefirst-api/internal/secrets"
	"github.com/konstructio/kubefirst-api/internal/vultr"
	"github.com/kubefirst/metrics-client/pkg/telemetry"
	log "github.com/rs/zerolog/log"
)

// DomainLivenessTest
func (clctrl *ClusterController) DomainLivenessTest() error {
	cl, err := secrets.GetCluster(clctrl.KubernetesClient, clctrl.ClusterName)
	if err != nil {
		return fmt.Errorf("failed to get cluster for domain liveness test: %w", err)
	}

	if !cl.DomainLivenessCheck {
		telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.DomainLivenessStarted, "")

		switch clctrl.DNSProvider {
		case "aws":
			domainLiveness := clctrl.AwsClient.TestHostedZoneLiveness(clctrl.DomainName)

			err = clctrl.HandleDomainLiveness(domainLiveness)
			if err != nil {
				return fmt.Errorf("domain liveness check failed for AWS: %w", err)
			}
		case "azure":
			domainLiveness, err := clctrl.AzureClient.TestHostedZoneLiveness(context.Background(), clctrl.DomainName, clctrl.AzureDNSZoneResourceGroup)
			if err != nil {
				return fmt.Errorf("domain liveness command failed for Azure: %w", err)
			}

			err = clctrl.HandleDomainLiveness(domainLiveness)
			if err != nil {
				return fmt.Errorf("domain liveness check failed for Azure: %w", err)
			}
		case "civo":
			civoConf := civo.Configuration{
				Client:  civo.NewCivo(cl.CivoAuth.Token, cl.CloudRegion),
				Context: context.Background(),
			}

			// domain id
			domainID, err := civoConf.GetDNSInfo(clctrl.DomainName)
			if err != nil {
				telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.DomainLivenessFailed, err.Error())
				log.Info().Msg(err.Error())
			}

			log.Info().Msgf("domainId: %s", domainID)
			domainLiveness := civoConf.TestDomainLiveness(clctrl.DomainName, domainID)

			err = clctrl.HandleDomainLiveness(domainLiveness)
			if err != nil {
				return fmt.Errorf("domain liveness check failed for Civo: %w", err)
			}
		case "cloudflare":

			client, err := cloudflare_api.NewWithAPIToken(clctrl.CloudflareAuth.APIToken)
			if err != nil {
				return fmt.Errorf("failed to create Cloudflare client: %w", err)
			}

			cloudflareConf := cloudflare.Configuration{
				Client:  client,
				Context: context.Background(),
			}

			domainLiveness := cloudflareConf.TestDomainLiveness(clctrl.DomainName)

			err = clctrl.HandleDomainLiveness(domainLiveness)
			if err != nil {
				return fmt.Errorf("domain liveness check failed for Cloudflare: %w", err)
			}
		case "digitalocean":
			digitaloceanConf := digitalocean.Configuration{
				Client:  digitalocean.NewDigitalocean(cl.DigitaloceanAuth.Token),
				Context: context.Background(),
			}

			// domain id
			domainID, err := digitaloceanConf.GetDNSInfo(clctrl.DomainName)
			if err != nil {
				log.Info().Msg(err.Error())
			}

			log.Info().Msgf("domainId: %s", domainID)
			domainLiveness := digitaloceanConf.TestDomainLiveness(clctrl.DomainName)

			err = clctrl.HandleDomainLiveness(domainLiveness)
			if err != nil {
				return fmt.Errorf("domain liveness check failed for DigitalOcean: %w", err)
			}
		case "vultr":
			vultrConf := vultr.Configuration{
				Client:  vultr.NewVultr(cl.VultrAuth.Token),
				Context: context.Background(),
			}

			// domain id
			domainID, err := vultrConf.GetDNSInfo(clctrl.DomainName)
			if err != nil {
				log.Info().Msg(err.Error())
			}

			// viper values set in above function
			log.Info().Msgf("domainId: %s", domainID)
			domainLiveness := vultrConf.TestDomainLiveness(clctrl.DomainName)

			err = clctrl.HandleDomainLiveness(domainLiveness)
			if err != nil {
				return fmt.Errorf("domain liveness check failed for Vultr: %w", err)
			}
		}

		clctrl.Cluster.DomainLivenessCheck = true
		err = secrets.UpdateCluster(clctrl.KubernetesClient, clctrl.Cluster)
		if err != nil {
			return fmt.Errorf("failed to update cluster after domain liveness test: %w", err)
		}

		telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.DomainLivenessCompleted, "")

		log.Info().Msgf("domain %s verified", clctrl.DomainName)
	}

	return nil
}

// HandleDomainLiveness
func (clctrl *ClusterController) HandleDomainLiveness(domainLiveness bool) error {
	if !domainLiveness {
		foundRecords, err := dns.GetDomainNSRecords(clctrl.DomainName)
		if err != nil {
			log.Warn().Msgf("error attempting to get NS records for domain %s: %s", clctrl.DomainName, err)
		}
		msg := fmt.Sprintf("failed to verify domain liveness for domain %s", clctrl.DomainName)
		if len(foundRecords) != 0 {
			msg += fmt.Sprintf(" - last result: %s - it may be necessary to wait for propagation", foundRecords)
		}
		return errors.New(msg)
	}

	return nil
}
