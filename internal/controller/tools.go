/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	awsinternal "github.com/kubefirst/runtime/pkg/aws"
	"github.com/kubefirst/runtime/pkg/civo"
	"github.com/kubefirst/runtime/pkg/digitalocean"
	"github.com/kubefirst/runtime/pkg/k3d"
	"github.com/kubefirst/runtime/pkg/vultr"
	log "github.com/sirupsen/logrus"
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
		log.Info("installing kubefirst dependencies")

		switch cl.CloudProvider {
		case "aws":
			err := awsinternal.DownloadTools(
				clctrl.ProviderConfig.(*awsinternal.AwsConfig),
				awsinternal.KubectlClientVersion,
				awsinternal.TerraformClientVersion,
			)
			if err != nil {
				return err
			}
		case "civo":
			err := civo.DownloadTools(
				clctrl.ProviderConfig.(*civo.CivoConfig).KubectlClient,
				civo.KubectlClientVersion,
				civo.LocalhostOS,
				civo.LocalhostArch,
				civo.TerraformClientVersion,
				toolsDir,
			)
			if err != nil {
				return err
			}
		case "digitalocean":
			err := digitalocean.DownloadTools(
				clctrl.ProviderConfig.(*digitalocean.DigitaloceanConfig).KubectlClient,
				digitalocean.KubectlClientVersion,
				digitalocean.LocalhostOS,
				digitalocean.LocalhostArch,
				digitalocean.TerraformClientVersion,
				toolsDir,
			)
			if err != nil {
				return err
			}
		case "k3d":
			err := k3d.DownloadTools(cl.ClusterName, cl.GitProvider, cl.GitOwner, toolsDir)
			if err != nil {
				return err
			}
		case "vultr":
			err := vultr.DownloadTools(
				clctrl.ProviderConfig.(*vultr.VultrConfig).KubectlClient,
				vultr.KubectlClientVersion,
				vultr.LocalhostOS,
				vultr.LocalhostArch,
				vultr.TerraformClientVersion,
				toolsDir,
			)
			if err != nil {
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
