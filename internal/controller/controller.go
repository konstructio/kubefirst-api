/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/kubefirst/kubefirst-api/internal/constants"
	"github.com/kubefirst/kubefirst-api/internal/db"
	"github.com/kubefirst/kubefirst-api/internal/env"
	"github.com/kubefirst/kubefirst-api/internal/utils"
	google "github.com/kubefirst/kubefirst-api/pkg/google"
	"github.com/kubefirst/kubefirst-api/pkg/handlers"
	"github.com/kubefirst/kubefirst-api/pkg/providerConfigs"
	pkgtypes "github.com/kubefirst/kubefirst-api/pkg/types"
	"github.com/kubefirst/metrics-client/pkg/telemetry"
	runtime "github.com/kubefirst/runtime/pkg"
	awsinternal "github.com/kubefirst/runtime/pkg/aws"
	"github.com/kubefirst/runtime/pkg/github"
	"github.com/kubefirst/runtime/pkg/gitlab"
	"github.com/kubefirst/runtime/pkg/k8s"
	"github.com/kubefirst/runtime/pkg/services"

	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	gitopsTemplateVersion = "v2.3.5"
)

type ClusterController struct {
	CloudProvider             string
	CloudRegion               string
	ClusterName               string
	ClusterID                 string
	ClusterType               string
	DomainName                string
	SubdomainName             string
	DnsProvider               string
	UseCloudflareOriginIssuer bool
	AlertsEmail               string

	// auth
	AWSAuth            pkgtypes.AWSAuth
	CivoAuth           pkgtypes.CivoAuth
	DigitaloceanAuth   pkgtypes.DigitaloceanAuth
	VultrAuth          pkgtypes.VultrAuth
	CloudflareAuth     pkgtypes.CloudflareAuth
	GitAuth            pkgtypes.GitAuth
	VaultAuth          pkgtypes.VaultAuth
	GoogleAuth         pkgtypes.GoogleAuth
	AwsAccessKeyID     string
	AwsSecretAccessKey string

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
	HttpClient *http.Client

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

	// Database Controller
	MdbCl *db.MongoDBClient

	// Telemetry
	TelemetryEvent telemetry.TelemetryEvent

	// Provider clients
	AwsClient    *awsinternal.AWSConfiguration
	GoogleClient google.GoogleConfiguration
	Kcfg         *k8s.KubernetesClient
}

