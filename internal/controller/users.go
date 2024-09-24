/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	"fmt"
	"time"

	akamaiext "github.com/konstructio/kubefirst-api/extensions/akamai"
	awsext "github.com/konstructio/kubefirst-api/extensions/aws"
	azureext "github.com/konstructio/kubefirst-api/extensions/azure"
	civoext "github.com/konstructio/kubefirst-api/extensions/civo"
	digitaloceanext "github.com/konstructio/kubefirst-api/extensions/digitalocean"
	googleext "github.com/konstructio/kubefirst-api/extensions/google"
	k3sext "github.com/konstructio/kubefirst-api/extensions/k3s"
	terraformext "github.com/konstructio/kubefirst-api/extensions/terraform"
	vultrext "github.com/konstructio/kubefirst-api/extensions/vultr"
	"github.com/konstructio/kubefirst-api/internal/k8s"
	"github.com/konstructio/kubefirst-api/internal/secrets"
	"github.com/kubefirst/metrics-client/pkg/telemetry"
	log "github.com/rs/zerolog/log"
)

// RunUsersTerraform
func (clctrl *ClusterController) RunUsersTerraform() error {
	cl, err := secrets.GetCluster(clctrl.KubernetesClient, clctrl.ClusterName)
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	if !cl.UsersTerraformApplyCheck {
		var kcfg *k8s.KubernetesClient

		switch clctrl.CloudProvider {
		case "aws":
			kcfg = awsext.CreateEKSKubeconfig(&clctrl.AwsClient.Config, clctrl.ClusterName)
		case "akamai", "azure", "civo", "digitalocean", "k3s", "vultr":
			kcfg, err = k8s.CreateKubeConfig(false, clctrl.ProviderConfig.Kubeconfig)
			if err != nil {
				return fmt.Errorf("failed to create Kubernetes config: %w", err)
			}
		case "google":
			var err error
			kcfg, err = clctrl.GoogleClient.GetContainerClusterAuth(clctrl.ClusterName, []byte(clctrl.GoogleAuth.KeyFile))
			if err != nil {
				return fmt.Errorf("failed to get Google container cluster auth: %w", err)
			}
		}

		telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.UsersTerraformApplyStarted, "")
		log.Info().Msg("applying users terraform")

		tfEnvs := map[string]string{}
		var tfEntrypoint, terraformClient string

		switch clctrl.CloudProvider {
		case "akamai":
			tfEnvs = akamaiext.GetAkamaiTerraformEnvs(tfEnvs, cl)
			tfEnvs = akamaiext.GetUsersTerraformEnvs(kcfg.Clientset, cl, tfEnvs)
		case "aws":
			tfEnvs = awsext.GetAwsTerraformEnvs(tfEnvs, cl)
			tfEnvs = awsext.GetUsersTerraformEnvs(kcfg.Clientset, cl, tfEnvs)
		case "azure":
			tfEnvs = azureext.GetAzureTerraformEnvs(tfEnvs, cl)
			tfEnvs = azureext.GetUsersTerraformEnvs(kcfg.Clientset, cl, tfEnvs)
		case "civo":
			tfEnvs = civoext.GetCivoTerraformEnvs(tfEnvs, cl)
			tfEnvs = civoext.GetUsersTerraformEnvs(kcfg.Clientset, cl, tfEnvs)
		case "google":
			tfEnvs = googleext.GetGoogleTerraformEnvs(tfEnvs, cl)
			tfEnvs = googleext.GetUsersTerraformEnvs(kcfg.Clientset, cl, tfEnvs)
		case "digitalocean":
			tfEnvs = digitaloceanext.GetDigitaloceanTerraformEnvs(tfEnvs, cl)
			tfEnvs = digitaloceanext.GetUsersTerraformEnvs(kcfg.Clientset, cl, tfEnvs)
		case "vultr":
			tfEnvs = vultrext.GetVultrTerraformEnvs(tfEnvs, cl)
			tfEnvs = vultrext.GetUsersTerraformEnvs(kcfg.Clientset, cl, tfEnvs)
		case "k3s":
			tfEnvs = k3sext.GetK3sTerraformEnvs(tfEnvs, cl)
			tfEnvs = k3sext.GetUsersTerraformEnvs(kcfg.Clientset, cl, tfEnvs)
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
				return fmt.Errorf("failed to apply users terraform on retry: %w", err)
			}
		}
		log.Info().Msg("executed users terraform successfully")
		telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.UsersTerraformApplyCompleted, "")

		clctrl.VaultAuth.RootToken = tfEnvs["VAULT_TOKEN"]

		clctrl.Cluster.VaultAuth.RootToken = clctrl.VaultAuth.RootToken
		err = secrets.UpdateCluster(clctrl.KubernetesClient, clctrl.Cluster)
		if err != nil {
			return fmt.Errorf("failed to update cluster after applying terraform: %w", err)
		}

		// Set kbot password in object
		err = clctrl.GetUserPassword("kbot")
		if err != nil {
			log.Info().Msgf("error fetching kbot password: %s", err)
		}

		clctrl.Cluster.UsersTerraformApplyCheck = true
		err = secrets.UpdateCluster(clctrl.KubernetesClient, clctrl.Cluster)
		if err != nil {
			return fmt.Errorf("failed to update cluster with new status details: %w", err)
		}
	}

	return nil
}
