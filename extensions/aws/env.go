/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package aws

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/kubefirst/kubefirst-api/internal/types"
	awsinternal "github.com/kubefirst/runtime/pkg/aws"
	"github.com/kubefirst/runtime/pkg/k8s"
	"github.com/kubefirst/runtime/pkg/vault"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
)

func readVaultTokenFromSecret(clientset *kubernetes.Clientset) string {
	existingKubernetesSecret, err := k8s.ReadSecretV2(clientset, vault.VaultNamespace, vault.VaultSecretName)
	if err != nil || existingKubernetesSecret == nil {
		log.Printf("Error reading existing Secret data: %s", err)
		return ""
	}

	return existingKubernetesSecret["root-token"]
}

func GetAwsTerraformEnvs(envs map[string]string, cl *types.Cluster) map[string]string {
	// needed for s3 api connectivity to object storage
	envs["AWS_ACCESS_KEY_ID"] = cl.StateStoreCredentials.AccessKeyID
	envs["AWS_SECRET_ACCESS_KEY"] = cl.StateStoreCredentials.SecretAccessKey
	envs["AWS_SESSION_TOKEN"] = cl.AWSAuth.SessionToken
	envs["TF_VAR_aws_access_key_id"] = cl.StateStoreCredentials.AccessKeyID
	envs["TF_VAR_aws_secret_access_key"] = cl.StateStoreCredentials.SecretAccessKey
	envs["TF_VAR_aws_session_token"] = cl.AWSAuth.SessionToken
	envs["TF_VAR_aws_region"] = cl.CloudRegion
	envs["TF_VAR_hosted_zone_name"] = cl.DomainName
	//envs["TF_LOG"] = "debug"

	return envs
}

func GetGithubTerraformEnvs(envs map[string]string, cl *types.Cluster) map[string]string {
	envs["GITHUB_TOKEN"] = cl.GitToken
	envs["GITHUB_OWNER"] = cl.GitOwner
	envs["TF_VAR_atlantis_repo_webhook_secret"] = cl.AtlantisWebhookSecret
	envs["TF_VAR_atlantis_repo_webhook_url"] = cl.AtlantisWebhookURL
	envs["TF_VAR_kbot_ssh_public_key"] = cl.PublicKey
	envs["AWS_ACCESS_KEY_ID"] = cl.StateStoreCredentials.AccessKeyID
	envs["AWS_SECRET_ACCESS_KEY"] = cl.StateStoreCredentials.SecretAccessKey
	envs["TF_VAR_aws_access_key_id"] = cl.StateStoreCredentials.AccessKeyID
	envs["TF_VAR_aws_secret_access_key"] = cl.StateStoreCredentials.SecretAccessKey

	return envs
}

func GetGitlabTerraformEnvs(envs map[string]string, gid int, cl *types.Cluster) map[string]string {
	envs["GITLAB_TOKEN"] = cl.GitToken
	envs["GITLAB_OWNER"] = cl.GitOwner
	envs["TF_VAR_atlantis_repo_webhook_secret"] = cl.AtlantisWebhookSecret
	envs["TF_VAR_atlantis_repo_webhook_url"] = cl.AtlantisWebhookURL
	envs["TF_VAR_kbot_ssh_public_key"] = cl.PublicKey
	envs["AWS_ACCESS_KEY_ID"] = cl.StateStoreCredentials.AccessKeyID
	envs["AWS_SECRET_ACCESS_KEY"] = cl.StateStoreCredentials.SecretAccessKey
	envs["TF_VAR_aws_access_key_id"] = cl.StateStoreCredentials.AccessKeyID
	envs["TF_VAR_aws_secret_access_key"] = cl.StateStoreCredentials.SecretAccessKey
	envs["TF_VAR_owner_group_id"] = strconv.Itoa(gid)
	envs["TF_VAR_gitlab_owner"] = cl.GitOwner

	return envs
}

func GetUsersTerraformEnvs(clientset *kubernetes.Clientset, cl *types.Cluster, envs map[string]string) map[string]string {
	envs["VAULT_TOKEN"] = readVaultTokenFromSecret(clientset)
	envs["VAULT_ADDR"] = awsinternal.VaultPortForwardURL
	envs[fmt.Sprintf("%s_TOKEN", strings.ToUpper(cl.GitProvider))] = cl.GitToken
	envs[fmt.Sprintf("%s_OWNER", strings.ToUpper(cl.GitProvider))] = cl.GitOwner

	return envs
}

func GetVaultTerraformEnvs(clientset *kubernetes.Clientset, cl *types.Cluster, envs map[string]string) map[string]string {
	envs[fmt.Sprintf("%s_TOKEN", strings.ToUpper(cl.GitProvider))] = cl.GitToken
	envs[fmt.Sprintf("%s_OWNER", strings.ToUpper(cl.GitProvider))] = cl.GitOwner
	envs["TF_VAR_email_address"] = cl.AlertsEmail
	envs["TF_VAR_vault_addr"] = awsinternal.VaultPortForwardURL
	envs["TF_VAR_vault_token"] = readVaultTokenFromSecret(clientset)
	envs[fmt.Sprintf("TF_VAR_%s_token", cl.GitProvider)] = cl.GitToken
	envs["VAULT_ADDR"] = awsinternal.VaultPortForwardURL
	envs["VAULT_TOKEN"] = readVaultTokenFromSecret(clientset)
	envs["TF_VAR_atlantis_repo_webhook_secret"] = cl.AtlantisWebhookSecret
	envs["TF_VAR_atlantis_repo_webhook_url"] = cl.AtlantisWebhookURL
	envs["TF_VAR_kbot_ssh_private_key"] = cl.PrivateKey
	envs["TF_VAR_kbot_ssh_public_key"] = cl.PublicKey

	switch cl.GitProvider {
	case "gitlab":
		envs["TF_VAR_owner_group_id"] = fmt.Sprint(cl.GitlabOwnerGroupID)
	}

	return envs
}
