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

	// config
	ProviderConfig *k3d.K3dConfig

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
	AtlantisWebhookSecret    string
	KubefirstTeam            string
	GitopsTemplateBranchFlag string
	GitopsTemplateURLFlag    string

	// keys
	// kbot public key
	PublicKey string
	// kbot private key
	PrivateKey string

	// Database Controller
	MdbCl *db.MongoDBClient
}

// InitController
// createCmd.Flags().StringVar(&clusterNameFlag, "cluster-name", "kubefirst", "the name of the cluster to create")
// createCmd.Flags().StringVar(&clusterTypeFlag, "cluster-type", "mgmt", "the type of cluster to create (i.e. mgmt|workload)")
// createCmd.Flags().BoolVar(&dryRun, "dry-run", false, "don't execute the installation")
// createCmd.Flags().StringVar(&gitProviderFlag, "git-provider", "github", fmt.Sprintf("the git provider - one of: %s", supportedGitProviders))
// createCmd.Flags().StringVar(&githubUserFlag, "github-user", "", "the GitHub user for the new gitops and metaphor repositories - this cannot be used with --github-org")
// createCmd.Flags().StringVar(&githubOrgFlag, "github-org", "", "the GitHub organization for the new gitops and metaphor repositories - this cannot be used with --github-user")
// createCmd.Flags().StringVar(&gitlabGroupFlag, "gitlab-group", "", "the GitLab group for the new gitops and metaphor projects - required if using gitlab")
// createCmd.Flags().StringVar(&gitopsTemplateBranchFlag, "gitops-template-branch", "main", "the branch to clone for the gitops-template repository")
// createCmd.Flags().StringVar(&gitopsTemplateURLFlag, "gitops-template-url", "https://github.com/kubefirst/gitops-template.git", "the fully qualified url to the gitops-template repository to clone")
// createCmd.Flags().StringVar(&kbotPasswordFlag, "kbot-password", "", "the default password to use for the kbot user")
// createCmd.Flags().BoolVar(&useTelemetryFlag, "use-telemetry", true, "whether to emit telemetry")
func (clctrl *ClusterController) InitController(def *types.ClusterDefinition) error {
	// Database controller
	clctrl.MdbCl = &db.MongoDBClient{}
	err := clctrl.MdbCl.InitDatabase()
	if err != nil {
		return err
	}

	clctrl.CloudProvider = "k3d"
	clctrl.CloudRegion = def.CloudRegion
	clctrl.ClusterName = def.ClusterName
	clctrl.ClusterID = pkg.GenerateClusterID()
	clctrl.DomainName = def.DomainName
	clctrl.ClusterType = def.Type
	clctrl.HttpClient = http.DefaultClient
	clctrl.Repositories = []string{"gitops", "metaphor"}
	clctrl.Teams = []string{"admins", "developers"}

	clctrl.GitopsTemplateBranchFlag = "main"
	clctrl.GitopsTemplateURLFlag = "https://github.com/kubefirst/gitops-template.git"

	clctrl.KubefirstTeam = os.Getenv("KUBEFIRST_TEAM")
	if clctrl.KubefirstTeam == "" {
		clctrl.KubefirstTeam = "false"
	}
	clctrl.AtlantisWebhookSecret = pkg.Random(20)

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

	// Check for git resources in provider
	//initGitParameters := gitShim.GitInitParameters{
	//	GitProvider:  clctrl.GitProvider,
	//	GitToken:     clctrl.GitToken,
	//	GitOwner:     clctrl.GitOwner,
	//	Repositories: clctrl.Repositories,
	//	Teams:        clctrl.Teams,
	//	GithubOrg:    clctrl.GitOwner,
	//	GitlabGroup:  clctrl.GitOwner,
	//}
	//err = gitShim.InitializeGitProvider(&initGitParameters)
	//if err != nil {
	//	return err
	//}

	// Instantiate provider configuration
	clctrl.ProviderConfig = k3d.GetConfig(clctrl.GitProvider, clctrl.GitOwner)

	// Write cluster record if it doesn't exist
	cl := db.Cluster{
		ID:                    primitive.NewObjectID(),
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
	}
	err = clctrl.MdbCl.InsertCluster(cl)
	if err != nil {
		return err
	}

	return nil
}

// GetCurrentClusterRecord will return an active cluster's record if it exists
func (clctrl *ClusterController) GetCurrentClusterRecord() (db.Cluster, error) {
	cl, err := clctrl.MdbCl.GetCluster(clctrl.ClusterName)
	if err != nil {
		return db.Cluster{}, err
	}

	return cl, nil

}
