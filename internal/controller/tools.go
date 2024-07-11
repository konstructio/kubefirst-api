/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	"github.com/kubefirst/kubefirst-api/internal/civo"
	"github.com/kubefirst/kubefirst-api/internal/digitalocean"
	"github.com/kubefirst/kubefirst-api/internal/secrets"
	"github.com/kubefirst/kubefirst-api/internal/vultr"
	awsinternal "github.com/kubefirst/kubefirst-api/pkg/aws"
	google "github.com/kubefirst/kubefirst-api/pkg/google"
	"github.com/kubefirst/kubefirst-api/pkg/providerConfigs"
	log "github.com/rs/zerolog/log"
)

// DownloadTools
// This obviously doesn't work in an api-based environment.
// It's included for testing and development.
func (clctrl *ClusterController) DownloadTools(toolsDir string) error {
	cl, err := secrets.GetCluster(clctrl.KubernetesClient, clctrl.ClusterName)
	if err != nil {
		return err
	}

	if !cl.InstallToolsCheck {
		log.Info().Msg("installing kubefirst dependencies")

		switch cl.CloudProvider {
		case "akamai":
			err := civo.DownloadTools(
				clctrl.ProviderConfig.KubectlClient,
				providerConfigs.KubectlClientVersion,
				providerConfigs.LocalhostOS,
				providerConfigs.LocalhostArch,
				providerConfigs.TerraformClientVersion,
				toolsDir,
			)
			if err != nil {
				log.Error().Msgf("error downloading dependencies: %s", err)
				return err
			}
		case "aws":
			err := awsinternal.DownloadTools(
				&clctrl.ProviderConfig,
				providerConfigs.KubectlClientVersion,
				providerConfigs.TerraformClientVersion,
			)
			if err != nil {
				log.Error().Msgf("error downloading dependencies: %s", err)
				return err
			}
		case "civo":
			err := civo.DownloadTools(
				clctrl.ProviderConfig.KubectlClient,
				providerConfigs.KubectlClientVersion,
				providerConfigs.LocalhostOS,
				providerConfigs.LocalhostArch,
				providerConfigs.TerraformClientVersion,
				toolsDir,
			)
			if err != nil {
				log.Error().Msgf("error downloading dependencies: %s", err)
				return err
			}
		case "google":
			err := google.DownloadTools(
				clctrl.ProviderConfig.KubectlClient,
				providerConfigs.KubectlClientVersion,
				providerConfigs.LocalhostOS,
				providerConfigs.LocalhostArch,
				providerConfigs.TerraformClientVersion,
				toolsDir,
			)
			if err != nil {
				log.Error().Msgf("error downloading dependencies: %s", err)
				return err
			}
		case "digitalocean":
			err := digitalocean.DownloadTools(
				clctrl.ProviderConfig.KubectlClient,
				providerConfigs.KubectlClientVersion,
				providerConfigs.LocalhostOS,
				providerConfigs.LocalhostArch,
				providerConfigs.TerraformClientVersion,
				toolsDir,
			)
			if err != nil {
				log.Error().Msgf("error downloading dependencies: %s", err)
				return err
			}
		case "vultr":
			err := vultr.DownloadTools(
				clctrl.ProviderConfig.KubectlClient,
				providerConfigs.KubectlClientVersion,
				providerConfigs.LocalhostOS,
				providerConfigs.LocalhostArch,
				providerConfigs.TerraformClientVersion,
				toolsDir,
			)
			if err != nil {
				log.Error().Msgf("error downloading dependencies: %s", err)
				return err
			}

			// TODO: move to runtime
			// use vultr DownloadTools meanwhile
		case "k3s":
			err := vultr.DownloadTools(
				clctrl.ProviderConfig.KubectlClient,
				providerConfigs.KubectlClientVersion,
				providerConfigs.LocalhostOS,
				providerConfigs.LocalhostArch,
				providerConfigs.TerraformClientVersion,
				toolsDir,
			)
			if err != nil {
				log.Fatal().Msgf("error downloading dependencies: %s", err)
				return err
			}
		}
		log.Info().Msg("dependency downloads complete")

		clctrl.Cluster.InstallToolsCheck = true
		err = secrets.UpdateCluster(clctrl.KubernetesClient, clctrl.Cluster)
		if err != nil {
			return err
		}
	}

	return nil
}
