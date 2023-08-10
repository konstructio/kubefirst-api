/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package types

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ClusterDefinition describes an incoming request to create a cluster
type ClusterDefinition struct {

	//Cluster
	AdminEmail    string `json:"admin_email" binding:"required"`
	CloudProvider string `json:"cloud_provider" binding:"required,oneof=aws civo digitalocean vultr"`
	CloudRegion   string `json:"cloud_region" binding:"required"`
	ClusterName   string `json:"cluster_name,omitempty"`
	DomainName    string `json:"domain_name" binding:"required"`
	DnsProvider   string `json:"dns_provider,omitempty" binding:"required"`
	Type          string `json:"type" binding:"required,oneof=mgmt workload"`

	//Git
	GitopsTemplateURL    string `json:"gitops_template_url"`
	GitopsTemplateBranch string `json:"gitops_template_branch"`
	GitProvider          string `json:"git_provider" binding:"required,oneof=github gitlab"`
	GitProtocol          string `bson:"git_protocol" json:"git_protocol" binding:"required,oneof=ssh https"`

	//AWS
	ECR bool `json:"ecr,omitempty"`

	//Auth
	AWSAuth          AWSAuth          `json:"aws_auth,omitempty"`
	CivoAuth         CivoAuth         `json:"civo_auth,omitempty"`
	DigitaloceanAuth DigitaloceanAuth `json:"do_auth,omitempty"`
	VultrAuth        VultrAuth        `json:"vultr_auth,omitempty"`
	CloudflareAuth   CloudflareAuth   `json:"cloudflare_auth,omitempty"`
	GitAuth          GitAuth          `json:"git_auth,omitempty"`
}

