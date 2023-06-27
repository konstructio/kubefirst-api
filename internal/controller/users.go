/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	"os"

	awsext "github.com/kubefirst/kubefirst-api/extensions/aws"
	civoext "github.com/kubefirst/kubefirst-api/extensions/civo"
	digitaloceanext "github.com/kubefirst/kubefirst-api/extensions/digitalocean"
	terraformext "github.com/kubefirst/kubefirst-api/extensions/terraform"
	vultrext "github.com/kubefirst/kubefirst-api/extensions/vultr"
	"github.com/kubefirst/kubefirst-api/internal/telemetryShim"
	awsinternal "github.com/kubefirst/runtime/pkg/aws"
	"github.com/kubefirst/runtime/pkg/civo"
	"github.com/kubefirst/runtime/pkg/digitalocean"
	"github.com/kubefirst/runtime/pkg/k8s"
	"github.com/kubefirst/runtime/pkg/segment"
	"github.com/kubefirst/runtime/pkg/vultr"
	log "github.com/sirupsen/logrus"
)

// RunUsersTerraform
func (clctrl *ClusterController) RunUsersTerraform() error {
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

	// Telemetry handler
	segmentClient, err := telemetryShim.SetupTelemetry(cl)
	if err != nil {
		return err
	}
	defer segmentClient.Client.Close()

	if !cl.UsersTerraformApplyCheck {
		var kcfg *k8s.KubernetesClient

		switch clctrl.CloudProvider {
		case "aws":
			kcfg = awsext.CreateEKSKubeconfig(&clctrl.AwsClient.Config, clctrl.ClusterName)
		case "civo":
			kcfg = k8s.CreateKubeConfig(false, clctrl.ProviderConfig.(*civo.CivoConfig).Kubeconfig)
		case "digitalocean":
			kcfg = k8s.CreateKubeConfig(false, clctrl.ProviderConfig.(*digitalocean.DigitaloceanConfig).Kubeconfig)
		case "vultr":
			kcfg = k8s.CreateKubeConfig(false, clctrl.ProviderConfig.(*vultr.VultrConfig).Kubeconfig)
		}

		telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricUsersTerraformApplyStarted, "")
		log.Info("applying users terraform")

		tfEnvs := map[string]string{}
		var tfEntrypoint, terraformClient string

		switch clctrl.CloudProvider {
		case "aws":
			tfEnvs = awsext.GetAwsTerraformEnvs(tfEnvs, &cl)
			tfEnvs = awsext.GetUsersTerraformEnvs(kcfg.Clientset, &cl, tfEnvs)
			tfEntrypoint = clctrl.ProviderConfig.(*awsinternal.AwsConfig).GitopsDir + "/terraform/users"
			terraformClient = clctrl.ProviderConfig.(*awsinternal.AwsConfig).TerraformClient
		case "civo":
			tfEnvs = civoext.GetCivoTerraformEnvs(tfEnvs, &cl)
			tfEnvs = civoext.GetUsersTerraformEnvs(kcfg.Clientset, &cl, tfEnvs)
			tfEntrypoint = clctrl.ProviderConfig.(*civo.CivoConfig).GitopsDir + "/terraform/users"
			terraformClient = clctrl.ProviderConfig.(*civo.CivoConfig).TerraformClient
		case "digitalocean":
			tfEnvs = digitaloceanext.GetDigitaloceanTerraformEnvs(tfEnvs, &cl)
			tfEnvs = digitaloceanext.GetUsersTerraformEnvs(kcfg.Clientset, &cl, tfEnvs)
			tfEntrypoint = clctrl.ProviderConfig.(*digitalocean.DigitaloceanConfig).GitopsDir + "/terraform/users"
			terraformClient = clctrl.ProviderConfig.(*digitalocean.DigitaloceanConfig).TerraformClient
		case "vultr":
			tfEnvs = vultrext.GetVultrTerraformEnvs(tfEnvs, &cl)
			tfEnvs = vultrext.GetUsersTerraformEnvs(kcfg.Clientset, &cl, tfEnvs)
			tfEntrypoint = clctrl.ProviderConfig.(*vultr.VultrConfig).GitopsDir + "/terraform/users"
			terraformClient = clctrl.ProviderConfig.(*vultr.VultrConfig).TerraformClient
		}

		err = terraformext.InitApplyAutoApprove(terraformClient, tfEntrypoint, tfEnvs)
		if err != nil {
			log.Errorf("error applying users terraform: %s", err)
			telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricUsersTerraformApplyStarted, err.Error())
			return err
		}
		log.Info("executed users terraform successfully")
		telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricUsersTerraformApplyCompleted, "")

		err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "users_terraform_apply_check", true)
		if err != nil {
			return err
		}
	}

	return nil
}
