/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	"fmt"
	"os"

	githttps "github.com/go-git/go-git/v5/plumbing/transport/http"
	awsext "github.com/kubefirst/kubefirst-api/extensions/aws"
	civoext "github.com/kubefirst/kubefirst-api/extensions/civo"
	digitaloceanext "github.com/kubefirst/kubefirst-api/extensions/digitalocean"
	terraformext "github.com/kubefirst/kubefirst-api/extensions/terraform"
	vultrext "github.com/kubefirst/kubefirst-api/extensions/vultr"
	gitShim "github.com/kubefirst/kubefirst-api/internal/gitShim"
	"github.com/kubefirst/kubefirst-api/internal/telemetryShim"
	"github.com/kubefirst/runtime/pkg/gitlab"
	"github.com/kubefirst/runtime/pkg/segment"
	log "github.com/sirupsen/logrus"
)

// GitInit
func (clctrl *ClusterController) GitInit() error {
	cl, err := clctrl.MdbCl.GetCluster(clctrl.ClusterName)
	if err != nil {
		return err
	}

	clctrl.GitAuth.HttpAuth = githttps.BasicAuth{
		Username: clctrl.GitAuth.User,
		Password: clctrl.GitAuth.Token,
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
	segmentClient, err := telemetryShim.SetupTelemetry(cl)
	if err != nil {
		return err
	}
	defer segmentClient.Client.Close()

	if !cl.GitTerraformApplyCheck {
		switch clctrl.CloudProvider {
		case "aws":
			switch clctrl.GitProvider {
			case "github":
				// //* create teams and repositories in github
				telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitTerraformApplyStarted, "")

				log.Info("Creating github resources with terraform")

				tfEntrypoint := clctrl.ProviderConfig.GitopsDir + "/terraform/github"
				tfEnvs := map[string]string{}
				tfEnvs = awsext.GetGithubTerraformEnvs(tfEnvs, &cl)
				err := terraformext.InitApplyAutoApprove(clctrl.ProviderConfig.TerraformClient, tfEntrypoint, tfEnvs)
				if err != nil {
					msg := fmt.Sprintf("error creating github resources with terraform %s: %s", tfEntrypoint, err)
					log.Error(msg)
					telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitTerraformApplyFailed, msg)
					return fmt.Errorf(msg)
				}

				log.Infof("Created git repositories and teams for github.com/%s", clctrl.GitAuth.Owner)
				telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitTerraformApplyCompleted, "")
			case "gitlab":
				// //* create teams and repositories in gitlab
				telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitTerraformApplyStarted, "")

				log.Info("Creating gitlab resources with terraform")

				tfEntrypoint := clctrl.ProviderConfig.GitopsDir + "/terraform/gitlab"
				tfEnvs := map[string]string{}
				tfEnvs = awsext.GetGitlabTerraformEnvs(tfEnvs, clctrl.GitlabOwnerGroupID, &cl)
				err := terraformext.InitApplyAutoApprove(clctrl.ProviderConfig.TerraformClient, tfEntrypoint, tfEnvs)
				if err != nil {
					msg := fmt.Sprintf("error creating gitlab resources with terraform %s: %s", tfEntrypoint, err)
					log.Error(msg)
					telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitTerraformApplyFailed, msg)
					return fmt.Errorf(msg)
				}

				log.Infof("created git projects and groups for gitlab.com/%s", clctrl.GitAuth.Owner)
				telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitTerraformApplyCompleted, "")
			}
		case "civo":
			switch clctrl.GitProvider {
			case "github":
				// //* create teams and repositories in github
				telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitTerraformApplyStarted, "")

				log.Info("Creating github resources with terraform")

				tfEntrypoint := clctrl.ProviderConfig.GitopsDir + "/terraform/github"
				tfEnvs := map[string]string{}
				tfEnvs = civoext.GetGithubTerraformEnvs(tfEnvs, &cl)
				err := terraformext.InitApplyAutoApprove(clctrl.ProviderConfig.TerraformClient, tfEntrypoint, tfEnvs)
				if err != nil {
					msg := fmt.Sprintf("error creating github resources with terraform %s: %s", tfEntrypoint, err)
					log.Error(msg)
					telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitTerraformApplyFailed, msg)
					return fmt.Errorf(msg)
				}

				log.Infof("Created git repositories and teams for github.com/%s", clctrl.GitAuth.Owner)
				telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitTerraformApplyCompleted, "")
			case "gitlab":
				// //* create teams and repositories in gitlab
				telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitTerraformApplyStarted, "")

				log.Info("Creating gitlab resources with terraform")

				tfEntrypoint := clctrl.ProviderConfig.GitopsDir + "/terraform/gitlab"
				tfEnvs := map[string]string{}
				tfEnvs = civoext.GetGitlabTerraformEnvs(tfEnvs, clctrl.GitlabOwnerGroupID, &cl)
				err := terraformext.InitApplyAutoApprove(clctrl.ProviderConfig.TerraformClient, tfEntrypoint, tfEnvs)
				if err != nil {
					msg := fmt.Sprintf("error creating gitlab resources with terraform %s: %s", tfEntrypoint, err)
					log.Error(msg)
					telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitTerraformApplyFailed, msg)
					return fmt.Errorf(msg)
				}

				log.Infof("created git projects and groups for gitlab.com/%s", clctrl.GitAuth.Owner)
				telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitTerraformApplyCompleted, "")
			}
		case "digitalocean":
			switch clctrl.GitProvider {
			case "github":
				// //* create teams and repositories in github
				telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitTerraformApplyStarted, "")

				log.Info("Creating github resources with terraform")

				tfEntrypoint := clctrl.ProviderConfig.GitopsDir + "/terraform/github"
				tfEnvs := map[string]string{}
				tfEnvs = digitaloceanext.GetGithubTerraformEnvs(tfEnvs, &cl)
				err := terraformext.InitApplyAutoApprove(clctrl.ProviderConfig.TerraformClient, tfEntrypoint, tfEnvs)
				if err != nil {
					msg := fmt.Sprintf("error creating github resources with terraform %s: %s", tfEntrypoint, err)
					log.Error(msg)
					telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitTerraformApplyFailed, msg)
					return fmt.Errorf(msg)
				}

				log.Infof("Created git repositories and teams for github.com/%s", clctrl.GitAuth.Owner)
				telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitTerraformApplyCompleted, "")
			case "gitlab":
				// //* create teams and repositories in gitlab
				telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitTerraformApplyStarted, "")

				log.Info("Creating gitlab resources with terraform")

				tfEntrypoint := clctrl.ProviderConfig.GitopsDir + "/terraform/gitlab"
				tfEnvs := map[string]string{}
				tfEnvs = digitaloceanext.GetGitlabTerraformEnvs(tfEnvs, clctrl.GitlabOwnerGroupID, &cl)
				err := terraformext.InitApplyAutoApprove(clctrl.ProviderConfig.TerraformClient, tfEntrypoint, tfEnvs)
				if err != nil {
					msg := fmt.Sprintf("error creating gitlab resources with terraform %s: %s", tfEntrypoint, err)
					log.Error(msg)
					telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitTerraformApplyFailed, msg)
					return fmt.Errorf(msg)
				}

				log.Infof("created git projects and groups for gitlab.com/%s", clctrl.GitAuth.Owner)
				telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitTerraformApplyCompleted, "")
			}
		case "vultr":
			switch clctrl.GitProvider {
			case "github":
				// //* create teams and repositories in github
				telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitTerraformApplyStarted, "")

				log.Info("Creating github resources with terraform")

				tfEntrypoint := clctrl.ProviderConfig.GitopsDir + "/terraform/github"
				tfEnvs := map[string]string{}
				tfEnvs = vultrext.GetGithubTerraformEnvs(tfEnvs, &cl)
				err := terraformext.InitApplyAutoApprove(clctrl.ProviderConfig.TerraformClient, tfEntrypoint, tfEnvs)
				if err != nil {
					msg := fmt.Sprintf("error creating github resources with terraform %s: %s", tfEntrypoint, err)
					log.Error(msg)
					telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitTerraformApplyFailed, msg)
					return fmt.Errorf(msg)
				}

				log.Infof("Created git repositories and teams for github.com/%s", clctrl.GitAuth.Owner)
				telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitTerraformApplyCompleted, "")
			case "gitlab":
				// //* create teams and repositories in gitlab
				telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitTerraformApplyStarted, "")

				log.Info("Creating gitlab resources with terraform")

				tfEntrypoint := clctrl.ProviderConfig.GitopsDir + "/terraform/gitlab"
				tfEnvs := map[string]string{}
				tfEnvs = vultrext.GetGitlabTerraformEnvs(tfEnvs, clctrl.GitlabOwnerGroupID, &cl)
				err := terraformext.InitApplyAutoApprove(clctrl.ProviderConfig.TerraformClient, tfEntrypoint, tfEnvs)
				if err != nil {
					msg := fmt.Sprintf("error creating gitlab resources with terraform %s: %s", tfEntrypoint, err)
					log.Error(msg)
					telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitTerraformApplyFailed, msg)
					return fmt.Errorf(msg)
				}

				log.Infof("created git projects and groups for gitlab.com/%s", clctrl.GitAuth.Owner)
				telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitTerraformApplyCompleted, "")
			}
		}

		err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "git_terraform_apply_check", true)
		if err != nil {
			return err
		}
	}

	return nil
}

func (clctrl *ClusterController) GitURL() (string, error) {

	var destinationGitopsRepoURL string

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
			// Return the url used for detokenization
			destinationGitopsRepoURL = clctrl.ProviderConfig.DestinationGitopsRepoHttpsURL

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
