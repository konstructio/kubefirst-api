/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	"fmt"
	"strconv"

	"github.com/kubefirst/runtime/pkg"
	"github.com/kubefirst/runtime/pkg/terraform"
	log "github.com/sirupsen/logrus"
)

// RunGitTerraform
func (clctrl *ClusterController) RunGitTerraform() error {
	cl, err := clctrl.MdbCl.GetCluster(clctrl.ClusterName)
	if err != nil {
		return err
	}

	if !cl.GitTerraformApplyCheck {
		switch clctrl.GitProvider {
		case "github":
			// //* create teams and repositories in github
			// stelemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricGitTerraformApplyStarted, "")
			log.Info("Creating github resources with terraform")

			tfEntrypoint := clctrl.ProviderConfig.GitopsDir + "/terraform/github"
			tfEnvs := map[string]string{}
			// tfEnvs = k3d.GetGithubTerraformEnvs(tfEnvs)
			tfEnvs["GITHUB_TOKEN"] = clctrl.GitToken
			tfEnvs["GITHUB_OWNER"] = clctrl.GitOwner
			tfEnvs["TF_VAR_kbot_ssh_public_key"] = clctrl.PublicKey
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

			tfEntrypoint := clctrl.ProviderConfig.GitopsDir + "/terraform/gitlab"
			tfEnvs := map[string]string{}
			tfEnvs["GITLAB_TOKEN"] = clctrl.GitToken
			tfEnvs["GITLAB_OWNER"] = clctrl.GitOwner
			tfEnvs["TF_VAR_owner_group_id"] = strconv.Itoa(clctrl.GitlabOwnerGroupID)
			tfEnvs["TF_VAR_kbot_ssh_public_key"] = clctrl.PublicKey
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

		err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "git_terraform_apply_check", true)
		if err != nil {
			return err
		}
	}

	return nil
}
