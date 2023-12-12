/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	vaultapi "github.com/hashicorp/vault/api"
	awsext "github.com/kubefirst/kubefirst-api/extensions/aws"
	civoext "github.com/kubefirst/kubefirst-api/extensions/civo"
	digitaloceanext "github.com/kubefirst/kubefirst-api/extensions/digitalocean"
	googleext "github.com/kubefirst/kubefirst-api/extensions/google"
	terraformext "github.com/kubefirst/kubefirst-api/extensions/terraform"
	vultrext "github.com/kubefirst/kubefirst-api/extensions/vultr"
	"github.com/kubefirst/metrics-client/pkg/telemetry"
	"github.com/kubefirst/runtime/pkg/k8s"
	vault "github.com/kubefirst/runtime/pkg/vault"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InitializeVault
func (clctrl *ClusterController) GetUserPassword(user string) error {
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

	// empty conf
	vaultConf := &vault.Conf
	//sets up vault client within function
	clctrl.VaultAuth.KbotPassword, err = vaultConf.GetUserPassword(vault.VaultDefaultAddress, cl.VaultAuth.RootToken, "kbot", "initial-password")
	if err != nil {
		return err
	}

	err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "vault_auth.kbot_password", clctrl.VaultAuth.KbotPassword)
	if err != nil {
		return err
	}

	return nil
}

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

	if !cl.VaultInitializedCheck {
		var kcfg *k8s.KubernetesClient
		var vaultHandlerPath string

		switch clctrl.CloudProvider {
		case "aws":
			kcfg = awsext.CreateEKSKubeconfig(&clctrl.AwsClient.Config, clctrl.ClusterName)
		case "civo", "digitalocean", "vultr":
			kcfg = k8s.CreateKubeConfig(false, clctrl.ProviderConfig.Kubeconfig)
			vaultHandlerPath = "github.com:kubefirst/manifests.git/vault-handler/replicas-3"
		case "google":
			var err error
			kcfg, err = clctrl.GoogleClient.GetContainerClusterAuth(clctrl.ClusterName, []byte(clctrl.GoogleAuth.KeyFile))
			if err != nil {
				return err
			}
		}

		telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.VaultInitializationStarted, "")

		switch clctrl.CloudProvider {
		case "aws", "google":
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
		case "civo", "digitalocean", "vultr":
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
				telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.VaultInitializationFailed, err.Error())
				log.Error(msg)
			}
		}
		telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.VaultInitializationCompleted, "")

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

	if !cl.VaultTerraformApplyCheck {
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

		telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.VaultTerraformApplyStarted, "")

		tfEnvs := map[string]string{}

		//Common TfEnvs
		var usernamePasswordString, base64DockerAuth, registryAuth string

		if clctrl.GitProvider == "gitlab" {
			registryAuth, err = clctrl.ContainerRegistryAuth()
			if err != nil {
				return err
			}

			usernamePasswordString = fmt.Sprintf("%s:%s", "container-registry-auth", registryAuth)
			base64DockerAuth = base64.StdEncoding.EncodeToString([]byte(usernamePasswordString))
		} else {
			usernamePasswordString = fmt.Sprintf("%s:%s", clctrl.GitAuth.User, clctrl.GitAuth.Token)
			base64DockerAuth = base64.StdEncoding.EncodeToString([]byte(usernamePasswordString))
		}

		tfEnvs["TF_VAR_b64_docker_auth"] = base64DockerAuth

		if clctrl.GitProvider == "gitlab" {
			tfEnvs["TF_VAR_container_registry_auth"] = registryAuth
			tfEnvs["TF_VAR_owner_group_id"] = strconv.Itoa(clctrl.GitlabOwnerGroupID)
		}

		//Specific TfEnvs
		switch clctrl.CloudProvider {
		case "aws":
			tfEnvs = awsext.GetVaultTerraformEnvs(kcfg.Clientset, &cl, tfEnvs)
			tfEnvs = awsext.GetAwsTerraformEnvs(tfEnvs, &cl)

		case "civo":
			tfEnvs = civoext.GetVaultTerraformEnvs(kcfg.Clientset, &cl, tfEnvs)
			tfEnvs = civoext.GetCivoTerraformEnvs(tfEnvs, &cl)
		case "google":
			tfEnvs = googleext.GetVaultTerraformEnvs(kcfg.Clientset, &cl, tfEnvs)
			tfEnvs = googleext.GetGoogleTerraformEnvs(tfEnvs, &cl)
		case "digitalocean":
			tfEnvs = digitaloceanext.GetVaultTerraformEnvs(kcfg.Clientset, &cl, tfEnvs)
			tfEnvs = digitaloceanext.GetDigitaloceanTerraformEnvs(tfEnvs, &cl)
		case "vultr":
			tfEnvs = vultrext.GetVaultTerraformEnvs(kcfg.Clientset, &cl, tfEnvs)
			tfEnvs = vultrext.GetVultrTerraformEnvs(tfEnvs, &cl)
		}

		tfEntrypoint := clctrl.ProviderConfig.GitopsDir + "/terraform/vault"
		terraformClient := clctrl.ProviderConfig.TerraformClient

		log.Info("configuring vault with terraform")
		err = terraformext.InitApplyAutoApprove(terraformClient, tfEntrypoint, tfEnvs)
		if err != nil {
			log.Errorf("error applying vault terraform: %s", err)
			log.Info("sleeping 10 seconds before retrying terraform execution once more")
			time.Sleep(10 * time.Second)
			err = terraformext.InitApplyAutoApprove(terraformClient, tfEntrypoint, tfEnvs)
			if err != nil {
				log.Errorf("error applying vault terraform: %s", err)
				telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.VaultTerraformApplyFailed, err.Error())
				return err
			}
		}

		log.Info("vault terraform executed successfully")
		telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.VaultTerraformApplyCompleted, "")

		err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "vault_terraform_apply_check", true)
		if err != nil {
			return err
		}
	}

	return nil
}

