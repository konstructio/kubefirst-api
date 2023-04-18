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

	"github.com/kubefirst/kubefirst-api/internal/db"
	"github.com/kubefirst/kubefirst-api/internal/types"
	"github.com/kubefirst/runtime/pkg"
	"github.com/kubefirst/runtime/pkg/civo"
	"github.com/kubefirst/runtime/pkg/github"
	"github.com/kubefirst/runtime/pkg/gitlab"
	"github.com/kubefirst/runtime/pkg/handlers"
	"github.com/kubefirst/runtime/pkg/k3d"
	"github.com/kubefirst/runtime/pkg/services"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ClusterController struct {
	CloudProvider string
	CloudRegion   string
	ClusterName   string
	ClusterID     string
	ClusterType   string
	DomainName    string
	AlertsEmail   string

	// tokens
	CivoToken string

	// configs
	ProviderConfig interface{}

	// git
	GitProvider        string
	GitHost            string
	GitOwner           string
	GitUser            string
	GitToken           string
	GitlabOwnerGroupID int

	// container registry
	ContainerRegistryHost string

	// http
	HttpClient *http.Client

	// repositories
	Repositories []string

	// teams
	Teams []string

	// other
	AtlantisWebhookSecret         string
	AtlantisWebhookURL            string
	KubefirstTeam                 string
	GitopsTemplateBranchFlag      string
	GitopsTemplateURLFlag         string
	KubefirstStateStoreBucketName string

	// keys
	// kbot public key
	PublicKey string
	// kbot private key
	PrivateKey string

	// Database Controller
	MdbCl *db.MongoDBClient
}

// InitController
func (clctrl *ClusterController) InitController(def *types.ClusterDefinition) error {
	// Database controller
	clctrl.MdbCl = &db.MongoDBClient{}
	err := clctrl.MdbCl.InitDatabase()
	if err != nil {
		return err
	}

	clctrl.AlertsEmail = def.AdminEmail
	clctrl.CloudProvider = def.CloudProvider
	clctrl.CloudRegion = def.CloudRegion
	clctrl.ClusterName = def.ClusterName
	clctrl.ClusterID = pkg.GenerateClusterID()
	clctrl.DomainName = def.DomainName
	clctrl.ClusterType = def.Type
	clctrl.HttpClient = http.DefaultClient

	switch clctrl.CloudProvider {
	case "civo":
		clctrl.CivoToken = os.Getenv("CIVO_TOKEN")
	}

	clctrl.Repositories = []string{"gitops", "metaphor"}
	clctrl.Teams = []string{"admins", "developers"}

	clctrl.GitopsTemplateBranchFlag = "main"
	clctrl.GitopsTemplateURLFlag = "https://github.com/kubefirst/gitops-template.git"
	clctrl.KubefirstStateStoreBucketName = fmt.Sprintf("k1-state-store-%s-%s", clctrl.ClusterName, clctrl.ClusterID)

	clctrl.KubefirstTeam = os.Getenv("KUBEFIRST_TEAM")
	if clctrl.KubefirstTeam == "" {
		clctrl.KubefirstTeam = "false"
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

		clctrl.GitHost = k3d.GithubHost
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
		clctrl.GitHost = k3d.GitlabHost
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
	case "k3d":
		clctrl.ProviderConfig = k3d.GetConfig(clctrl.GitProvider, clctrl.GitOwner)
	case "civo":
		clctrl.ProviderConfig = civo.GetConfig(clctrl.ClusterName, clctrl.DomainName, clctrl.GitProvider, clctrl.GitOwner)
	}

	// Write cluster record if it doesn't exist
	cl := types.Cluster{
		ID:                    primitive.NewObjectID(),
		AlertsEmail:           clctrl.AlertsEmail,
		ClusterName:           clctrl.ClusterName,
		CloudProvider:         clctrl.CloudProvider,
		CloudRegion:           clctrl.CloudRegion,
		DomainName:            clctrl.DomainName,
		ClusterID:             clctrl.ClusterID,
		ClusterType:           clctrl.ClusterType,
		GitProvider:           clctrl.GitProvider,
		GitHost:               clctrl.GitHost,
		GitOwner:              clctrl.GitOwner,
		GitUser:               clctrl.GitUser,
		GitToken:              clctrl.GitToken,
		GitlabOwnerGroupID:    clctrl.GitlabOwnerGroupID,
		AtlantisWebhookSecret: clctrl.AtlantisWebhookSecret,
		KubefirstTeam:         clctrl.KubefirstTeam,
		CivoToken:             clctrl.CivoToken,
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

// ParseProviderConfig
func (clctrl *ClusterController) ParseProviderConfig(provider string) interface{} {
	switch provider {
	case "k3d":
		return clctrl.ProviderConfig.(k3d.K3dConfig)
	case "civo":
		return clctrl.ProviderConfig.(*civo.CivoConfig)
	}

	return nil
}
