package azure

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/konstructio/kubefirst-api/internal/vault"
	"github.com/konstructio/kubefirst-api/pkg/k8s"
	"github.com/konstructio/kubefirst-api/pkg/providerConfigs"
	pkgtypes "github.com/konstructio/kubefirst-api/pkg/types"
	"k8s.io/client-go/kubernetes"
)

func readVaultTokenFromSecret(clientset kubernetes.Interface) string {
	existingKubernetesSecret, err := k8s.ReadSecretV2(clientset, vault.VaultNamespace, vault.VaultSecretName)
	if err != nil || existingKubernetesSecret == nil {
		log.Printf("Error reading existing Secret data: %s", err)
		return ""
	}

	return existingKubernetesSecret["root-token"]
}

func GetAzureTerraformEnvs(envs map[string]string, cl *pkgtypes.Cluster) map[string]string {
	envs["ARM_CLIENT_ID"] = cl.AzureAuth.ClientID
	envs["ARM_CLIENT_SECRET"] = cl.AzureAuth.ClientSecret
	envs["ARM_TENANT_ID"] = cl.AzureAuth.TenantID
	envs["ARM_SUBSCRIPTION_ID"] = cl.AzureAuth.SubscriptionID

	return envs
}

func GetGithubTerraformEnvs(envs map[string]string, cl *pkgtypes.Cluster) map[string]string {
	envs["GITHUB_TOKEN"] = cl.GitAuth.Token
	envs["GITHUB_OWNER"] = cl.GitAuth.Owner
	envs["TF_VAR_atlantis_repo_webhook_secret"] = cl.AtlantisWebhookSecret
	envs["TF_VAR_kbot_ssh_public_key"] = cl.GitAuth.PublicKey
	envs["ARM_CLIENT_ID"] = cl.AzureAuth.ClientID
	envs["ARM_CLIENT_SECRET"] = cl.AzureAuth.ClientSecret
	envs["ARM_TENANT_ID"] = cl.AzureAuth.TenantID
	envs["ARM_SUBSCRIPTION_ID"] = cl.AzureAuth.SubscriptionID

	return envs
}

func GetGitlabTerraformEnvs(envs map[string]string, gid int, cl *pkgtypes.Cluster) map[string]string {
	envs["GITLAB_TOKEN"] = cl.GitAuth.Token
	envs["GITLAB_OWNER"] = cl.GitAuth.Owner
	envs["TF_VAR_atlantis_repo_webhook_secret"] = cl.AtlantisWebhookSecret
	envs["TF_VAR_atlantis_repo_webhook_url"] = cl.AtlantisWebhookURL
	envs["TF_VAR_kbot_ssh_public_key"] = cl.GitAuth.PublicKey
	envs["ARM_CLIENT_ID"] = cl.AzureAuth.ClientID
	envs["ARM_CLIENT_SECRET"] = cl.AzureAuth.ClientSecret
	envs["ARM_TENANT_ID"] = cl.AzureAuth.TenantID
	envs["ARM_SUBSCRIPTION_ID"] = cl.AzureAuth.SubscriptionID
	envs["TF_VAR_owner_group_id"] = strconv.Itoa(gid)
	envs["TF_VAR_gitlab_owner"] = cl.GitAuth.Owner

	return envs
}

func GetUsersTerraformEnvs(clientset kubernetes.Interface, cl *pkgtypes.Cluster, envs map[string]string) map[string]string {
	envs["VAULT_TOKEN"] = readVaultTokenFromSecret(clientset)
	envs["VAULT_ADDR"] = providerConfigs.VaultPortForwardURL
	envs[fmt.Sprintf("%s_TOKEN", strings.ToUpper(cl.GitProvider))] = cl.GitAuth.Token
	envs[fmt.Sprintf("%s_OWNER", strings.ToUpper(cl.GitProvider))] = cl.GitAuth.Owner
	envs["ARM_CLIENT_ID"] = cl.AzureAuth.ClientID
	envs["ARM_CLIENT_SECRET"] = cl.AzureAuth.ClientSecret
	envs["ARM_TENANT_ID"] = cl.AzureAuth.TenantID
	envs["ARM_SUBSCRIPTION_ID"] = cl.AzureAuth.SubscriptionID

	return envs
}

func GetVaultTerraformEnvs(clientset kubernetes.Interface, cl *pkgtypes.Cluster, envs map[string]string) map[string]string {
	envs[fmt.Sprintf("%s_TOKEN", strings.ToUpper(cl.GitProvider))] = cl.GitAuth.Token
	envs[fmt.Sprintf("%s_OWNER", strings.ToUpper(cl.GitProvider))] = cl.GitAuth.Owner
	envs["TF_VAR_email_address"] = cl.AlertsEmail
	envs["TF_VAR_vault_addr"] = providerConfigs.VaultPortForwardURL
	envs["TF_VAR_vault_token"] = readVaultTokenFromSecret(clientset)
	envs[fmt.Sprintf("TF_VAR_%s_token", cl.GitProvider)] = cl.GitAuth.Token
	envs["VAULT_ADDR"] = providerConfigs.VaultPortForwardURL
	envs["VAULT_TOKEN"] = readVaultTokenFromSecret(clientset)
	envs["TF_VAR_civo_token"] = cl.CivoAuth.Token
	envs["TF_VAR_atlantis_repo_webhook_secret"] = cl.AtlantisWebhookSecret
	envs["TF_VAR_atlantis_repo_webhook_url"] = cl.AtlantisWebhookURL
	envs["TF_VAR_kbot_ssh_private_key"] = cl.GitAuth.PrivateKey
	envs["TF_VAR_kbot_ssh_public_key"] = cl.GitAuth.PublicKey
	envs["TF_VAR_cloudflare_origin_ca_api_key"] = cl.CloudflareAuth.OriginCaIssuerKey
	envs["TF_VAR_cloudflare_api_key"] = cl.CloudflareAuth.APIToken
	envs["ARM_CLIENT_ID"] = cl.AzureAuth.ClientID
	envs["ARM_CLIENT_SECRET"] = cl.AzureAuth.ClientSecret
	envs["ARM_TENANT_ID"] = cl.AzureAuth.TenantID
	envs["ARM_SUBSCRIPTION_ID"] = cl.AzureAuth.SubscriptionID

	if cl.GitProvider == "gitlab" {
		envs["TF_VAR_owner_group_id"] = fmt.Sprint(cl.GitlabOwnerGroupID)
	}

	return envs
}