// Cluster describes the configuration storage for a Kubefirst cluster object
type Cluster struct {
	ID                primitive.ObjectID `bson:"_id" json:"_id"`
	CreationTimestamp string             `bson:"creation_timestamp" json:"creation_timestamp"`

	// Status
	Status        string `bson:"status" json:"status"`
	LastCondition string `bson:"last_condition" json:"last_condition"`
	InProgress    bool   `bson:"in_progress" json:"in_progress"`

	// Identifiers
	AlertsEmail   string `bson:"alerts_email" json:"alerts_email"`
	CloudProvider string `bson:"cloud_provider" json:"cloud_provider"`
	CloudRegion   string `bson:"cloud_region" json:"cloud_region"`
	ClusterName   string `bson:"cluster_name" json:"cluster_name"`
	ClusterID     string `bson:"cluster_id" json:"cluster_id"`
	ClusterType   string `bson:"cluster_type" json:"cluster_type"`
	DomainName    string `bson:"domain_name" json:"domain_name"`
	DnsProvider   string `bson:"dns_provider" json:"dns_provider"`

	// Auth
	AWSAuth          AWSAuth          `bson:"aws_auth,omitempty" json:"aws_auth,omitempty"`
	CivoAuth         CivoAuth         `bson:"civo_auth,omitempty" json:"civo_auth,omitempty"`
	DigitaloceanAuth DigitaloceanAuth `bson:"do_auth,omitempty" json:"do_auth,omitempty"`
	VultrAuth        VultrAuth        `bson:"vultr_auth,omitempty" json:"vultr_auth,omitempty"`
	CloudflareAuth   CloudflareAuth   `bson:"cf_api_token,omitempty" json:"cf_api_token,omitempty"`
	GitAuth          GitAuth          `json:"git_auth,omitempty"`

	GitopsTemplateURL    string `bson:"gitops_template_url" json:"gitops_template_url"`
	GitopsTemplateBranch string `bson:"gitops_template_branch" json:"gitops_template_branch"`
	GitProvider          string `bson:"git_provider" json:"git_provider"`
	GitProtocol          string `bson:"git_protocol" json:"git_protocol"`
	GitHost              string `bson:"git_host" json:"git_host"`
	GitlabOwnerGroupID   int    `bson:"gitlab_owner_group_id" json:"gitlab_owner_group_id"`

	AtlantisWebhookSecret string `bson:"atlantis_webhook_secret" json:"atlantis_webhook_secret"`
	AtlantisWebhookURL    string `bson:"atlantis_webhook_url" json:"atlantis_webhook_url"`
	KubefirstTeam         string `bson:"kubefirst_team" json:"kubefirst_team"`

	StateStoreCredentials StateStoreCredentials `bson:"state_store_credentials,omitempty" json:"state_store_credentials,omitempty"`
	StateStoreDetails     StateStoreDetails     `bson:"state_store_details,omitempty" json:"state_store_details,omitempty"`

	ArgoCDUsername  string `bson:"argocd_username" json:"argocd_username"`
	ArgoCDPassword  string `bson:"argocd_password" json:"argocd_password"`
	ArgoCDAuthToken string `bson:"argocd_auth_token" json:"argocd_auth_token"`

	//container Registry
	ECR bool `bson:"ecr" json:"ecr"`

	// kms
	AWSAccountId              string `bson:"aws_account_id,omitempty" json:"aws_account_id,omitempty"`
	AWSKMSKeyId               string `bson:"aws_kms_key_id,omitempty" json:"aws_kms_key_id,omitempty"`
	AWSKMSKeyDetokenizedCheck bool   `bson:"aws_kms_key_detokenized_check" json:"aws_kms_key_detokenized_check"`

	// Telemetry
	UseTelemetry bool `bson:"use_telemetry"`

	// Checks
	InstallToolsCheck              bool `bson:"install_tools_check" json:"install_tools_check"`
	DomainLivenessCheck            bool `bson:"domain_liveness_check" json:"domain_liveness_check"`
	StateStoreCredsCheck           bool `bson:"state_store_creds_check" json:"state_store_creds_check"`
	StateStoreCreateCheck          bool `bson:"state_store_create_check" json:"state_store_create_check"`
	GitInitCheck                   bool `bson:"git_init_check" json:"git_init_check"`
	KbotSetupCheck                 bool `bson:"kbot_setup_check" json:"kbot_setup_check"`
	GitopsReadyCheck               bool `bson:"gitops_ready_check" json:"gitops_ready_check"`
	GitTerraformApplyCheck         bool `bson:"git_terraform_apply_check" json:"git_terraform_apply_check"`
	GitopsPushedCheck              bool `bson:"gitops_pushed_check" json:"gitops_pushed_check"`
	CloudTerraformApplyCheck       bool `bson:"cloud_terraform_apply_check" json:"cloud_terraform_apply_check"`
	CloudTerraformApplyFailedCheck bool `bson:"cloud_terraform_apply_failed_check" json:"cloud_terraform_apply_failed_check"`
	ClusterSecretsCreatedCheck     bool `bson:"cluster_secrets_created_check" json:"cluster_secrets_created_check"`
	ArgoCDInstallCheck             bool `bson:"argocd_install_check" json:"argocd_install_check"`
	ArgoCDInitializeCheck          bool `bson:"argocd_initialize_check" json:"argocd_initialize_check"`
	ArgoCDCreateRegistryCheck      bool `bson:"argocd_create_registry_check" json:"argocd_create_registry_check"`
	ArgoCDDeleteRegistryCheck      bool `bson:"argocd_delete_registry_check" json:"argocd_delete_registry_check"`
	VaultInitializedCheck          bool `bson:"vault_initialized_check" json:"vault_initialized_check"`
	VaultTerraformApplyCheck       bool `bson:"vault_terraform_apply_check" json:"vault_terraform_apply_check"`
	UsersTerraformApplyCheck       bool `bson:"users_terraform_apply_check" json:"users_terraform_apply_check"`
}

// StateStoreDetails
type StateStoreDetails struct {
	Name                string `bson:"name,omitempty" json:"name,omitempty"`
	ID                  string `bson:"id,omitempty" json:"id,omitempty"`
	Hostname            string `bson:"hostname,omitempty" json:"hostname,omitempty"`
	AWSStateStoreBucket string `bson:"aws_state_store_bucket,omitempty" json:"aws_state_store_bucket,omitempty"`
	AWSArtifactsBucket  string `bson:"aws_artifacts_bucket,omitempty" json:"aws_artifacts_bucket,omitempty"`
}

// PushBucketObject
type PushBucketObject struct {
	LocalFilePath  string `json:"local_file_path"`
	RemoteFilePath string `json:"remote_file_path"`
	ContentType    string `json:"content_type"`
}

// ImportClusterRequest
type ImportClusterRequest struct {
	ClusterName           string                `bson:"cluster_name" json:"cluster_name"`
	CloudRegion           string                `bson:"cloud_region" json:"cloud_region"`
	CloudProvider         string                `bson:"cloud_provider" json:"cloud_provider"`
	StateStoreCredentials StateStoreCredentials `bson:"state_store_credentials" json:"state_store_credentials"`
	StateStoreDetails     StateStoreDetails     `bson:"state_store_details" json:"state_store_details"`
}
