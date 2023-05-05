/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package types

import "go.mongodb.org/mongo-driver/bson/primitive"

// ClusterDefinition is used to create a cluster
type ClusterDefinition struct {
	AdminEmail    string `json:"admin_email" binding:"required"`
	CloudProvider string `json:"cloud_provider" binding:"required,oneof=aws civo digitalocean vultr"`
	CloudRegion   string `json:"cloud_region" binding:"required"`
	ClusterName   string `json:"cluster_name,omitempty"`
	DomainName    string `json:"domain_name" binding:"required"`
	GitProvider   string `json:"git_provider" binding:"required,oneof=github gitlab"`
	GitOwner      string `json:"git_owner" binding:"required"`
	GitToken      string `json:"git_token" binding:"required"`
	Type          string `json:"type" binding:"required,oneof=mgmt workload"`

	AWSAuth          AWSAuth          `json:"aws_auth,omitempty"`
	CivoAuth         CivoAuth         `json:"civo_auth,omitempty"`
	DigitaloceanAuth DigitaloceanAuth `json:"do_auth,omitempty"`
	VultrAuth        VultrAuth        `json:"vultr_auth,omitempty"`
}

// Cluster describes the configuration storage for a Kubefirst cluster object
type Cluster struct {
	ID                primitive.ObjectID `bson:"_id" json:"_id"`
	CreationTimestamp string             `bson:"creation_timestamp" json:"creation_timestamp"`
	Status            string             `bson:"status" json:"status"`
	InProgress        bool               `bson:"in_progress" json:"in_progress"`

	ClusterName   string `bson:"cluster_name" json:"cluster_name"`
	CloudProvider string `bson:"cloud_provider" json:"cloud_provider"`
	CloudRegion   string `bson:"cloud_region" json:"cloud_region"`
	DomainName    string `bson:"domain_name" json:"domain_name"`
	ClusterID     string `bson:"cluster_id" json:"cluster_id"`
	ClusterType   string `bson:"cluster_type" json:"cluster_type"`
	AlertsEmail   string `bson:"alerts_email" json:"alerts_email"`

	// Auth
	AWSAuth          AWSAuth          `bson:"aws_auth,omitempty" json:"aws_auth,omitempty"`
	CivoAuth         CivoAuth         `bson:"civo_auth,omitempty" json:"civo_auth,omitempty"`
	DigitaloceanAuth DigitaloceanAuth `bson:"do_auth,omitempty" json:"do_auth,omitempty"`
	VultrAuth        VultrAuth        `bson:"vultr_auth,omitempty" json:"vultr_auth,omitempty"`

	GitProvider        string `bson:"git_provider" json:"git_provider"`
	GitHost            string `bson:"git_host" json:"git_host"`
	GitOwner           string `bson:"git_owner" json:"git_owner"`
	GitUser            string `bson:"git_user" json:"git_user"`
	GitToken           string `bson:"git_token" json:"git_token"`
	GitlabOwnerGroupID int    `bson:"gitlab_owner_group_id" json:"gitlab_owner_group_id"`

	AtlantisWebhookSecret string `bson:"atlantis_webhook_secret" json:"atlantis_webhook_secret"`
	AtlantisWebhookURL    string `bson:"atlantis_webhook_url" json:"atlantis_webhook_url"`
	KubefirstTeam         string `bson:"kubefirst_team" json:"kubefirst_team"`

	StateStoreCredentials StateStoreCredentials `bson:"state_store_credentials,omitempty" json:"state_store_credentials,omitempty"`
	StateStoreDetails     StateStoreDetails     `bson:"state_store_details,omitempty" json:"state_store_details,omitempty"`

	PublicKey  string `bson:"public_key" json:"public_key"`
	PrivateKey string `bson:"private_key" json:"private_key"`
	PublicKeys string `bson:"public_keys" json:"public_keys"`

	ArgoCDUsername  string `bson:"argocd_username" json:"argocd_username"`
	ArgoCDPassword  string `bson:"argocd_password" json:"argocd_password"`
	ArgoCDAuthToken string `bson:"argocd_auth_token" json:"argocd_auth_token"`

	// kms
	AWSAccountId              string `bson:"aws_account_id,omitempty" json:"aws_account_id,omitempty"`
	AWSKMSKeyId               string `bson:"aws_kms_key_id,omitempty" json:"aws_kms_key_id,omitempty"`
	AWSKMSKeyDetokenizedCheck bool   `bson:"aws_kms_key_detokenized_check" json:"aws_kms_key_detokenized_check"`

	// Telemetry
	UseTelemetry bool `bson:"use_telemetry"`

	// Checks
	GitInitCheck                   bool `bson:"git_init_check" json:"git_init_check"`
	InstallToolsCheck              bool `bson:"install_tools_check" json:"install_tools_check"`
	KbotSetupCheck                 bool `bson:"kbot_setup_check" json:"kbot_setup_check"`
	StateStoreCredsCheck           bool `bson:"state_store_creds_check" json:"state_store_creds_check"`
	StateStoreCreateCheck          bool `bson:"state_store_create_check" json:"state_store_create_check"`
	DomainLivenessCheck            bool `bson:"domain_liveness_check" json:"domain_liveness_check"`
	GitCredentialsCheck            bool `bson:"git_credentials_check" json:"git_credentials_check"`
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
	PostDetokenizeCheck            bool `bson:"post_detokenize_check" json:"post_detokenize_check"`
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
