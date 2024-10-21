/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"time"

	runtime "github.com/konstructio/kubefirst-api/internal"
	awsinternal "github.com/konstructio/kubefirst-api/internal/aws"
	azureinternal "github.com/konstructio/kubefirst-api/internal/azure"
	"github.com/konstructio/kubefirst-api/internal/constants"
	"github.com/konstructio/kubefirst-api/internal/env"
	"github.com/konstructio/kubefirst-api/internal/github"
	"github.com/konstructio/kubefirst-api/internal/gitlab"
	"github.com/konstructio/kubefirst-api/internal/k8s"
	"github.com/konstructio/kubefirst-api/internal/secrets"
	"github.com/konstructio/kubefirst-api/internal/services"
	"github.com/konstructio/kubefirst-api/internal/utils"
	google "github.com/konstructio/kubefirst-api/pkg/google"
	"github.com/konstructio/kubefirst-api/pkg/handlers"
	"github.com/konstructio/kubefirst-api/pkg/providerConfigs"
	"github.com/konstructio/kubefirst-api/pkg/types"
	"github.com/kubefirst/metrics-client/pkg/telemetry"
	log "github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"k8s.io/client-go/kubernetes"
)

type ClusterController struct {
	CloudProvider             string
	CloudRegion               string
	ClusterName               string
	ClusterID                 string
	ClusterType               string
	DomainName                string
	SubdomainName             string
	DNSProvider               string
	UseCloudflareOriginIssuer bool
	AlertsEmail               string

	// auth
	AkamaiAuth             types.AkamaiAuth
	AWSAuth                types.AWSAuth
	AzureAuth              types.AzureAuth
	CivoAuth               types.CivoAuth
	DigitaloceanAuth       types.DigitaloceanAuth
	VultrAuth              types.VultrAuth
	CloudflareAuth         types.CloudflareAuth
	GitAuth                types.GitAuth
	VaultAuth              types.VaultAuth
	GoogleAuth             types.GoogleAuth
	K3sAuth                types.K3sAuth
	AwsAccessKeyID         string
	AwsSecretAccessKey     string
	NodeType               string
	NodeCount              int
	PostInstallCatalogApps []types.GitopsCatalogApp
	InstallKubefirstPro    bool

	// configs
	ProviderConfig providerConfigs.ProviderConfig

	// git
	GitopsTemplateURL    string
	GitopsTemplateBranch string
	GitProvider          string
	GitProtocol          string
	GitHost              string
	GitOwner             string
	GitUser              string
	GitToken             string
	GitlabOwnerGroupID   int

	// container registry
	ContainerRegistryHost string
	ECR                   bool

	// http
	Client *http.Client

	// repositories
	Repositories []string

	// teams
	Teams []string

	// atlantis
	AtlantisWebhookSecret string
	AtlantisWebhookURL    string

	// internal
	KubefirstTeam string

	// state store
	KubefirstStateStoreBucketName string
	KubefirstArtifactsBucketName  string

	KubernetesClient kubernetes.Interface

	// Telemetry
	TelemetryEvent telemetry.TelemetryEvent

	// Azure
	AzureDNSZoneResourceGroup string

	// Provider clients
	AwsClient    *awsinternal.Configuration
	AzureClient  *azureinternal.Client
	GoogleClient google.Configuration
	Kcfg         *k8s.KubernetesClient
	Cluster      types.Cluster
}

