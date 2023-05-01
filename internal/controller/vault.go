/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"

	awsext "github.com/kubefirst/kubefirst-api/extensions/aws"
	civoext "github.com/kubefirst/kubefirst-api/extensions/civo"
	digitaloceanext "github.com/kubefirst/kubefirst-api/extensions/digitalocean"
	vultrext "github.com/kubefirst/kubefirst-api/extensions/vultr"
	"github.com/kubefirst/kubefirst-api/internal/telemetryShim"
	"github.com/kubefirst/runtime/pkg"
	awsinternal "github.com/kubefirst/runtime/pkg/aws"
	"github.com/kubefirst/runtime/pkg/civo"
	"github.com/kubefirst/runtime/pkg/digitalocean"
	"github.com/kubefirst/runtime/pkg/k3d"
	"github.com/kubefirst/runtime/pkg/k8s"
	"github.com/kubefirst/runtime/pkg/segment"
	"github.com/kubefirst/runtime/pkg/terraform"
	"github.com/kubefirst/runtime/pkg/vault"
	"github.com/kubefirst/runtime/pkg/vultr"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InitializeVault
func (clctrl *ClusterController) InitializeVault() error {
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

	if !cl.VaultInitializedCheck {
		var kcfg *k8s.KubernetesClient
		var vaultHandlerPath string

		switch clctrl.CloudProvider {
		case "aws":
			kcfg = awsext.CreateEKSKubeconfig(&clctrl.AwsClient.Config, clctrl.ClusterName)
		case "civo":
			kcfg = k8s.CreateKubeConfig(false, clctrl.ProviderConfig.(*civo.CivoConfig).Kubeconfig)
			vaultHandlerPath = "github.com:kubefirst/manifests.git/vault-handler/replicas-3"
		case "digitalocean":
			kcfg = k8s.CreateKubeConfig(false, clctrl.ProviderConfig.(*digitalocean.DigitaloceanConfig).Kubeconfig)
			vaultHandlerPath = "github.com:kubefirst/manifests.git/vault-handler/replicas-3"
		case "k3d":
			kcfg = k8s.CreateKubeConfig(false, clctrl.ProviderConfig.(k3d.K3dConfig).Kubeconfig)
			vaultHandlerPath = "github.com:kubefirst/manifests.git/vault-handler/replicas-1"
		case "vultr":
			kcfg = k8s.CreateKubeConfig(false, clctrl.ProviderConfig.(*vultr.VultrConfig).Kubeconfig)
			vaultHandlerPath = "github.com:kubefirst/manifests.git/vault-handler/replicas-3"
		}

		telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricVaultInitializationStarted, "")

		switch clctrl.CloudProvider {
		case "aws":
			vaultClient := &vault.Conf

			initResponse, err := vaultClient.AutoUnseal()
			if err != nil {
				return err
			}

			vaultRootToken := initResponse.RootToken

			dataToWrite := make(map[string][]byte)
			dataToWrite["root-token"] = []byte(vaultRootToken)
			for i, value := range initResponse.Keys {
				dataToWrite[fmt.Sprintf("root-unseal-key-%v", i+1)] = []byte(value)
			}
			secret := v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      vault.VaultSecretName,
					Namespace: vault.VaultNamespace,
				},
				Data: dataToWrite,
			}

			err = k8s.CreateSecretV2(kcfg.Clientset, &secret)
			if err != nil {
				return err
			}
		case "civo", "digitalocean", "k3d", "vultr":
			// Initialize and unseal Vault
			// Build and apply manifests
			yamlData, err := kcfg.KustomizeBuild(vaultHandlerPath)
			if err != nil {
				return err
			}
			output, err := kcfg.SplitYAMLFile(yamlData)
			if err != nil {
				return err
			}
			err = kcfg.ApplyObjects("", output)
			if err != nil {
				return err
			}

			// Wait for the Job to finish
			job, err := k8s.ReturnJobObject(kcfg.Clientset, "vault", "vault-handler")
			if err != nil {
				return err
			}
			_, err = k8s.WaitForJobComplete(kcfg.Clientset, job, 240)
			if err != nil {
				msg := fmt.Sprintf("could not run vault unseal job: %s", err)
				telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricVaultInitializationFailed, msg)
				log.Error(msg)
			}
		}

		telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricVaultInitializationCompleted, "")

		err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "vault_initialized_check", true)
		if err != nil {
			return err
		}
	}

	return nil
}

