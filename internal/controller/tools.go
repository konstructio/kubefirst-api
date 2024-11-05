/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	"fmt"

	"github.com/konstructio/kubefirst-api/internal/secrets"
	"github.com/konstructio/kubefirst-api/internal/utils"
	awsinternal "github.com/konstructio/kubefirst-api/pkg/aws"
	"github.com/konstructio/kubefirst-api/pkg/providerConfigs"
	log "github.com/rs/zerolog/log"
)

// DownloadTools
// This obviously doesn't work in an api-based environment.
// It's included for testing and development.
func (clctrl *ClusterController) DownloadTools(toolsDir string) error {
	cl, err := secrets.GetCluster(clctrl.KubernetesClient, clctrl.ClusterName)
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	if !cl.InstallToolsCheck {
		log.Info().Msg("installing kubefirst dependencies")

		switch cl.CloudProvider {
		case "akamai":
			err := utils.DownloadTools(
				clctrl.ProviderConfig.KubectlClient,
				providerConfigs.KubectlClientVersion,
				providerConfigs.LocalhostOS,
				providerConfigs.LocalhostArch,
				providerConfigs.TerraformClientVersion,
				toolsDir,
			)
			if err != nil {
				log.Error().Msgf("error downloading dependencies: %s", err)
				return fmt.Errorf("failed to download tools for akamai: %w", err)
			}
		case "aws":
			err := awsinternal.DownloadTools(
				&clctrl.ProviderConfig,
				providerConfigs.KubectlClientVersion,
				providerConfigs.TerraformClientVersion,
			)
			if err != nil {
				log.Error().Msgf("error downloading dependencies: %s", err)
				return fmt.Errorf("failed to download tools for aws: %w", err)
			}
		case "azure":
			err := utils.DownloadTools(
				clctrl.ProviderConfig.KubectlClient,
				providerConfigs.KubectlClientVersion,
				providerConfigs.LocalhostOS,
				providerConfigs.LocalhostArch,
				providerConfigs.TerraformClientVersion,
				toolsDir,
			)
			if err != nil {
				log.Error().Msgf("error downloading dependencies: %s", err)
				return fmt.Errorf("failed to download tools for azure: %w", err)
			}
		case "civo":
			err := utils.DownloadTools(
				clctrl.ProviderConfig.KubectlClient,
				providerConfigs.KubectlClientVersion,
				providerConfigs.LocalhostOS,
				providerConfigs.LocalhostArch,
				providerConfigs.TerraformClientVersion,
				toolsDir,
			)
			if err != nil {
				log.Error().Msgf("error downloading dependencies: %s", err)
				return fmt.Errorf("failed to download tools for civo: %w", err)
			}
		case "google":
			err := utils.DownloadTools(
				clctrl.ProviderConfig.KubectlClient,
				providerConfigs.KubectlClientVersion,
				providerConfigs.LocalhostOS,
				providerConfigs.LocalhostArch,
				providerConfigs.TerraformClientVersion,
				toolsDir,
			)
			if err != nil {
				log.Error().Msgf("error downloading dependencies: %s", err)
				return fmt.Errorf("failed to download tools for google: %w", err)
			}
		case "digitalocean":
			err := utils.DownloadTools(
				clctrl.ProviderConfig.KubectlClient,
				providerConfigs.KubectlClientVersion,
				providerConfigs.LocalhostOS,
				providerConfigs.LocalhostArch,
				providerConfigs.TerraformClientVersion,
				toolsDir,
			)
			if err != nil {
				log.Error().Msgf("error downloading dependencies: %s", err)
				return fmt.Errorf("failed to download tools for digitalocean: %w", err)
			}
		case "vultr":
			err := utils.DownloadTools(
				clctrl.ProviderConfig.KubectlClient,
				providerConfigs.KubectlClientVersion,
				providerConfigs.LocalhostOS,
				providerConfigs.LocalhostArch,
				providerConfigs.TerraformClientVersion,
				toolsDir,
			)
			if err != nil {
				log.Error().Msgf("error downloading dependencies: %s", err)
				return fmt.Errorf("failed to download tools for vultr: %w", err)
			}

			// TODO: move to runtime
			// use vultr DownloadTools meanwhile
		case "k3s":
			err := utils.DownloadTools(
				clctrl.ProviderConfig.KubectlClient,
				providerConfigs.KubectlClientVersion,
				providerConfigs.LocalhostOS,
				providerConfigs.LocalhostArch,
				providerConfigs.TerraformClientVersion,
				toolsDir,
			)
			if err != nil {
				log.Error().Msgf("error downloading dependencies: %s", err)
				return fmt.Errorf("failed to download tools for k3s: %w", err)
			}
		}
		log.Info().Msg("dependency downloads complete")

		clctrl.Cluster.InstallToolsCheck = true
		err = secrets.UpdateCluster(clctrl.KubernetesClient, clctrl.Cluster)
		if err != nil {
			return fmt.Errorf("failed to update cluster after downloading tools: %w", err)
		}
	}

	return nil
}
