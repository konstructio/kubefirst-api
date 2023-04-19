/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	"fmt"
	"strconv"

	civoext "github.com/kubefirst/kubefirst-api/extensions/civo"
	digitaloceanext "github.com/kubefirst/kubefirst-api/extensions/digitalocean"
	vultrext "github.com/kubefirst/kubefirst-api/extensions/vultr"
	gitShim "github.com/kubefirst/kubefirst-api/internal/gitShim"
	"github.com/kubefirst/runtime/pkg"
	"github.com/kubefirst/runtime/pkg/civo"
	"github.com/kubefirst/runtime/pkg/digitalocean"
	"github.com/kubefirst/runtime/pkg/k3d"
	"github.com/kubefirst/runtime/pkg/terraform"
	"github.com/kubefirst/runtime/pkg/vultr"
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
			GitToken:     clctrl.GitToken,
			GitOwner:     clctrl.GitOwner,
			Repositories: clctrl.Repositories,
			Teams:        clctrl.Teams,
			GithubOrg:    clctrl.GitOwner,
			GitlabGroup:  clctrl.GitOwner,
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
	cl, err := clctrl.MdbCl.GetCluster(clctrl.ClusterName)
	if err != nil {
		return err
	}

	if !cl.GitTerraformApplyCheck {
		switch clctrl.CloudProvider {
		case "k3d":
			switch clctrl.GitProvider {
			case "github":
				// //* create teams and repositories in github
				// stelemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricGitTerraformApplyStarted, "")
				log.Info("Creating github resources with terraform")

				tfEntrypoint := clctrl.ProviderConfig.(k3d.K3dConfig).GitopsDir + "/terraform/github"
				tfEnvs := map[string]string{}
				// tfEnvs = k3d.GetGithubTerraformEnvs(tfEnvs)
				tfEnvs["GITHUB_TOKEN"] = clctrl.GitToken
				tfEnvs["GITHUB_OWNER"] = clctrl.GitOwner
				tfEnvs["TF_VAR_kbot_ssh_public_key"] = cl.PublicKey
				tfEnvs["AWS_ACCESS_KEY_ID"] = pkg.MinioDefaultUsername
				tfEnvs["AWS_SECRET_ACCESS_KEY"] = pkg.MinioDefaultPassword
				tfEnvs["TF_VAR_aws_access_key_id"] = pkg.MinioDefaultUsername
				tfEnvs["TF_VAR_aws_secret_access_key"] = pkg.MinioDefaultPassword
				err := terraform.InitApplyAutoApprove(false, tfEntrypoint, tfEnvs)
				if err != nil {
					msg := fmt.Sprintf("error creating github resources with terraform %s: %s", tfEntrypoint, err)
					// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricGitTerraformApplyFailed, msg)
					return fmt.Errorf(msg)
				}

				log.Info("created git repositories for github.com/%s", clctrl.GitOwner)
				// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricGitTerraformApplyCompleted, "")
			case "gitlab":
				// //* create teams and repositories in gitlab
				// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricGitTerraformApplyStarted, "")
				log.Info("Creating gitlab resources with terraform")

				tfEntrypoint := clctrl.ProviderConfig.(k3d.K3dConfig).GitopsDir + "/terraform/gitlab"
				tfEnvs := map[string]string{}
				tfEnvs["GITLAB_TOKEN"] = clctrl.GitToken
				tfEnvs["GITLAB_OWNER"] = clctrl.GitOwner
				tfEnvs["TF_VAR_owner_group_id"] = strconv.Itoa(clctrl.GitlabOwnerGroupID)
				tfEnvs["TF_VAR_kbot_ssh_public_key"] = cl.PublicKey
				tfEnvs["AWS_ACCESS_KEY_ID"] = pkg.MinioDefaultUsername
				tfEnvs["AWS_SECRET_ACCESS_KEY"] = pkg.MinioDefaultPassword
				tfEnvs["TF_VAR_aws_access_key_id"] = pkg.MinioDefaultUsername
				tfEnvs["TF_VAR_aws_secret_access_key"] = pkg.MinioDefaultPassword
				err := terraform.InitApplyAutoApprove(false, tfEntrypoint, tfEnvs)
				if err != nil {
					msg := fmt.Sprintf("error creating gitlab resources with terraform %s: %s", tfEntrypoint, err)
					// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricGitTerraformApplyFailed, msg)
					return fmt.Errorf(msg)
				}

				log.Infof("created git projects and groups for gitlab.com/%s", clctrl.GitOwner)
				// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricGitTerraformApplyCompleted, "")
			}
		case "civo":
			switch clctrl.GitProvider {
			case "github":
				// //* create teams and repositories in github
				// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricGitTerraformApplyStarted, "")

				log.Info("Creating github resources with terraform")

				tfEntrypoint := clctrl.ProviderConfig.(*civo.CivoConfig).GitopsDir + "/terraform/github"
				tfEnvs := map[string]string{}
				tfEnvs = civoext.GetGithubTerraformEnvs(tfEnvs, &cl)
				err := terraform.InitApplyAutoApprove(false, tfEntrypoint, tfEnvs)
				if err != nil {
					msg := fmt.Sprintf("error creating github resources with terraform %s: %s", tfEntrypoint, err)
					// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricGitTerraformApplyFailed, msg)
					return fmt.Errorf(msg)
				}

				log.Infof("Created git repositories and teams for github.com/%s", clctrl.GitOwner)
				// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricGitTerraformApplyCompleted, "")
			case "gitlab":
				// //* create teams and repositories in gitlab
				// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricGitTerraformApplyStarted, "")

				log.Info("Creating gitlab resources with terraform")

				tfEntrypoint := clctrl.ProviderConfig.(*civo.CivoConfig).GitopsDir + "/terraform/gitlab"
				tfEnvs := map[string]string{}
				tfEnvs = civoext.GetGitlabTerraformEnvs(tfEnvs, clctrl.GitlabOwnerGroupID, &cl)
				err := terraform.InitApplyAutoApprove(false, tfEntrypoint, tfEnvs)
				if err != nil {
					msg := fmt.Sprintf("error creating gitlab resources with terraform %s: %s", tfEntrypoint, err)
					//telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricGitTerraformApplyFailed, msg)
					return fmt.Errorf(msg)
				}

				log.Infof("created git projects and groups for gitlab.com/%s", clctrl.GitOwner)
				// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricGitTerraformApplyCompleted, "")
			}
		case "digitalocean":
			switch clctrl.GitProvider {
			case "github":
				// //* create teams and repositories in github
				// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricGitTerraformApplyStarted, "")

				log.Info("Creating github resources with terraform")

				tfEntrypoint := clctrl.ProviderConfig.(*digitalocean.DigitaloceanConfig).GitopsDir + "/terraform/github"
				tfEnvs := map[string]string{}
				tfEnvs = digitaloceanext.GetGithubTerraformEnvs(tfEnvs, &cl)
				err := terraform.InitApplyAutoApprove(false, tfEntrypoint, tfEnvs)
				if err != nil {
					msg := fmt.Sprintf("error creating github resources with terraform %s: %s", tfEntrypoint, err)
					// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricGitTerraformApplyFailed, msg)
					return fmt.Errorf(msg)
				}

				log.Infof("Created git repositories and teams for github.com/%s", clctrl.GitOwner)
				// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricGitTerraformApplyCompleted, "")
			case "gitlab":
				// //* create teams and repositories in gitlab
				// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricGitTerraformApplyStarted, "")

				log.Info("Creating gitlab resources with terraform")

				tfEntrypoint := clctrl.ProviderConfig.(*digitalocean.DigitaloceanConfig).GitopsDir + "/terraform/gitlab"
				tfEnvs := map[string]string{}
				tfEnvs = digitaloceanext.GetGitlabTerraformEnvs(tfEnvs, clctrl.GitlabOwnerGroupID, &cl)
				err := terraform.InitApplyAutoApprove(false, tfEntrypoint, tfEnvs)
				if err != nil {
					msg := fmt.Sprintf("error creating gitlab resources with terraform %s: %s", tfEntrypoint, err)
					//telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricGitTerraformApplyFailed, msg)
					return fmt.Errorf(msg)
				}

				log.Infof("created git projects and groups for gitlab.com/%s", clctrl.GitOwner)
				// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricGitTerraformApplyCompleted, "")
			}
		case "vultr":
			switch clctrl.GitProvider {
			case "github":
				// //* create teams and repositories in github
				// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricGitTerraformApplyStarted, "")

				log.Info("Creating github resources with terraform")

				tfEntrypoint := clctrl.ProviderConfig.(*vultr.VultrConfig).GitopsDir + "/terraform/github"
				tfEnvs := map[string]string{}
				tfEnvs = vultrext.GetGithubTerraformEnvs(tfEnvs, &cl)
				err := terraform.InitApplyAutoApprove(false, tfEntrypoint, tfEnvs)
				if err != nil {
					msg := fmt.Sprintf("error creating github resources with terraform %s: %s", tfEntrypoint, err)
					// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricGitTerraformApplyFailed, msg)
					return fmt.Errorf(msg)
				}

				log.Infof("Created git repositories and teams for github.com/%s", clctrl.GitOwner)
				// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricGitTerraformApplyCompleted, "")
			case "gitlab":
				// //* create teams and repositories in gitlab
				// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricGitTerraformApplyStarted, "")

				log.Info("Creating gitlab resources with terraform")

				tfEntrypoint := clctrl.ProviderConfig.(*vultr.VultrConfig).GitopsDir + "/terraform/gitlab"
				tfEnvs := map[string]string{}
				tfEnvs = vultrext.GetGitlabTerraformEnvs(tfEnvs, clctrl.GitlabOwnerGroupID, &cl)
				err := terraform.InitApplyAutoApprove(false, tfEntrypoint, tfEnvs)
				if err != nil {
					msg := fmt.Sprintf("error creating gitlab resources with terraform %s: %s", tfEntrypoint, err)
					//telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricGitTerraformApplyFailed, msg)
					return fmt.Errorf(msg)
				}

				log.Infof("created git projects and groups for gitlab.com/%s", clctrl.GitOwner)
				// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricGitTerraformApplyCompleted, "")
			}
		}

		err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "git_terraform_apply_check", true)
		if err != nil {
			return err
		}
	}

	return nil
}
