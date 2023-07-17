/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/kubefirst/kubefirst-api/internal/constants"
	"github.com/kubefirst/kubefirst-api/internal/db"
	"github.com/kubefirst/kubefirst-api/internal/telemetryShim"
	"github.com/kubefirst/kubefirst-api/internal/types"
	"github.com/kubefirst/kubefirst-api/internal/utils"
	"github.com/kubefirst/runtime/pkg"
	awsinternal "github.com/kubefirst/runtime/pkg/aws"
	"github.com/kubefirst/runtime/pkg/github"
	"github.com/kubefirst/runtime/pkg/gitlab"
	"github.com/kubefirst/runtime/pkg/handlers"
	"github.com/kubefirst/runtime/pkg/providerConfigs"
	"github.com/kubefirst/runtime/pkg/segment"
	"github.com/kubefirst/runtime/pkg/services"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	gitopsTemplateVersion = "v2.2.0"
)

type ClusterController struct {
	CloudProvider string
	CloudRegion   string
	ClusterName   string
	ClusterID     string
	ClusterType   string
	DomainName    string
	AlertsEmail   string

	// auth
	AWSAuth            types.AWSAuth
	CivoAuth           types.CivoAuth
	DigitaloceanAuth   types.DigitaloceanAuth
	VultrAuth          types.VultrAuth
	AwsAccessKeyID     string
	AwsSecretAccessKey string

	// configs
	ProviderConfig providerConfigs.ProviderConfig

	// git
	GitopsTemplateURL    string
	GitopsTemplateBranch string
	GitProvider          string
	GitHost              string
	GitOwner             string
	GitUser              string
	GitToken             string
	GitlabOwnerGroupID   int

	// container registry
	ContainerRegistryHost string

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

	// keys
	// kbot public key
	PublicKey string
	// kbot private key
	PrivateKey string

	// Telemetry
	UseTelemetry bool

	// Database Controller
	MdbCl *db.MongoDBClient

	// Provider clients
	AwsClient *awsinternal.AWSConfiguration
}