// InitController
func (clctrl *ClusterController) InitController(def *pkgtypes.ClusterDefinition) error {
	// Create k1 dir if it doesn't exist
	utils.CreateK1Directory(def.ClusterName)

	// Database controller
	clctrl.MdbCl = db.Client

	// Determine if record already exists
	recordExists := true
	rec, err := clctrl.MdbCl.GetCluster(def.ClusterName)
	if err != nil {
		recordExists = false
		log.Info("cluster record doesn't exist, continuing")
	}

	// If record exists but status is deleted, entry should be deleted
	// and process should start fresh
	if recordExists && rec.Status == constants.ClusterStatusDeleted {
		err = clctrl.MdbCl.DeleteCluster(def.ClusterName)
		if err != nil {
			return fmt.Errorf("could not delete existing cluster %s: %s", def.ClusterName, err)
		}
	}

	var clusterID string
	if recordExists {
		clusterID = rec.ClusterID
	} else {
		clusterID = runtime.GenerateClusterID()
	}

	env, _ := env.GetEnv()

	telemetryEvent := telemetry.TelemetryEvent{
		CliVersion:        env.KubefirstVersion,
		CloudProvider:     env.CloudProvider,
		ClusterID:         env.ClusterId,
		ClusterType:       env.ClusterType,
		DomainName:        env.DomainName,
		ErrorMessage:      "",
		GitProvider:       env.GitProvider,
		InstallMethod:     env.InstallMethod,
		KubefirstClient:   "api",
		KubefirstTeam:     env.KubefirstTeam,
		KubefirstTeamInfo: env.KubefirstTeamInfo,
		MachineID:         env.ClusterId,
		MetricName:        telemetry.ClusterInstallStarted,
		UserId:            env.ClusterId,
	}
	clctrl.TelemetryEvent = telemetryEvent

	telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.ClusterInstallStarted, "")

	//Copy Cluster Definiion to Cluster Controller
	clctrl.AlertsEmail = def.AdminEmail
	clctrl.CloudProvider = def.CloudProvider
	clctrl.CloudRegion = def.CloudRegion
	clctrl.ClusterName = def.ClusterName
	clctrl.ClusterID = clusterID
	clctrl.DomainName = def.DomainName
	clctrl.SubdomainName = def.SubdomainName
	clctrl.DnsProvider = def.DnsProvider
	clctrl.ClusterType = def.Type
	clctrl.HttpClient = http.DefaultClient

	clctrl.AWSAuth = def.AWSAuth
	clctrl.CivoAuth = def.CivoAuth
	clctrl.DigitaloceanAuth = def.DigitaloceanAuth
	clctrl.VultrAuth = def.VultrAuth
	clctrl.GoogleAuth = def.GoogleAuth
	clctrl.CloudflareAuth = def.CloudflareAuth

	clctrl.Repositories = []string{"gitops", "metaphor"}
	clctrl.Teams = []string{"admins", "developers"}

	clctrl.ECR = def.ECR

	if def.GitopsTemplateBranch != "" {
		clctrl.GitopsTemplateBranch = def.GitopsTemplateBranch
	} else {
		clctrl.GitopsTemplateBranch = gitopsTemplateVersion
	}

	if def.GitopsTemplateURL != "" {
		if def.GitopsTemplateBranch != "" {
			clctrl.GitopsTemplateURL = def.GitopsTemplateURL
		} else {
			return fmt.Errorf("must supply branch of gitops template repo when supplying a gitops template url")
		}
	} else {
		clctrl.GitopsTemplateURL = "https://github.com/kubefirst/gitops-template.git"
	}

	clctrl.KubefirstStateStoreBucketName = fmt.Sprintf("k1-state-store-%s-%s", clctrl.ClusterName, clusterID)
	clctrl.KubefirstArtifactsBucketName = fmt.Sprintf("k1-artifacts-%s-%s", clctrl.ClusterName, clusterID)

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
		return err
	}

	// Instantiate provider configuration
	switch clctrl.CloudProvider {
	case "aws":
		clctrl.ProviderConfig = *providerConfigs.GetConfig(clctrl.ClusterName, clctrl.DomainName, clctrl.GitProvider, clctrl.GitAuth.Owner, clctrl.GitProtocol, clctrl.CloudflareAuth.Token, "")
	case "civo":
		clctrl.ProviderConfig = *providerConfigs.GetConfig(clctrl.ClusterName, clctrl.DomainName, clctrl.GitProvider, clctrl.GitAuth.Owner, clctrl.GitProtocol, clctrl.CloudflareAuth.APIToken, clctrl.CloudflareAuth.OriginCaIssuerKey)
		clctrl.ProviderConfig.CivoToken = clctrl.CivoAuth.Token
	case "google":
		clctrl.ProviderConfig = *providerConfigs.GetConfig(clctrl.ClusterName, clctrl.DomainName, clctrl.GitProvider, clctrl.GitAuth.Owner, clctrl.GitProtocol, clctrl.CloudflareAuth.Token, "")
		clctrl.ProviderConfig.GoogleAuth = clctrl.GoogleAuth.KeyFile
		clctrl.ProviderConfig.GoogleProject = clctrl.GoogleAuth.ProjectId
	case "digitalocean":
		clctrl.ProviderConfig = *providerConfigs.GetConfig(clctrl.ClusterName, clctrl.DomainName, clctrl.GitProvider, clctrl.GitAuth.Owner, clctrl.GitProtocol, clctrl.CloudflareAuth.Token, "")
		clctrl.ProviderConfig.DigitaloceanToken = clctrl.DigitaloceanAuth.Token
	case "vultr":
		clctrl.ProviderConfig = *providerConfigs.GetConfig(clctrl.ClusterName, clctrl.DomainName, clctrl.GitProvider, clctrl.GitAuth.Owner, clctrl.GitProtocol, clctrl.CloudflareAuth.Token, "")
		clctrl.ProviderConfig.VultrToken = clctrl.VultrAuth.Token
	}

	// Instantiate provider clients and copy cluster controller to cluster type
	switch clctrl.CloudProvider {
	case "aws":
		clctrl.AwsClient = &awsinternal.AWSConfiguration{
			Config: awsinternal.NewAwsV3(
				clctrl.CloudRegion,
				clctrl.AWSAuth.AccessKeyID,
				clctrl.AWSAuth.SecretAccessKey,
				clctrl.AWSAuth.SessionToken,
			),
		}
	case "google":
		clctrl.GoogleClient = google.GoogleConfiguration{
			Context: context.Background(),
			Project: def.GoogleAuth.ProjectId,
			Region:  clctrl.CloudRegion,
		}

	}

	// Write cluster record if it doesn't exist
	cl := pkgtypes.Cluster{
		ID:                    primitive.NewObjectID(),
		CreationTimestamp:     fmt.Sprintf("%v", primitive.NewDateTimeFromTime(time.Now().UTC())),
		Status:                constants.ClusterStatusProvisioning,
		AlertsEmail:           clctrl.AlertsEmail,
		ClusterName:           clctrl.ClusterName,
		CloudProvider:         clctrl.CloudProvider,
		CloudRegion:           clctrl.CloudRegion,
		DomainName:            clctrl.DomainName,
		SubdomainName:         clctrl.SubdomainName,
		DnsProvider:           clctrl.DnsProvider,
		ClusterID:             clctrl.ClusterID,
		ECR:                   clctrl.ECR,
		ClusterType:           clctrl.ClusterType,
		GitopsTemplateURL:     clctrl.GitopsTemplateURL,
		GitopsTemplateBranch:  clctrl.GitopsTemplateBranch,
		GitProvider:           clctrl.GitProvider,
		GitProtocol:           clctrl.GitProtocol,
		GitHost:               clctrl.GitHost,
		GitAuth:               clctrl.GitAuth,
		GitlabOwnerGroupID:    clctrl.GitlabOwnerGroupID,
		AtlantisWebhookSecret: clctrl.AtlantisWebhookSecret,
		AtlantisWebhookURL:    clctrl.AtlantisWebhookURL,
		KubefirstTeam:         clctrl.KubefirstTeam,
		AWSAuth:               clctrl.AWSAuth,
		CivoAuth:              clctrl.CivoAuth,
		GoogleAuth:            clctrl.GoogleAuth,
		DigitaloceanAuth:      clctrl.DigitaloceanAuth,
		VultrAuth:             clctrl.VultrAuth,
		CloudflareAuth:        clctrl.CloudflareAuth,
	}
	err = clctrl.MdbCl.InsertCluster(cl)
	if err != nil {
		return err
	}

	return nil
}