// RunVaultTerraform
func (clctrl *ClusterController) RunVaultTerraform() error {
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

	if !cl.VaultTerraformApplyCheck {
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

		telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricVaultTerraformApplyStarted, "")

		tfEnvs := map[string]string{}
		var usernamePasswordString, base64DockerAuth, registryAuth string

		if clctrl.GitProvider == "gitlab" {
			registryAuth, err = clctrl.ContainerRegistryAuth()
			if err != nil {
				return err
			}

			usernamePasswordString = fmt.Sprintf("%s:%s", "container-registry-auth", registryAuth)
			base64DockerAuth = base64.StdEncoding.EncodeToString([]byte(usernamePasswordString))
		} else {
			usernamePasswordString = fmt.Sprintf("%s:%s", clctrl.GitUser, clctrl.GitToken)
			base64DockerAuth = base64.StdEncoding.EncodeToString([]byte(usernamePasswordString))
		}

		log.Info("configuring vault with terraform")

		var tfEntrypoint, terraformClient string

		switch clctrl.CloudProvider {
		case "aws":
			tfEnvs = awsext.GetVaultTerraformEnvs(kcfg.Clientset, &cl, tfEnvs)
			tfEnvs = awsext.GetAwsTerraformEnvs(tfEnvs, &cl)
			tfEnvs["TF_VAR_b64_docker_auth"] = base64DockerAuth

			if clctrl.GitProvider == "gitlab" {
				tfEnvs["TF_VAR_container_registry_auth"] = registryAuth
				tfEnvs["TF_VAR_owner_group_id"] = strconv.Itoa(clctrl.GitlabOwnerGroupID)
			}

			tfEntrypoint = clctrl.ProviderConfig.(*awsinternal.AwsConfig).GitopsDir + "/terraform/vault"
			terraformClient = clctrl.ProviderConfig.(*awsinternal.AwsConfig).TerraformClient
		case "civo":
			tfEnvs = civoext.GetVaultTerraformEnvs(kcfg.Clientset, &cl, tfEnvs)
			tfEnvs = civoext.GetCivoTerraformEnvs(tfEnvs, &cl)
			tfEnvs["TF_VAR_b64_docker_auth"] = base64DockerAuth

			if clctrl.GitProvider == "gitlab" {
				tfEnvs["TF_VAR_container_registry_auth"] = registryAuth
				tfEnvs["TF_VAR_owner_group_id"] = strconv.Itoa(clctrl.GitlabOwnerGroupID)
			}

			tfEntrypoint = clctrl.ProviderConfig.(*civo.CivoConfig).GitopsDir + "/terraform/vault"
			terraformClient = clctrl.ProviderConfig.(*civo.CivoConfig).TerraformClient
		case "digitalocean":
			tfEnvs = digitaloceanext.GetVaultTerraformEnvs(kcfg.Clientset, &cl, tfEnvs)
			tfEnvs = digitaloceanext.GetDigitaloceanTerraformEnvs(tfEnvs, &cl)
			tfEnvs["TF_VAR_b64_docker_auth"] = base64DockerAuth

			if clctrl.GitProvider == "gitlab" {
				tfEnvs["TF_VAR_container_registry_auth"] = registryAuth
				tfEnvs["TF_VAR_owner_group_id"] = strconv.Itoa(clctrl.GitlabOwnerGroupID)
			}

			tfEntrypoint = clctrl.ProviderConfig.(*digitalocean.DigitaloceanConfig).GitopsDir + "/terraform/vault"
			terraformClient = clctrl.ProviderConfig.(*digitalocean.DigitaloceanConfig).TerraformClient
		case "k3d":
			kubernetesInClusterAPIService, err := k8s.ReadService(clctrl.ProviderConfig.(k3d.K3dConfig).Kubeconfig, "default", "kubernetes")
			if err != nil {
				log.Errorf("error looking up kubernetes api server service: %s")
				return err
			}

			var vaultRootToken string
			secData, err := k8s.ReadSecretV2(kcfg.Clientset, "vault", "vault-unseal-secret")
			if err != nil {
				return err
			}

			vaultRootToken = secData["root-token"]

			tfEnvs["TF_VAR_email_address"] = "your@email.com"
			tfEnvs[fmt.Sprintf("TF_VAR_%s_token", clctrl.GitProvider)] = clctrl.GitToken
			tfEnvs["TF_VAR_vault_addr"] = k3d.VaultPortForwardURL
			tfEnvs["TF_VAR_b64_docker_auth"] = base64DockerAuth
			tfEnvs["TF_VAR_vault_token"] = vaultRootToken
			tfEnvs["VAULT_ADDR"] = k3d.VaultPortForwardURL
			tfEnvs["VAULT_TOKEN"] = vaultRootToken
			tfEnvs["TF_VAR_atlantis_repo_webhook_secret"] = clctrl.AtlantisWebhookSecret
			tfEnvs["TF_VAR_kbot_ssh_private_key"] = cl.PrivateKey
			tfEnvs["TF_VAR_kbot_ssh_public_key"] = cl.PublicKey
			tfEnvs["TF_VAR_kubernetes_api_endpoint"] = fmt.Sprintf("https://%s", kubernetesInClusterAPIService.Spec.ClusterIP)
			tfEnvs[fmt.Sprintf("%s_OWNER", strings.ToUpper(clctrl.GitProvider))] = clctrl.GitOwner
			tfEnvs["AWS_ACCESS_KEY_ID"] = pkg.MinioDefaultUsername
			tfEnvs["AWS_SECRET_ACCESS_KEY"] = pkg.MinioDefaultPassword
			tfEnvs["TF_VAR_aws_access_key_id"] = pkg.MinioDefaultUsername
			tfEnvs["TF_VAR_aws_secret_access_key"] = pkg.MinioDefaultPassword
			// tfEnvs["TF_LOG"] = "DEBUG"

			tfEntrypoint = clctrl.ProviderConfig.(k3d.K3dConfig).GitopsDir + "/terraform/vault"
			terraformClient = clctrl.ProviderConfig.(k3d.K3dConfig).TerraformClient
		case "vultr":
			tfEnvs = vultrext.GetVaultTerraformEnvs(kcfg.Clientset, &cl, tfEnvs)
			tfEnvs = vultrext.GetVultrTerraformEnvs(tfEnvs, &cl)
			tfEnvs["TF_VAR_b64_docker_auth"] = base64DockerAuth

			if clctrl.GitProvider == "gitlab" {
				tfEnvs["TF_VAR_container_registry_auth"] = registryAuth
				tfEnvs["TF_VAR_owner_group_id"] = strconv.Itoa(clctrl.GitlabOwnerGroupID)
			}

			tfEntrypoint = clctrl.ProviderConfig.(*vultr.VultrConfig).GitopsDir + "/terraform/vault"
			terraformClient = clctrl.ProviderConfig.(*vultr.VultrConfig).TerraformClient
		}

		err = terraform.InitApplyAutoApprove(terraformClient, tfEntrypoint, tfEnvs)
		if err != nil {
			log.Errorf("error applying vault terraform: %s", err)
			telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricVaultTerraformApplyFailed, err.Error())
			return err
		}

		log.Info("vault terraform executed successfully")
		telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricVaultTerraformApplyCompleted, "")

		err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "vault_terraform_apply_check", true)
		if err != nil {
			return err
		}
	}

	return nil
}

// WaitForVault
func (clctrl *ClusterController) WaitForVault() error {
	// Logging handler
	// Logs to stdout to maintain compatibility with event streaming
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "",
	})
	log.SetReportCaller(false)
	log.SetOutput(os.Stdout)

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

	vaultStatefulSet, err := k8s.ReturnStatefulSetObject(
		kcfg.Clientset,
		"app.kubernetes.io/instance",
		"vault",
		"vault",
		1200,
	)
	if err != nil {
		log.Errorf("error finding Vault StatefulSet: %s", err)
		return err
	}
	_, err = k8s.WaitForStatefulSetReady(kcfg.Clientset, vaultStatefulSet, 120, true)
	if err != nil {
		log.Errorf("error waiting for Vault StatefulSet ready state: %s", err)
		return err
	}

	return nil
}
