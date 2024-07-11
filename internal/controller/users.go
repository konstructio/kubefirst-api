/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	"time"

	akamaiext "github.com/kubefirst/kubefirst-api/extensions/akamai"
	awsext "github.com/kubefirst/kubefirst-api/extensions/aws"
	civoext "github.com/kubefirst/kubefirst-api/extensions/civo"
	digitaloceanext "github.com/kubefirst/kubefirst-api/extensions/digitalocean"
	googleext "github.com/kubefirst/kubefirst-api/extensions/google"
	k3sext "github.com/kubefirst/kubefirst-api/extensions/k3s"
	terraformext "github.com/kubefirst/kubefirst-api/extensions/terraform"
	vultrext "github.com/kubefirst/kubefirst-api/extensions/vultr"
	"github.com/kubefirst/kubefirst-api/internal/k8s"
	"github.com/kubefirst/kubefirst-api/internal/secrets"
	"github.com/kubefirst/metrics-client/pkg/telemetry"
	log "github.com/rs/zerolog/log"
)

// RunUsersTerraform
func (clctrl *ClusterController) RunUsersTerraform() error {
	cl, err := secrets.GetCluster(clctrl.KubernetesClient, clctrl.ClusterName)
	if err != nil {
		return err
	}

	if !cl.UsersTerraformApplyCheck {
		var kcfg *k8s.KubernetesClient

		switch clctrl.CloudProvider {
		case "aws":
			kcfg = awsext.CreateEKSKubeconfig(&clctrl.AwsClient.Config, clctrl.ClusterName)
		case "akamai", "civo", "digitalocean", "k3s", "vultr":
			kcfg = k8s.CreateKubeConfig(false, clctrl.ProviderConfig.Kubeconfig)
		case "google":
			var err error
			kcfg, err = clctrl.GoogleClient.GetContainerClusterAuth(clctrl.ClusterName, []byte(clctrl.GoogleAuth.KeyFile))
			if err != nil {
				return err
			}
		}

		telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.UsersTerraformApplyStarted, "")
		log.Info().Msg("applying users terraform")

		tfEnvs := map[string]string{}
		var tfEntrypoint, terraformClient string

		switch clctrl.CloudProvider {
		case "akamai":
			tfEnvs = akamaiext.GetAkamaiTerraformEnvs(tfEnvs, &cl)
			tfEnvs = akamaiext.GetUsersTerraformEnvs(kcfg.Clientset, &cl, tfEnvs)
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
		case "k3s":
			tfEnvs = k3sext.GetK3sTerraformEnvs(tfEnvs, &cl)
			tfEnvs = k3sext.GetUsersTerraformEnvs(kcfg.Clientset, &cl, tfEnvs)
		}
		tfEntrypoint = clctrl.ProviderConfig.GitopsDir + "/terraform/users"
		terraformClient = clctrl.ProviderConfig.TerraformClient
		err = terraformext.InitApplyAutoApprove(terraformClient, tfEntrypoint, tfEnvs)
		if err != nil {
			log.Error().Msgf("error applying users terraform: %s", err)
			log.Info().Msg("sleeping 10 seconds before retrying terraform execution once more")
			time.Sleep(10 * time.Second)
			err = terraformext.InitApplyAutoApprove(terraformClient, tfEntrypoint, tfEnvs)
			if err != nil {
				log.Error().Msgf("error applying users terraform: %s", err)
				telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.UsersTerraformApplyFailed, err.Error())
				return err
			}
		}
		log.Info().Msg("executed users terraform successfully")
		telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.UsersTerraformApplyCompleted, "")

		clctrl.VaultAuth.RootToken = tfEnvs["VAULT_TOKEN"]

		clctrl.Cluster.VaultAuth.RootToken = clctrl.VaultAuth.RootToken
		err = secrets.UpdateCluster(clctrl.KubernetesClient, clctrl.Cluster)
		if err != nil {
			return err
		}

		// Set kbot password in object
		err = clctrl.GetUserPassword("kbot")
		if err != nil {
			log.Info().Msgf("error fetching kbot password: %s", err)
		}

		clctrl.Cluster.UsersTerraformApplyCheck = true
		err = secrets.UpdateCluster(clctrl.KubernetesClient, clctrl.Cluster)
		if err != nil {
			return err
		}
	}

	return nil
}
