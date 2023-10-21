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
	googleext "github.com/kubefirst/kubefirst-api/extensions/google"
	terraformext "github.com/kubefirst/kubefirst-api/extensions/terraform"
	vultrext "github.com/kubefirst/kubefirst-api/extensions/vultr"
	"github.com/kubefirst/kubefirst-api/pkg/segment"
	"github.com/kubefirst/kubefirst-api/pkg/telemetryShim"
	"github.com/kubefirst/runtime/pkg/k8s"
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
		case "civo", "digitalocean", "vultr":
			kcfg = k8s.CreateKubeConfig(false, clctrl.ProviderConfig.Kubeconfig)
		case "google":
			var err error
			kcfg, err = clctrl.GoogleClient.GetContainerClusterAuth(clctrl.ClusterName, []byte(clctrl.GoogleAuth.KeyFile))
			if err != nil {
				return err
			}
		}

		telemetryShim.Transmit(segmentClient, segment.MetricUsersTerraformApplyStarted, "")
		log.Info("applying users terraform")

		tfEnvs := map[string]string{}
		var tfEntrypoint, terraformClient string

		switch clctrl.CloudProvider {
		case "aws":
			tfEnvs = awsext.GetAwsTerraformEnvs(tfEnvs, &cl)
			tfEnvs = awsext.GetUsersTerraformEnvs(kcfg.Clientset, &cl, tfEnvs)
		case "civo":
			tfEnvs = civoext.GetCivoTerraformEnvs(tfEnvs, &cl)
			tfEnvs = civoext.GetUsersTerraformEnvs(kcfg.Clientset, &cl, tfEnvs)
		case "google":
			tfEnvs = googleext.GetGoogleTerraformEnvs(tfEnvs, &cl)
			tfEnvs = googleext.GetUsersTerraformEnvs(kcfg.Clientset, &cl, tfEnvs)
		case "digitalocean":
			tfEnvs = digitaloceanext.GetDigitaloceanTerraformEnvs(tfEnvs, &cl)
			tfEnvs = digitaloceanext.GetUsersTerraformEnvs(kcfg.Clientset, &cl, tfEnvs)
		case "vultr":
			tfEnvs = vultrext.GetVultrTerraformEnvs(tfEnvs, &cl)
			tfEnvs = vultrext.GetUsersTerraformEnvs(kcfg.Clientset, &cl, tfEnvs)
		}
		tfEntrypoint = clctrl.ProviderConfig.GitopsDir + "/terraform/users"
		terraformClient = clctrl.ProviderConfig.TerraformClient
		err = terraformext.InitApplyAutoApprove(terraformClient, tfEntrypoint, tfEnvs)
		if err != nil {
			log.Errorf("error applying users terraform: %s", err)
			telemetryShim.Transmit(segmentClient, segment.MetricUsersTerraformApplyStarted, err.Error())
			return err
		}
		log.Info("executed users terraform successfully")
		telemetryShim.Transmit(segmentClient, segment.MetricUsersTerraformApplyCompleted, "")

		clctrl.VaultAuth.RootToken = tfEnvs["VAULT_TOKEN"]
		err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "vault_auth.root_token", clctrl.VaultAuth.RootToken)
		if err != nil {
			return err
		}

		// Set kbot password in object
		err = clctrl.GetUserPassword("kbot")
		if err != nil {
			log.Infof("error fetching kbot password: %s", err)
		}

		err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "users_terraform_apply_check", true)
		if err != nil {
			return err
		}
	}

	return nil
}
