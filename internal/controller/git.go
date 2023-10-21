/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	"fmt"
	"os"

	awsext "github.com/kubefirst/kubefirst-api/extensions/aws"
	civoext "github.com/kubefirst/kubefirst-api/extensions/civo"
	digitaloceanext "github.com/kubefirst/kubefirst-api/extensions/digitalocean"
	googleext "github.com/kubefirst/kubefirst-api/extensions/google"
	terraformext "github.com/kubefirst/kubefirst-api/extensions/terraform"
	vultrext "github.com/kubefirst/kubefirst-api/extensions/vultr"
	gitShim "github.com/kubefirst/kubefirst-api/internal/gitShim"
	"github.com/kubefirst/metrics-client/pkg/telemetry"
	"github.com/kubefirst/runtime/pkg/gitlab"
	log "github.com/sirupsen/logrus"
)

// GitInit
func (clctrl *ClusterController) GitInit() error {
	cl, err := clctrl.MdbCl.GetCluster(clctrl.ClusterName)
	if err != nil {
		return err
	}

	if !cl.GitInitCheck {
		// Check for git resources in provider
		initGitParameters := gitShim.GitInitParameters{
			GitProvider:  clctrl.GitProvider,
			GitToken:     clctrl.GitAuth.Token,
			GitOwner:     clctrl.GitAuth.Owner,
			GitProtocol:  clctrl.GitProtocol,
			Repositories: clctrl.Repositories,
			Teams:        clctrl.Teams,
			GithubOrg:    clctrl.GitAuth.Owner,
			GitlabGroup:  clctrl.GitAuth.Owner,
		}
		err := gitShim.InitializeGitProvider(&initGitParameters)
		if err != nil {
			return err
		}

		err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "git_init_check", true)
		if err != nil {
			return err
		}
	}

	return nil
}

// RunGitTerraform
func (clctrl *ClusterController) RunGitTerraform() error {
	// Logging handler
	// Logs to stdout to maintain compatibility with event streaming
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "",
	})
	log.SetReportCaller(false)
	log.SetOutput(os.Stdout)

	cl, err := clctrl.MdbCl.GetCluster(clctrl.ClusterName)
	if err != nil {
		return err
	}

	// Telemetry handler
	segmentClient, err := telemetry.SetupTelemetry(cl)
	if err != nil {
		return err
	}
	defer segmentClient.Client.Close()

	// //* create teams and repositories in github
	//telemetry.Transmit(segmentClient, segment.MetricGitTerraformApplyStarted, "")

	log.Infof("Creating %s resources with terraform", clctrl.GitProvider)

	tfEntrypoint := clctrl.ProviderConfig.GitopsDir + fmt.Sprintf("/terraform/%s", clctrl.GitProvider)
	tfEnvs := map[string]string{}

	if !cl.GitTerraformApplyCheck {
		switch clctrl.GitProvider {
		case "github":
			switch clctrl.CloudProvider {
			case "aws":
				tfEnvs = awsext.GetGithubTerraformEnvs(tfEnvs, &cl)
			case "civo":
				tfEnvs = civoext.GetGithubTerraformEnvs(tfEnvs, &cl)
			case "google":
				tfEnvs = googleext.GetGithubTerraformEnvs(tfEnvs, &cl)
			case "digitalocean":
				tfEnvs = digitaloceanext.GetGithubTerraformEnvs(tfEnvs, &cl)
			case "vultr":
				tfEnvs = vultrext.GetGithubTerraformEnvs(tfEnvs, &cl)
			}
		case "gitlab":
			switch clctrl.CloudProvider {
			case "aws":
				tfEnvs = awsext.GetGitlabTerraformEnvs(tfEnvs, clctrl.GitlabOwnerGroupID, &cl)
			case "civo":
				tfEnvs = civoext.GetGitlabTerraformEnvs(tfEnvs, clctrl.GitlabOwnerGroupID, &cl)
			case "google":
				tfEnvs = googleext.GetGitlabTerraformEnvs(tfEnvs, clctrl.GitlabOwnerGroupID, &cl)
			case "digitalocean":
				tfEnvs = digitaloceanext.GetGitlabTerraformEnvs(tfEnvs, clctrl.GitlabOwnerGroupID, &cl)
			case "vultr":
				tfEnvs = vultrext.GetGitlabTerraformEnvs(tfEnvs, clctrl.GitlabOwnerGroupID, &cl)
			}
		}

		err := terraformext.InitApplyAutoApprove(clctrl.ProviderConfig.TerraformClient, tfEntrypoint, tfEnvs)
		if err != nil {
			msg := fmt.Sprintf("error creating %s resources with terraform %s: %s", clctrl.GitProvider, tfEntrypoint, err)
			log.Error(msg)
			//telemetry.Transmit(segmentClient, segment.MetricGitTerraformApplyFailed, msg)
			return fmt.Errorf(msg)
		}

		log.Infof("created git projects and groups for %s.com/%s", clctrl.GitProvider, clctrl.GitAuth.Owner)
		//telemetry.Transmit(segmentClient, segment.MetricGitTerraformApplyCompleted, "")

		err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "git_terraform_apply_check", true)
		if err != nil {
			return err
		}
	}

	return nil
}

func (clctrl *ClusterController) GetRepoURL() (string, error) {

	// default case is https
	destinationGitopsRepoURL := clctrl.ProviderConfig.DestinationGitopsRepoURL

	switch clctrl.GitProvider {
	case "github":

		// Define constant url based on flag input, only expecting 2 protocols
		switch clctrl.GitProtocol {
		case "ssh": //"ssh"
			destinationGitopsRepoURL = clctrl.ProviderConfig.DestinationGitopsRepoGitURL
		}
	case "gitlab":
		gitlabClient, err := gitlab.NewGitLabClient(clctrl.GitAuth.Token, clctrl.GitAuth.Owner)
		if err != nil {
			return "", err
		}
		// Format git url based on full path to group
		switch clctrl.ProviderConfig.GitProtocol {
		case "https":
			// Update the urls in the cluster for gitlab parent groups
			clctrl.ProviderConfig.DestinationGitopsRepoHttpsURL = fmt.Sprintf("https://gitlab.com/%s/gitops.git", gitlabClient.ParentGroupPath)
			clctrl.ProviderConfig.DestinationMetaphorRepoHttpsURL = fmt.Sprintf("https://gitlab.com/%s/metaphor.git", gitlabClient.ParentGroupPath)
		default:
			// Update the urls in the cluster for gitlab parent group
			clctrl.ProviderConfig.DestinationGitopsRepoGitURL = fmt.Sprintf("git@gitlab.com:%s/gitops.git", gitlabClient.ParentGroupPath)
			clctrl.ProviderConfig.DestinationMetaphorRepoGitURL = fmt.Sprintf("git@gitlab.com:%s/metaphor.git", gitlabClient.ParentGroupPath)
			// Return the url used for detokenization
			destinationGitopsRepoURL = clctrl.ProviderConfig.DestinationGitopsRepoGitURL
		}
	}

	return destinationGitopsRepoURL, nil
}