// InitController
func (clctrl *ClusterController) InitController(def *types.ClusterDefinition) error {
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
		clusterID = pkg.GenerateClusterID()
	}

	// Telemetry handler
	segmentClient, err := telemetryShim.SetupTelemetry(rec)
	if err != nil {
		return err
	}
	defer segmentClient.Client.Close()

	telemetryShim.Transmit(rec.UseTelemetry, segmentClient, segment.MetricClusterInstallStarted, "")

	clctrl.AlertsEmail = def.AdminEmail
	clctrl.CloudProvider = def.CloudProvider
	clctrl.CloudRegion = def.CloudRegion
	clctrl.ClusterName = def.ClusterName
	clctrl.ClusterID = clusterID
	clctrl.DomainName = def.DomainName
	clctrl.ClusterType = def.Type
	clctrl.HttpClient = http.DefaultClient

	switch clctrl.CloudProvider {
	case "aws":
		clctrl.AWSAuth = def.AWSAuth
	case "civo":
		clctrl.CivoAuth = def.CivoAuth
	case "digitalocean":
		clctrl.DigitaloceanAuth = def.DigitaloceanAuth
	case "vultr":
		clctrl.VultrAuth = def.VultrAuth
	}

	clctrl.Repositories = []string{"gitops", "metaphor"}
	clctrl.Teams = []string{"admins", "developers"}

	if def.GitopsTemplateBranch != "" {
		clctrl.GitopsTemplateBranch = def.GitopsTemplateBranch
	} else {
		clctrl.GitopsTemplateBranch = gitopsTemplateVersion
	}

	if def.GitopsTemplateURL != "" {
		clctrl.GitopsTemplateURL = def.GitopsTemplateURL
	} else {
		clctrl.GitopsTemplateURL = "https://github.com/kubefirst/gitops-template.git"
	}

	clctrl.KubefirstStateStoreBucketName = fmt.Sprintf("k1-state-store-%s-%s", clctrl.ClusterName, clusterID)
	clctrl.KubefirstArtifactsBucketName = fmt.Sprintf("k1-artifacts-%s-%s", clctrl.ClusterName, clusterID)

	clctrl.KubefirstTeam = os.Getenv("KUBEFIRST_TEAM")
	if clctrl.KubefirstTeam == "" {
		clctrl.KubefirstTeam = "undefined"
	}
	clctrl.AtlantisWebhookSecret = pkg.Random(20)
	clctrl.AtlantisWebhookURL = fmt.Sprintf("https://atlantis.%s/events", clctrl.DomainName)

	// Initialize git parameters
	clctrl.GitProvider = def.GitProvider
	clctrl.GitToken = def.GitToken
	clctrl.GitOwner = def.GitOwner

	switch def.GitProvider {
	case "github":
		gitHubService := services.NewGitHubService(clctrl.HttpClient)
		gitHubHandler := handlers.NewGitHubHandler(gitHubService)

		clctrl.GitHost = "github.com"
		clctrl.ContainerRegistryHost = "ghcr.io"
		// Verify token scopes
		err := github.VerifyTokenPermissions(def.GitToken)
		if err != nil {
			return err
		}
		// Get authenticated user's name
		githubUser, err := gitHubHandler.GetGitHubUser(clctrl.GitToken)
		if err != nil {
			return err
		}
		clctrl.GitUser = githubUser
	case "gitlab":
		clctrl.GitHost = "gitlab.com"
		clctrl.ContainerRegistryHost = "registry.gitlab.com"
		// Verify token scopes
		err := gitlab.VerifyTokenPermissions(def.GitToken)
		if err != nil {
			return err
		}
		gitlabClient, err := gitlab.NewGitLabClient(def.GitToken, def.GitOwner)
		if err != nil {
			return err
		}
		clctrl.GitOwner = gitlabClient.ParentGroupPath
		clctrl.GitlabOwnerGroupID = gitlabClient.ParentGroupID
		// Get authenticated user's name
		user, _, err := gitlabClient.Client.Users.CurrentUser()
		if err != nil {
			return fmt.Errorf("unable to get authenticated user info - please make sure GITLAB_TOKEN env var is set %s", err.Error())
		}
		clctrl.GitUser = user.Username
	default:
		return fmt.Errorf("invalid git provider option")
	}

	// Instantiate provider configuration
	switch clctrl.CloudProvider {
	case "aws":
		clctrl.ProviderConfig = *providerConfigs.GetConfig(clctrl.ClusterName, clctrl.DomainName, clctrl.GitProvider, clctrl.GitOwner)
	case "civo":
		clctrl.ProviderConfig = *providerConfigs.GetConfig(clctrl.ClusterName, clctrl.DomainName, clctrl.GitProvider, clctrl.GitOwner)
		clctrl.ProviderConfig.CivoToken = clctrl.CivoAuth.Token
	case "digitalocean":
		clctrl.ProviderConfig = *providerConfigs.GetConfig(clctrl.ClusterName, clctrl.DomainName, clctrl.GitProvider, clctrl.GitOwner)
		clctrl.ProviderConfig.DigitaloceanToken = clctrl.DigitaloceanAuth.Token
	case "vultr":
		clctrl.ProviderConfig = *providerConfigs.GetConfig(clctrl.ClusterName, clctrl.DomainName, clctrl.GitProvider, clctrl.GitOwner)
		clctrl.ProviderConfig.VultrToken = clctrl.VultrAuth.Token
	}

	// Instantiate provider clients
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
	}

	if os.Getenv("USE_TELEMETRY") == "false" {
		clctrl.UseTelemetry = false
	} else {
		clctrl.UseTelemetry = true
	}

	// Write cluster record if it doesn't exist
	cl := types.Cluster{
		ID:                    primitive.NewObjectID(),
		CreationTimestamp:     fmt.Sprintf("%v", time.Now().UTC()),
		UseTelemetry:          clctrl.UseTelemetry,
		Status:                constants.ClusterStatusProvisioning,
		AlertsEmail:           clctrl.AlertsEmail,
		ClusterName:           clctrl.ClusterName,
		CloudProvider:         clctrl.CloudProvider,
		CloudRegion:           clctrl.CloudRegion,
		DomainName:            clctrl.DomainName,
		ClusterID:             clctrl.ClusterID,
		ClusterType:           clctrl.ClusterType,
		GitopsTemplateURL:     clctrl.GitopsTemplateURL,
		GitopsTemplateBranch:  clctrl.GitopsTemplateBranch,
		GitProvider:           clctrl.GitProvider,
		GitHost:               clctrl.GitHost,
		GitOwner:              clctrl.GitOwner,
		GitUser:               clctrl.GitUser,
		GitToken:              clctrl.GitToken,
		GitlabOwnerGroupID:    clctrl.GitlabOwnerGroupID,
		AtlantisWebhookSecret: clctrl.AtlantisWebhookSecret,
		AtlantisWebhookURL:    clctrl.AtlantisWebhookURL,
		KubefirstTeam:         clctrl.KubefirstTeam,
		AWSAuth:               clctrl.AWSAuth,
		CivoAuth:              clctrl.CivoAuth,
		DigitaloceanAuth:      clctrl.DigitaloceanAuth,
		VultrAuth:             clctrl.VultrAuth,
	}
	err = clctrl.MdbCl.InsertCluster(cl)
	if err != nil {
		return err
	}

	return nil
}

// GetCurrentClusterRecord will return an active cluster's record if it exists
func (clctrl *ClusterController) GetCurrentClusterRecord() (types.Cluster, error) {
	cl, err := clctrl.MdbCl.GetCluster(clctrl.ClusterName)
	if err != nil {
		return types.Cluster{}, err
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