func (clctrl *ClusterController) WriteVaultSecrets() error {
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

	vaultAddr := "http://localhost:8200"

	vaultClient, err := vaultapi.NewClient(&vaultapi.Config{
		Address: vaultAddr,
	})
	if err != nil {
		log.Errorf("error creating vault client: %s", err)
		return err
	}

	var externalDnsToken string
	switch cl.DnsProvider {
	case "civo":
		externalDnsToken = cl.CivoAuth.Token
	case "vultr":
		externalDnsToken = cl.VultrAuth.Token
	case "digitalocean":
		externalDnsToken = cl.DigitaloceanAuth.Token
	case "aws":
		externalDnsToken = "implement with cluster management"
	case "google":
		externalDnsToken = "implement with cluster management"
	case "cloudflare":
		externalDnsToken = cl.CloudflareAuth.APIToken
	}
	//
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

	clientset := kcfg.Clientset

	var vaultRootToken string
	vaultUnsealSecretData, err := k8s.ReadSecretV2(clientset, "vault", "vault-unseal-secret")
	if err != nil {
		log.Errorf("error reading vault-unseal-secret: %s", err)
	}
	if len(vaultUnsealSecretData) != 0 {
		vaultRootToken = vaultUnsealSecretData["root-token"]
	}
	vaultClient.SetToken(vaultRootToken)

	_, err = vaultClient.KVv2("secret").Put(context.Background(), "external-dns", map[string]interface{}{
		"token": externalDnsToken,
	})

	_, err = vaultClient.KVv2("secret").Put(context.Background(), "cloudflare", map[string]interface{}{
		"origin-ca-api-key": cl.CloudflareAuth.OriginCaIssuerKey,
	})

	if cl.CloudProvider == "google" {
		log.Info("writing google specific secrets to vault secret store")
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("error getting home path: %s", err)
		}
		writeGoogleSecrets(homeDir, vaultClient)
		log.Info("successfully wrote google specific secrets to vault")
	}

	if err != nil {
		log.Errorf("error writing secret to vault: %s", err)
		return err
	}

	log.Info("successfully wrote platform secrets to vault secret store")
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
	case "civo", "digitalocean", "vultr":
		kcfg = k8s.CreateKubeConfig(false, clctrl.ProviderConfig.Kubeconfig)
	case "google":
		var err error
		kcfg, err = clctrl.GoogleClient.GetContainerClusterAuth(clctrl.ClusterName, []byte(clctrl.GoogleAuth.KeyFile))
		if err != nil {
			return err
		}
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
	_, err = k8s.WaitForStatefulSetReady(kcfg.Clientset, vaultStatefulSet, 300, true)
	if err != nil {
		log.Errorf("error waiting for Vault StatefulSet ready state: %s", err)
		return err
	}

	return nil
}

func writeGoogleSecrets(homeDir string, vaultClient *vaultapi.Client) error {

	// vault path - gcp/application-default-credentials
	adcJSON, err := os.ReadFile(fmt.Sprintf("%s/.k1/application-default-credentials.json", homeDir))
	if err != nil {
		log.Error("error: reading google json credentials file")
		return err
	}

	var data map[string]interface{}
	err = json.Unmarshal([]byte(adcJSON), &data)
	if err != nil {
		return err
	}

	data["private_key"] = strings.Replace(data["private_key"].(string), "\n", "\\n", -1)

	_, err = vaultClient.KVv2("secret").Put(context.Background(), "gcp/application-default-credentials", data)
	if err != nil {
		return err
	}
	return nil
}