// InitController
func (clctrl *ClusterController) InitController(def *types.ClusterDefinition) error {
	// Create k1 dir if it doesn't exist
	utils.CreateK1Directory(def.ClusterName)

	// Get Environment variables
	env, _ := env.GetEnv(constants.SilenceGetEnv)

	kcfg := utils.GetKubernetesClient(def.ClusterName)
	clctrl.KubernetesClient = kcfg.Clientset

	// If on local environment, automatically create the namespace
	// if it doesn't exist
	if env.K1LocalDebug {
		if err := utils.CreateKubefirstNamespaceIfNotExists(clctrl.KubernetesClient); err != nil {
			return fmt.Errorf("error creating Kubefirst namespace: %w", err)
		}
	}

	// Determine if record already exists
	recordExists := true

	// Get cluster record if it exists
	rec, err := secrets.GetCluster(clctrl.KubernetesClient, def.ClusterName)
	if err != nil && !errors.Is(err, &secrets.ClusterNotFoundError{}) {
		return fmt.Errorf("could not read cluster secret %s: %w", def.ClusterName, err)
	}

	if rec == nil {
		recordExists = false
		log.Info().Msg("cluster record doesn't exist, continuing")
	}

	logFileName := def.LogFileName
	if recordExists {
		logFileName = rec.LogFileName
	}

	utils.InitializeLogs(logFileName)

	// If record exists but status is deleted, entry should be deleted
	// and process should start fresh
	if recordExists && rec.Status == constants.ClusterStatusDeleted {
		err = secrets.DeleteCluster(clctrl.KubernetesClient, def.ClusterName)
		if err != nil {
			return fmt.Errorf("error deleting existing cluster %q: %w", def.ClusterName, err)
		}
	}

	var clusterID string
	if recordExists {
		clusterID = rec.ClusterID
	} else {
		clusterID = runtime.GenerateClusterID()
	}

	telemetryEvent := telemetry.TelemetryEvent{
		CliVersion:        env.KubefirstVersion,
		CloudProvider:     env.CloudProvider,
		ClusterID:         env.ClusterID,
		ClusterType:       env.ClusterType,
		DomainName:        env.DomainName,
		ErrorMessage:      "",
		GitProvider:       env.GitProvider,
		InstallMethod:     env.InstallMethod,
		KubefirstClient:   "api",
		KubefirstTeam:     env.KubefirstTeam,
		KubefirstTeamInfo: env.KubefirstTeamInfo,
		MachineID:         env.ClusterID,
		ParentClusterId:   env.ParentClusterID,
		MetricName:        telemetry.ClusterInstallCompleted,
		UserId:            env.ClusterID,
	}
	clctrl.TelemetryEvent = telemetryEvent

	// Copy Cluster Definiion to Cluster Controller
	clctrl.AlertsEmail = def.AdminEmail
	clctrl.CloudProvider = def.CloudProvider
	clctrl.CloudRegion = def.CloudRegion
	clctrl.ClusterName = def.ClusterName
	clctrl.ClusterID = clusterID
	clctrl.DomainName = def.DomainName
	clctrl.SubdomainName = def.SubdomainName
	clctrl.DNSProvider = def.DNSProvider
	clctrl.ClusterType = def.Type
	clctrl.Client = http.DefaultClient
	clctrl.NodeType = def.NodeType
	clctrl.NodeCount = def.NodeCount
	clctrl.PostInstallCatalogApps = def.PostInstallCatalogApps
	clctrl.InstallKubefirstPro = def.InstallKubefirstPro

	clctrl.AkamaiAuth = def.AkamaiAuth
	clctrl.AWSAuth = def.AWSAuth
	clctrl.AzureAuth = def.AzureAuth
	clctrl.CivoAuth = def.CivoAuth
	clctrl.DigitaloceanAuth = def.DigitaloceanAuth
	clctrl.VultrAuth = def.VultrAuth
	clctrl.GoogleAuth = def.GoogleAuth
	clctrl.K3sAuth = def.K3sAuth
	clctrl.CloudflareAuth = def.CloudflareAuth

	clctrl.Repositories = []string{"gitops", "metaphor"}
	clctrl.Teams = []string{"admins", "developers"}

	clctrl.ECR = def.ECR

	if def.GitopsTemplateBranch != "" {
		clctrl.GitopsTemplateBranch = def.GitopsTemplateBranch
	} else {
		clctrl.GitopsTemplateBranch = env.KubefirstVersion
	}

	if def.GitopsTemplateURL != "" {
		if def.GitopsTemplateBranch != "" {
			clctrl.GitopsTemplateURL = def.GitopsTemplateURL
		} else {
			return fmt.Errorf("invalid GitOps template configuration: must supply branch when supplying a GitOps template URL")
		}
	} else {
		clctrl.GitopsTemplateURL = "https://github.com/kubefirst/gitops-template.git"
	}
	switch def.CloudProvider {
	case "akamai":
		clctrl.KubefirstStateStoreBucketName = clctrl.ClusterName
	case "azure":
		// Azure storage accounts are 3-24 characters and only letters/numbers
		maxLen := 24
		reg := regexp.MustCompile(`\W`)

		storeName := fmt.Sprintf("k1%s%s", clusterID, clctrl.ClusterName)

		clctrl.KubefirstStateStoreBucketName = reg.ReplaceAllString(storeName, "")

		if len(clctrl.KubefirstStateStoreBucketName) > maxLen {
			clctrl.KubefirstStateStoreBucketName = clctrl.KubefirstStateStoreBucketName[:maxLen]
		}
	default:
		clctrl.KubefirstStateStoreBucketName = fmt.Sprintf("k1-state-store-%s-%s", clctrl.ClusterName, clusterID)
	}

	clctrl.KubefirstArtifactsBucketName = fmt.Sprintf("k1-artifacts-%s-%s", clctrl.ClusterName, clusterID)
	clctrl.NodeType = def.NodeType
	clctrl.NodeCount = def.NodeCount

	clctrl.KubefirstTeam = env.KubefirstTeam

	clctrl.AtlantisWebhookSecret = runtime.Random(20)

	var fullDomainName string
	if clctrl.SubdomainName != "" {
		fullDomainName = fmt.Sprintf("%s.%s", clctrl.SubdomainName, clctrl.DomainName)
	} else {
		fullDomainName = clctrl.DomainName
	}
	clctrl.AtlantisWebhookURL = fmt.Sprintf("https://atlantis.%s/events", fullDomainName)

	// Initialize git parameters
	clctrl.GitProvider = def.GitProvider
	clctrl.GitProtocol = def.GitProtocol
	clctrl.GitAuth = def.GitAuth

	err = clctrl.SetGitTokens(*def)
	if err != nil {
		return fmt.Errorf("failed to set Git tokens: %w", err)
	}

	// Instantiate provider configuration
	switch clctrl.CloudProvider {
	case "akamai":
		conf, err := providerConfigs.GetConfig(clctrl.ClusterName, clctrl.DomainName, clctrl.GitProvider, clctrl.GitAuth.Owner, clctrl.GitProtocol, clctrl.CloudflareAuth.APIToken, clctrl.CloudflareAuth.OriginCaIssuerKey)
		if err != nil {
			return fmt.Errorf("unable to get provider configuration for Akamai: %w", err)
		}

		clctrl.ProviderConfig = *conf
		clctrl.ProviderConfig.AkamaiToken = clctrl.AkamaiAuth.Token
	case "aws":
		conf, err := providerConfigs.GetConfig(clctrl.ClusterName, clctrl.DomainName, clctrl.GitProvider, clctrl.GitAuth.Owner, clctrl.GitProtocol, clctrl.CloudflareAuth.Token, "")
		if err != nil {
			return fmt.Errorf("unable to get provider configuration for AWS: %w", err)
		}

		clctrl.ProviderConfig = *conf
	case "azure":
		conf, err := providerConfigs.GetConfig(clctrl.ClusterName, clctrl.DomainName, clctrl.GitProvider, clctrl.GitAuth.Owner, clctrl.GitProtocol, clctrl.CloudflareAuth.APIToken, clctrl.CloudflareAuth.OriginCaIssuerKey)
		if err != nil {
			return fmt.Errorf("unable to get provider configuration for AWS: %w", err)
		}

		clctrl.ProviderConfig = *conf
		clctrl.AzureDNSZoneResourceGroup = def.AzureDNSZoneResourceGroup
	case "civo":
		conf, err := providerConfigs.GetConfig(clctrl.ClusterName, clctrl.DomainName, clctrl.GitProvider, clctrl.GitAuth.Owner, clctrl.GitProtocol, clctrl.CloudflareAuth.APIToken, clctrl.CloudflareAuth.OriginCaIssuerKey)
		if err != nil {
			return fmt.Errorf("unable to get provider configuration for Civo: %w", err)
		}

		clctrl.ProviderConfig = *conf
		clctrl.ProviderConfig.CivoToken = clctrl.CivoAuth.Token
	case "google":
		conf, err := providerConfigs.GetConfig(clctrl.ClusterName, clctrl.DomainName, clctrl.GitProvider, clctrl.GitAuth.Owner, clctrl.GitProtocol, clctrl.CloudflareAuth.Token, "")
		if err != nil {
			return fmt.Errorf("unable to get provider configuration for Google: %w", err)
		}

		clctrl.ProviderConfig = *conf
		clctrl.ProviderConfig.GoogleAuth = clctrl.GoogleAuth.KeyFile
		clctrl.ProviderConfig.GoogleProject = clctrl.GoogleAuth.ProjectID
	case "digitalocean":
		conf, err := providerConfigs.GetConfig(clctrl.ClusterName, clctrl.DomainName, clctrl.GitProvider, clctrl.GitAuth.Owner, clctrl.GitProtocol, clctrl.CloudflareAuth.Token, "")
		if err != nil {
			return fmt.Errorf("unable to get provider configuration for DigitalOcean: %w", err)
		}

		clctrl.ProviderConfig = *conf
		clctrl.ProviderConfig.DigitaloceanToken = clctrl.DigitaloceanAuth.Token
	case "vultr":
		conf, err := providerConfigs.GetConfig(clctrl.ClusterName, clctrl.DomainName, clctrl.GitProvider, clctrl.GitAuth.Owner, clctrl.GitProtocol, clctrl.CloudflareAuth.Token, "")
		if err != nil {
			return fmt.Errorf("unable to get provider configuration for Vultr: %w", err)
		}

		clctrl.ProviderConfig = *conf
		clctrl.ProviderConfig.VultrToken = clctrl.VultrAuth.Token
	case "k3s":
		conf, err := providerConfigs.GetConfig(clctrl.ClusterName, clctrl.DomainName, clctrl.GitProvider, clctrl.GitAuth.Owner, clctrl.GitProtocol, clctrl.CloudflareAuth.Token, "")
		if err != nil {
			return fmt.Errorf("unable to get provider configuration for K3s: %w", err)
		}

		clctrl.ProviderConfig = *conf
		clctrl.ProviderConfig.K3sServersPrivateIps = clctrl.K3sAuth.K3sServersPrivateIps
		clctrl.ProviderConfig.K3sServersPublicIps = clctrl.K3sAuth.K3sServersPublicIps
		clctrl.ProviderConfig.K3sSSHPrivateKey = clctrl.K3sAuth.K3sSSHPrivateKey
		clctrl.ProviderConfig.K3sSSHUser = clctrl.K3sAuth.K3sSSHUser
		clctrl.ProviderConfig.K3sServersArgs = clctrl.K3sAuth.K3sServersArgs
	}

	// Instantiate provider clients and copy cluster controller to cluster type
	switch clctrl.CloudProvider {
	case "aws":
		conf, err := awsinternal.NewAwsV3(
			clctrl.CloudRegion,
			clctrl.AWSAuth.AccessKeyID,
			clctrl.AWSAuth.SecretAccessKey,
			clctrl.AWSAuth.SessionToken,
		)
		if err != nil {
			return fmt.Errorf("unable to create AWS client: %w", err)
		}

		clctrl.AwsClient = &awsinternal.Configuration{Config: conf}
	case "azure":
		azureClient, err := azureinternal.NewClient(
			clctrl.AzureAuth.ClientID,
			clctrl.AzureAuth.ClientSecret,
			clctrl.AzureAuth.SubscriptionID,
			clctrl.AzureAuth.TenantID,
		)
		if err != nil {
			return fmt.Errorf("error creating azure client: %w", err)
		}
		clctrl.AzureClient = azureClient
	case "google":
		clctrl.GoogleClient = google.Configuration{
			Context: context.Background(),
			Project: def.GoogleAuth.ProjectID,
			Region:  clctrl.CloudRegion,
		}
	}

	// Write cluster record if it doesn't exist
	clctrl.Cluster = types.Cluster{
		ID:                     primitive.NewObjectID(),
		CreationTimestamp:      fmt.Sprintf("%v", primitive.NewDateTimeFromTime(time.Now().UTC())),
		Status:                 constants.ClusterStatusProvisioning,
		AlertsEmail:            clctrl.AlertsEmail,
		ClusterName:            clctrl.ClusterName,
		CloudProvider:          clctrl.CloudProvider,
		CloudRegion:            clctrl.CloudRegion,
		DomainName:             clctrl.DomainName,
		SubdomainName:          clctrl.SubdomainName,
		DNSProvider:            clctrl.DNSProvider,
		ClusterID:              clctrl.ClusterID,
		ECR:                    clctrl.ECR,
		ClusterType:            clctrl.ClusterType,
		GitopsTemplateURL:      clctrl.GitopsTemplateURL,
		GitopsTemplateBranch:   clctrl.GitopsTemplateBranch,
		GitProvider:            clctrl.GitProvider,
		GitProtocol:            clctrl.GitProtocol,
		GitHost:                clctrl.GitHost,
		GitAuth:                clctrl.GitAuth,
		GitlabOwnerGroupID:     clctrl.GitlabOwnerGroupID,
		AtlantisWebhookSecret:  clctrl.AtlantisWebhookSecret,
		AtlantisWebhookURL:     clctrl.AtlantisWebhookURL,
		KubefirstTeam:          clctrl.KubefirstTeam,
		AkamaiAuth:             clctrl.AkamaiAuth,
		AWSAuth:                clctrl.AWSAuth,
		AzureAuth:              clctrl.AzureAuth,
		CivoAuth:               clctrl.CivoAuth,
		GoogleAuth:             clctrl.GoogleAuth,
		DigitaloceanAuth:       clctrl.DigitaloceanAuth,
		VultrAuth:              clctrl.VultrAuth,
		K3sAuth:                clctrl.K3sAuth,
		CloudflareAuth:         clctrl.CloudflareAuth,
		NodeType:               clctrl.NodeType,
		NodeCount:              clctrl.NodeCount,
		LogFileName:            def.LogFileName,
		PostInstallCatalogApps: clctrl.PostInstallCatalogApps,
	}

	if !recordExists {
		log.Info().Msg("cluster record doesn't exist after initialization, inserting")
		err = secrets.InsertCluster(clctrl.KubernetesClient, clctrl.Cluster)
		if err != nil {
			return fmt.Errorf("error inserting cluster record: %w", err)
		}
	} else {
		clctrl.Cluster = *rec
	}

	return nil
}

