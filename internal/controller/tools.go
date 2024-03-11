/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	awsinternal "github.com/kubefirst/kubefirst-api/pkg/aws"
	google "github.com/kubefirst/kubefirst-api/pkg/google"
	"github.com/kubefirst/kubefirst-api/pkg/providerConfigs"
	"github.com/kubefirst/runtime/pkg/civo"
	"github.com/kubefirst/runtime/pkg/digitalocean"
	"github.com/kubefirst/runtime/pkg/vultr"
	log "github.com/rs/zerolog/log"
)

// DownloadTools
// This obviously doesn't work in an api-based environment.
// It's included for testing and development.
func (clctrl *ClusterController) DownloadTools(toolsDir string) error {
	cl, err := clctrl.MdbCl.GetCluster(clctrl.ClusterName)
	if err != nil {
		return err
	}

	if !cl.InstallToolsCheck {
		log.Info().Msg("installing kubefirst dependencies")

		switch cl.CloudProvider {
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
				log.Errorf("error downloading dependencies: %s", err)
				return err
			}
		}
		log.Info().Msg("dependency downloads complete")

		err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "install_tools_check", true)
		if err != nil {
			return err
		}
	}

	return nil
}
