/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package segment

import "github.com/segmentio/analytics-go"

// SegmentIO constants
// SegmentIOWriteKey The write key is the unique identifier for a source that tells Segment which source data comes
// from, to which workspace the data belongs, and which destinations should receive the data.
const (
	SegmentIOWriteKey = "0gAYkX5RV3vt7s4pqCOOsDb6WHPLT30M"

	// Heartbeat
	MetricKubefirstHeartbeat = "kubefirst.heartbeat"

	// Install
	MetricKubefirstInstalled = "kubefirst.installed"

	// Init
	MetricInitStarted   = "kubefirst.init.started"
	MetricInitCompleted = "kubefirst.init.completed"

	MetricCloudCredentialsCheckStarted   = "kubefirst.init.cloud_credentials_check.started"
	MetricCloudCredentialsCheckCompleted = "kubefirst.init.cloud_credentials_check.completed"
	MetricCloudCredentialsCheckFailed    = "kubefirst.init.cloud_credentials_check.failed"

	MetricDomainLivenessStarted   = "kubefirst.init.domain_liveness.started"
	MetricDomainLivenessCompleted = "kubefirst.init.domain_liveness.completed"
	MetricDomainLivenessFailed    = "kubefirst.init.domain_liveness.failed"

	MetricStateStoreCreateStarted   = "kubefirst.init.state_store_create.started"
	MetricStateStoreCreateCompleted = "kubefirst.init.state_store_create.completed"
	MetricStateStoreCreateFailed    = "kubefirst.init.state_store_create.failed"

	MetricGitCredentialsCheckStarted   = "kubefirst.init.git_credentials_check.started"
	MetricGitCredentialsCheckCompleted = "kubefirst.init.git_credentials_check.completed"
	MetricGitCredentialsCheckFailed    = "kubefirst.init.git_credentials_check.failed"

	MetricKbotSetupStarted   = "kubefirst.init.kbot_setup.started"
	MetricKbotSetupCompleted = "kubefirst.init.kbot_setup.completed"
	MetricKbotSetupFailed    = "kubefirst.init.kbot_setup.failed"

	// Create
	MetricClusterInstallStarted   = "kubefirst.cluster_install.started"
	MetricClusterInstallCompleted = "kubefirst.cluster_install.completed"

	MetricGitTerraformApplyStarted   = "kubefirst.git_terraform_apply.started"
	MetricGitTerraformApplyCompleted = "kubefirst.git_terraform_apply.completed"
	MetricGitTerraformApplyFailed    = "kubefirst.git_terraform_apply.failed"

	MetricGitopsRepoPushStarted   = "kubefirst.gitops_repo_push.started"
	MetricGitopsRepoPushCompleted = "kubefirst.gitops_repo_push.completed"
	MetricGitopsRepoPushFailed    = "kubefirst.gitops_repo_push.failed"

	MetricCloudTerraformApplyStarted   = "kubefirst.cloud_terraform_apply.started"
	MetricCloudTerraformApplyCompleted = "kubefirst.cloud_terraform_apply.completed"
	MetricCloudTerraformApplyFailed    = "kubefirst.cloud_terraform_apply.failed"

	MetricArgoCDInstallStarted   = "kubefirst.argocd_install.started"
	MetricArgoCDInstallCompleted = "kubefirst.argocd_install.completed"
	MetricArgoCDInstallFailed    = "kubefirst.argocd_install.failed"

	MetricCreateRegistryStarted   = "kubefirst.create_registry.started"
	MetricCreateRegistryCompleted = "kubefirst.create_registry.completed"
	MetricCreateRegistryFailed    = "kubefirst.create_registry.failed"

	MetricVaultInitializationStarted   = "kubefirst.vault_initialization.started"
	MetricVaultInitializationCompleted = "kubefirst.vault_initialization.completed"
	MetricVaultInitializationFailed    = "kubefirst.vault_initialization.failed"

	MetricVaultTerraformApplyStarted   = "kubefirst.vault_terraform_apply.started"
	MetricVaultTerraformApplyCompleted = "kubefirst.vault_terraform_apply.completed"
	MetricVaultTerraformApplyFailed    = "kubefirst.vault_terraform_apply.failed"

	MetricUsersTerraformApplyStarted   = "kubefirst.users_terraform_apply.started"
	MetricUsersTerraformApplyCompleted = "kubefirst.users_terraform_apply.completed"
	MetricUsersTerraformApplyFailed    = "kubefirst.users_terraform_apply.failed"

	// Delete
	MetricClusterDeleteStarted   = "kubefirst.cluster_delete.started"
	MetricClusterDeleteCompleted = "kubefirst.cluster_delete.completed"
)

type SegmentClient struct {
	Client            analytics.Client
	CliVersion        string
	CloudProvider     string
	ClusterID         string
	ClusterType       string
	DomainName        string
	GitProvider       string
	InstallMethod     string
	KubefirstClient   string
	KubefirstTeam     string
	KubefirstTeamInfo string
}