// GetCurrentClusterRecord will return an active cluster's record if it exists
func (clctrl *ClusterController) SetGitTokens(def types.ClusterDefinition) error {
	switch def.GitProvider {
	case "github":
		gitHubService := services.NewGitHubService(clctrl.Client)
		gitHubHandler := handlers.NewGitHubHandler(gitHubService)

		clctrl.GitHost = "github.com"
		clctrl.ContainerRegistryHost = "ghcr.io"
		// Verify token scopes
		err := github.VerifyTokenPermissions(def.GitAuth.Token)
		if err != nil {
			return fmt.Errorf("GitHub token verification failed: %w", err)
		}
		// Get authenticated user's name
		githubUser, err := gitHubHandler.GetGitHubUser(def.GitAuth.Token)
		if err != nil {
			return fmt.Errorf("error retrieving GitHub user: %w", err)
		}
		clctrl.GitAuth.User = githubUser
	case "gitlab":
		clctrl.GitHost = "gitlab.com"
		clctrl.ContainerRegistryHost = "registry.gitlab.com"
		// Verify token scopes
		err := gitlab.VerifyTokenPermissions(def.GitAuth.Token)
		if err != nil {
			return fmt.Errorf("GitLab token verification failed: %w", err)
		}
		gitlabClient, err := gitlab.NewGitLabClient(def.GitAuth.Token, def.GitAuth.Owner)
		if err != nil {
			return fmt.Errorf("error creating GitLab client: %w", err)
		}
		clctrl.GitAuth.Owner = gitlabClient.ParentGroupPath
		clctrl.GitlabOwnerGroupID = gitlabClient.ParentGroupID
		// Get authenticated user's name
		user, _, err := gitlabClient.Client.Users.CurrentUser()
		if err != nil {
			return fmt.Errorf("unable to get authenticated user info from GitLab: %w", err)
		}
		clctrl.GitAuth.User = user.Username
	default:
		return fmt.Errorf("invalid git provider option: %q", def.GitProvider)
	}

	return nil
}

// GetCurrentClusterRecord will return an active cluster's record if it exists
func (clctrl *ClusterController) GetCurrentClusterRecord() (*types.Cluster, error) {
	cl, err := secrets.GetCluster(clctrl.KubernetesClient, clctrl.ClusterName)
	if err != nil {
		return nil, fmt.Errorf("error retrieving current cluster record: %w", err)
	}

	return cl, nil
}

// UpdateClusterOnError implements an error handler for cluster controller objects
func (clctrl *ClusterController) UpdateClusterOnError(condition string) error {
	clctrl.Cluster.InProgress = false
	clctrl.Cluster.Status = constants.ClusterStatusError
	clctrl.Cluster.LastCondition = condition

	log.Error().Msgf("unexpected error: %s", condition)
	if err := secrets.UpdateCluster(clctrl.KubernetesClient, clctrl.Cluster); err != nil {
		return fmt.Errorf("error updating cluster after condition failure: %w", err)
	}

	return nil
}
