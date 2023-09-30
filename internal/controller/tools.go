/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	"os"

	awsinternal "github.com/kubefirst/kubefirst-api/pkg/aws"
	google "github.com/kubefirst/kubefirst-api/pkg/google"
	"github.com/kubefirst/kubefirst-api/pkg/providerConfigs"
	"github.com/kubefirst/runtime/pkg/civo"
	"github.com/kubefirst/runtime/pkg/digitalocean"
	"github.com/kubefirst/runtime/pkg/vultr"
	log "github.com/sirupsen/logrus"
)

// DownloadTools
// This obviously doesn't work in an api-based environment.
// It's included for testing and development.
func (clctrl *ClusterController) DownloadTools(toolsDir string) error {
	// Logging handler
	// Logs to stdout to maintain compatibility with event streaming
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "",
	})
	log.SetReportCaller(false)
	log.SetOutput(os.Stdout)

	cl, err := clctrl.MdbCl.GetCluster(clctrl.ClusterName)
	if err != nil {
		return err
	}

	if !cl.InstallToolsCheck {
		log.Info("installing kubefirst dependencies")

		switch cl.CloudProvider {
		case "aws":
			err := awsinternal.DownloadTools(
				&clctrl.ProviderConfig,
				providerConfigs.KubectlClientVersion,
				providerConfigs.TerraformClientVersion,
			)
			if err != nil {
				log.Errorf("error downloading dependencies: %s", err)
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
				log.Errorf("error downloading dependencies: %s", err)
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
				log.Errorf("error downloading dependencies: %s", err)
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
				log.Errorf("error downloading dependencies: %s", err)
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
				log.Errorf("error downloading dependencies: %s", err)
				return err
			}
		}
		log.Info("dependency downloads complete")

		err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "install_tools_check", true)
		if err != nil {
			return err
		}
	}

	return nil
}
