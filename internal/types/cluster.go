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
	CloudProvider string `json:"cloud_provider" binding:"required,oneof=aws civo digitalocean k3d vultr"`
	CloudRegion   string `json:"cloud_region" binding:"required"`
	ClusterName   string `json:"cluster_name,omitempty"`
	DomainName    string `json:"domain_name" binding:"required"`
	GitProvider   string `json:"git_provider" binding:"required,oneof=github gitlab"`
	GitOwner      string `json:"git_owner" binding:"required"`
	GitToken      string `json:"git_token" binding:"required"`
	Type          string `json:"type" binding:"required,oneof=mgmt workload"`
}

// Cluster describes the configuration storage for a Kubefirst cluster object
type Cluster struct {
	ID                primitive.ObjectID `bson:"_id"`
	CreationTimestamp string             `bson:"creation_timestamp"`
	Status            string             `bson:"status"`
	InProgress        bool               `bson:"in_progress"`

	ClusterName   string `bson:"cluster_name"`
	CloudProvider string `bson:"cloud_provider"`
	CloudRegion   string `bson:"cloud_region"`
	DomainName    string `bson:"domain_name"`
	ClusterID     string `bson:"cluster_id"`
	ClusterType   string `bson:"cluster_type"`
	AlertsEmail   string `bson:"alerts_email"`

	CivoToken string `bson:"civo_token"`

	GitProvider        string `bson:"git_provider"`
	GitHost            string `bson:"git_host"`
	GitOwner           string `bson:"git_owner"`
	GitUser            string `bson:"git_user"`
	GitToken           string `bson:"git_token"`
	GitlabOwnerGroupID int    `bson:"gitlab_owner_group_id"`

	AtlantisWebhookSecret string `bson:"atlantis_webhook_secret"`
	AtlantisWebhookURL    string `bson:"atlantis_webhook_url"`
	KubefirstTeam         string `bson:"kubefirst_team"`

	StateStoreCredentials StateStoreCredentials `bson:"state_store_credentials,omitempty"`
	StateStoreDetails     StateStoreDetails     `bson:"state_store_details,omitempty"`

	PublicKey  string `bson:"public_key"`
	PrivateKey string `bson:"private_key"`
	PublicKeys string `bson:"public_keys"`

	ArgoCDUsername  string `bson:"argocd_username"`
	ArgoCDPassword  string `bson:"argocd_password"`
	ArgoCDAuthToken string `bson:"argocd_auth_token"`

	// kms
	AWSAccountId              string `bson:"aws_account_id,omitempty"`
	AWSKMSKeyId               string `bson:"aws_kms_key_id,omitempty"`
	AWSKMSKeyDetokenizedCheck bool   `bson:"aws_kms_key_detokenized_check"`

	// Telemetry
	UseTelemetry bool `bson:"use_telemetry"`

	// Checks
	GitInitCheck                   bool `bson:"git_init_check"`
	InstallToolsCheck              bool `bson:"install_tools_check"`
	KbotSetupCheck                 bool `bson:"kbot_setup_check"`
	StateStoreCredsCheck           bool `bson:"state_store_creds_check"`
	StateStoreCreateCheck          bool `bson:"state_store_create_check"`
	DomainLivenessCheck            bool `bson:"domain_liveness_check"`
	GitCredentialsCheck            bool `bson:"git_credentials_check"`
	GitopsReadyCheck               bool `bson:"gitops_ready_check"`
	GitTerraformApplyCheck         bool `bson:"git_terraform_apply_check"`
	GitopsPushedCheck              bool `bson:"gitops_pushed_check"`
	CloudTerraformApplyCheck       bool `bson:"cloud_terraform_apply_check"`
	CloudTerraformApplyFailedCheck bool `bson:"cloud_terraform_apply_failed_check"`
	ClusterSecretsCreatedCheck     bool `bson:"cluster_secrets_created_check"`
	ArgoCDInstallCheck             bool `bson:"argocd_install_check"`
	ArgoCDInitializeCheck          bool `bson:"argocd_initialize_check"`
	ArgoCDCreateRegistryCheck      bool `bson:"argocd_create_registry_check"`
	ArgoCDDeleteRegistryCheck      bool `bson:"argocd_delete_registry_check"`
	VaultInitializedCheck          bool `bson:"vault_initialized_check"`
	VaultTerraformApplyCheck       bool `bson:"vault_terraform_apply_check"`
	UsersTerraformApplyCheck       bool `bson:"users_terraform_apply_check"`
	PostDetokenizeCheck            bool `bson:"post_detokenize_check"`
}

// StateStoreCredentials
type StateStoreCredentials struct {
	AccessKeyID     string `bson:"access_key_id,omitempty"`
	SecretAccessKey string `bson:"secret_access_key,omitempty"`
	Name            string `bson:"name,omitempty"`
	ID              string `bson:"id,omitempty"`
}

// StateStoreDetails
type StateStoreDetails struct {
	Name                string `bson:"name,omitempty"`
	ID                  string `bson:"id,omitempty"`
	Hostname            string `bson:"hostname,omitempty"`
	AWSStateStoreBucket string `bson:"aws_state_store_bucket,omitempty"`
	AWSArtifactsBucket  string `bson:"aws_artifacts_bucket,omitempty"`
}

// PushBucketObject
type PushBucketObject struct {
	LocalFilePath  string
	RemoteFilePath string
	ContentType    string
}

// ImportClusterRequest
type ImportClusterRequest struct {
	ClusterName           string
	StateStoreCredentials StateStoreCredentials
	StateStoreDetails     StateStoreDetails
}