// GetCurrentClusterRecord will return an active cluster's record if it exists
func (clctrl *ClusterController) SetGitTokens(def pkgtypes.ClusterDefinition) error {
	switch def.GitProvider {
	case "github":
		gitHubService := services.NewGitHubService(clctrl.HttpClient)
		gitHubHandler := handlers.NewGitHubHandler(gitHubService)

		clctrl.GitHost = "github.com"
		clctrl.ContainerRegistryHost = "ghcr.io"
		// Verify token scopes
		err := github.VerifyTokenPermissions(def.GitAuth.Token)
		if err != nil {
			return err
		}
		// Get authenticated user's name
		githubUser, err := gitHubHandler.GetGitHubUser(def.GitAuth.Token)
		if err != nil {
			return err
		}
		clctrl.GitAuth.User = githubUser
	case "gitlab":
		clctrl.GitHost = "gitlab.com"
		clctrl.ContainerRegistryHost = "registry.gitlab.com"
		// Verify token scopes
		err := gitlab.VerifyTokenPermissions(def.GitAuth.Token)
		if err != nil {
			return err
		}
		gitlabClient, err := gitlab.NewGitLabClient(def.GitAuth.Token, def.GitAuth.Owner)
		if err != nil {
			return err
		}
		clctrl.GitAuth.Owner = gitlabClient.ParentGroupPath
		clctrl.GitlabOwnerGroupID = gitlabClient.ParentGroupID
		// Get authenticated user's name
		user, _, err := gitlabClient.Client.Users.CurrentUser()
		if err != nil {
			return fmt.Errorf("unable to get authenticated user info - please make sure GITLAB_TOKEN env var is set %s", err.Error())
		}
		clctrl.GitAuth.User = user.Username
	default:
		return fmt.Errorf("invalid git provider option")
	}

	return nil
}

// GetCurrentClusterRecord will return an active cluster's record if it exists
func (clctrl *ClusterController) GetCurrentClusterRecord() (pkgtypes.Cluster, error) {
	cl, err := clctrl.MdbCl.GetCluster(clctrl.ClusterName)
	if err != nil {
		return pkgtypes.Cluster{}, err
	}

	return cl, nil
}

// HandleError implements an error handler for cluster controller objects
func (clctrl *ClusterController) HandleError(condition string) error {
	err := clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "in_progress", false)
	if err != nil {
		return err
	}
	err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "status", constants.ClusterStatusError)
	if err != nil {
		return err
	}
	err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "last_condition", condition)
	if err != nil {
		return err
	}

	return nil
}
