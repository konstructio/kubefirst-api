/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	"fmt"
	"strings"

	awsext "github.com/kubefirst/kubefirst-api/extensions/aws"
	civoext "github.com/kubefirst/kubefirst-api/extensions/civo"
	digitaloceanext "github.com/kubefirst/kubefirst-api/extensions/digitalocean"
	vultrext "github.com/kubefirst/kubefirst-api/extensions/vultr"
	awsinternal "github.com/kubefirst/runtime/pkg/aws"
	"github.com/kubefirst/runtime/pkg/civo"
	"github.com/kubefirst/runtime/pkg/digitalocean"
	"github.com/kubefirst/runtime/pkg/k3d"
	"github.com/kubefirst/runtime/pkg/k8s"
	"github.com/kubefirst/runtime/pkg/terraform"
	"github.com/kubefirst/runtime/pkg/vultr"
	log "github.com/sirupsen/logrus"
)

// RunUsersTerraform
func (clctrl *ClusterController) RunUsersTerraform() error {
	cl, err := clctrl.MdbCl.GetCluster(clctrl.ClusterName)
	if err != nil {
		return err
	}

	if !cl.UsersTerraformApplyCheck {
		var kcfg *k8s.KubernetesClient

		switch clctrl.CloudProvider {
		case "aws":
			kcfg = awsext.CreateEKSKubeconfig(&clctrl.AwsClient.Config, clctrl.ClusterName)
		case "civo":
			kcfg = k8s.CreateKubeConfig(false, clctrl.ProviderConfig.(*civo.CivoConfig).Kubeconfig)
		case "digitalocean":
			kcfg = k8s.CreateKubeConfig(false, clctrl.ProviderConfig.(*digitalocean.DigitaloceanConfig).Kubeconfig)
		case "k3d":
			kcfg = k8s.CreateKubeConfig(false, clctrl.ProviderConfig.(k3d.K3dConfig).Kubeconfig)
		case "vultr":
			kcfg = k8s.CreateKubeConfig(false, clctrl.ProviderConfig.(*vultr.VultrConfig).Kubeconfig)
		}

		// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricUsersTerraformApplyStarted, "")
		log.Info("applying users terraform")

		var vaultRootToken string
		secData, err := k8s.ReadSecretV2(kcfg.Clientset, "vault", "vault-unseal-secret")
		if err != nil {
			return err
		}
		vaultRootToken = secData["root-token"]

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
		case "k3d":
			tfEnvs["TF_VAR_email_address"] = "your@email.com"
			tfEnvs[fmt.Sprintf("TF_VAR_%s_token", clctrl.GitProvider)] = clctrl.GitToken
			tfEnvs["TF_VAR_vault_addr"] = k3d.VaultPortForwardURL
			tfEnvs["TF_VAR_vault_token"] = vaultRootToken
			tfEnvs["VAULT_ADDR"] = k3d.VaultPortForwardURL
			tfEnvs["VAULT_TOKEN"] = vaultRootToken
			tfEnvs[fmt.Sprintf("%s_TOKEN", strings.ToUpper(clctrl.GitProvider))] = clctrl.GitToken
			tfEnvs[fmt.Sprintf("%s_OWNER", strings.ToUpper(clctrl.GitProvider))] = clctrl.GitOwner

			tfEntrypoint = clctrl.ProviderConfig.(k3d.K3dConfig).GitopsDir + "/terraform/users"
			terraformClient = clctrl.ProviderConfig.(k3d.K3dConfig).TerraformClient
		case "vultr":
			tfEnvs = vultrext.GetVultrTerraformEnvs(tfEnvs, &cl)
			tfEnvs = vultrext.GetUsersTerraformEnvs(kcfg.Clientset, &cl, tfEnvs)
			tfEntrypoint = clctrl.ProviderConfig.(*vultr.VultrConfig).GitopsDir + "/terraform/users"
			terraformClient = clctrl.ProviderConfig.(*vultr.VultrConfig).TerraformClient
		}

		err = terraform.InitApplyAutoApprove(terraformClient, tfEntrypoint, tfEnvs)
		if err != nil {
			// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricUsersTerraformApplyStarted, err.Error())
			return err
		}
		log.Info("executed users terraform successfully")
		// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricUsersTerraformApplyCompleted, "")

		err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "users_terraform_apply_check", true)
		if err != nil {
			return err
		}
	}
	return nil
}
