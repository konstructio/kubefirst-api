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
	terraformext "github.com/kubefirst/kubefirst-api/extensions/terraform"
	vultrext "github.com/kubefirst/kubefirst-api/extensions/vultr"
	gitShim "github.com/kubefirst/kubefirst-api/internal/gitShim"
	"github.com/kubefirst/kubefirst-api/internal/telemetryShim"
	awsinternal "github.com/kubefirst/runtime/pkg/aws"
	"github.com/kubefirst/runtime/pkg/civo"
	"github.com/kubefirst/runtime/pkg/digitalocean"
	"github.com/kubefirst/runtime/pkg/segment"
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

				tfEntrypoint := clctrl.ProviderConfig.(*awsinternal.AwsConfig).GitopsDir + "/terraform/github"
				tfEnvs := map[string]string{}
				tfEnvs = awsext.GetGithubTerraformEnvs(tfEnvs, &cl)
				err := terraformext.InitApplyAutoApprove(clctrl.ProviderConfig.(*awsinternal.AwsConfig).TerraformClient, tfEntrypoint, tfEnvs)
				if err != nil {
					msg := fmt.Sprintf("error creating github resources with terraform %s: %s", tfEntrypoint, err)
					log.Error(msg)
					telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitTerraformApplyFailed, msg)
					return fmt.Errorf(msg)
				}

				log.Infof("Created git repositories and teams for github.com/%s", clctrl.GitOwner)
				telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitTerraformApplyCompleted, "")
			case "gitlab":
				// //* create teams and repositories in gitlab
				telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitTerraformApplyStarted, "")

				log.Info("Creating gitlab resources with terraform")

				tfEntrypoint := clctrl.ProviderConfig.(*awsinternal.AwsConfig).GitopsDir + "/terraform/gitlab"
				tfEnvs := map[string]string{}
				tfEnvs = awsext.GetGitlabTerraformEnvs(tfEnvs, clctrl.GitlabOwnerGroupID, &cl)
				err := terraformext.InitApplyAutoApprove(clctrl.ProviderConfig.(*awsinternal.AwsConfig).TerraformClient, tfEntrypoint, tfEnvs)
				if err != nil {
					msg := fmt.Sprintf("error creating gitlab resources with terraform %s: %s", tfEntrypoint, err)
					log.Error(msg)
					telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitTerraformApplyFailed, msg)
					return fmt.Errorf(msg)
				}

				log.Infof("created git projects and groups for gitlab.com/%s", clctrl.GitOwner)
				telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitTerraformApplyCompleted, "")
			}
		case "civo":
			switch clctrl.GitProvider {
			case "github":
				// //* create teams and repositories in github
				telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitTerraformApplyStarted, "")

				log.Info("Creating github resources with terraform")

				tfEntrypoint := clctrl.ProviderConfig.(*civo.CivoConfig).GitopsDir + "/terraform/github"
				tfEnvs := map[string]string{}
				tfEnvs = civoext.GetGithubTerraformEnvs(tfEnvs, &cl)
				err := terraformext.InitApplyAutoApprove(clctrl.ProviderConfig.(*civo.CivoConfig).TerraformClient, tfEntrypoint, tfEnvs)
				if err != nil {
					msg := fmt.Sprintf("error creating github resources with terraform %s: %s", tfEntrypoint, err)
					log.Error(msg)
					telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitTerraformApplyFailed, msg)
					return fmt.Errorf(msg)
				}

				log.Infof("Created git repositories and teams for github.com/%s", clctrl.GitOwner)
				telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitTerraformApplyCompleted, "")
			case "gitlab":
				// //* create teams and repositories in gitlab
				telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitTerraformApplyStarted, "")

				log.Info("Creating gitlab resources with terraform")

				tfEntrypoint := clctrl.ProviderConfig.(*civo.CivoConfig).GitopsDir + "/terraform/gitlab"
				tfEnvs := map[string]string{}
				tfEnvs = civoext.GetGitlabTerraformEnvs(tfEnvs, clctrl.GitlabOwnerGroupID, &cl)
				err := terraformext.InitApplyAutoApprove(clctrl.ProviderConfig.(*civo.CivoConfig).TerraformClient, tfEntrypoint, tfEnvs)
				if err != nil {
					msg := fmt.Sprintf("error creating gitlab resources with terraform %s: %s", tfEntrypoint, err)
					log.Error(msg)
					telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitTerraformApplyFailed, msg)
					return fmt.Errorf(msg)
				}

				log.Infof("created git projects and groups for gitlab.com/%s", clctrl.GitOwner)
				telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitTerraformApplyCompleted, "")
			}
		case "digitalocean":
			switch clctrl.GitProvider {
			case "github":
				// //* create teams and repositories in github
				telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitTerraformApplyStarted, "")

				log.Info("Creating github resources with terraform")

				tfEntrypoint := clctrl.ProviderConfig.(*digitalocean.DigitaloceanConfig).GitopsDir + "/terraform/github"
				tfEnvs := map[string]string{}
				tfEnvs = digitaloceanext.GetGithubTerraformEnvs(tfEnvs, &cl)
				err := terraformext.InitApplyAutoApprove(clctrl.ProviderConfig.(*digitalocean.DigitaloceanConfig).TerraformClient, tfEntrypoint, tfEnvs)
				if err != nil {
					msg := fmt.Sprintf("error creating github resources with terraform %s: %s", tfEntrypoint, err)
					log.Error(msg)
					telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitTerraformApplyFailed, msg)
					return fmt.Errorf(msg)
				}

				log.Infof("Created git repositories and teams for github.com/%s", clctrl.GitOwner)
				telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitTerraformApplyCompleted, "")
			case "gitlab":
				// //* create teams and repositories in gitlab
				telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitTerraformApplyStarted, "")

				log.Info("Creating gitlab resources with terraform")

				tfEntrypoint := clctrl.ProviderConfig.(*digitalocean.DigitaloceanConfig).GitopsDir + "/terraform/gitlab"
				tfEnvs := map[string]string{}
				tfEnvs = digitaloceanext.GetGitlabTerraformEnvs(tfEnvs, clctrl.GitlabOwnerGroupID, &cl)
				err := terraformext.InitApplyAutoApprove(clctrl.ProviderConfig.(*digitalocean.DigitaloceanConfig).TerraformClient, tfEntrypoint, tfEnvs)
				if err != nil {
					msg := fmt.Sprintf("error creating gitlab resources with terraform %s: %s", tfEntrypoint, err)
					log.Error(msg)
					telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitTerraformApplyFailed, msg)
					return fmt.Errorf(msg)
				}

				log.Infof("created git projects and groups for gitlab.com/%s", clctrl.GitOwner)
				telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitTerraformApplyCompleted, "")
			}
		case "vultr":
			switch clctrl.GitProvider {
			case "github":
				// //* create teams and repositories in github
				telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitTerraformApplyStarted, "")

				log.Info("Creating github resources with terraform")

				tfEntrypoint := clctrl.ProviderConfig.(*vultr.VultrConfig).GitopsDir + "/terraform/github"
				tfEnvs := map[string]string{}
				tfEnvs = vultrext.GetGithubTerraformEnvs(tfEnvs, &cl)
				err := terraformext.InitApplyAutoApprove(clctrl.ProviderConfig.(*vultr.VultrConfig).TerraformClient, tfEntrypoint, tfEnvs)
				if err != nil {
					msg := fmt.Sprintf("error creating github resources with terraform %s: %s", tfEntrypoint, err)
					log.Error(msg)
					telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitTerraformApplyFailed, msg)
					return fmt.Errorf(msg)
				}

				log.Infof("Created git repositories and teams for github.com/%s", clctrl.GitOwner)
				telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitTerraformApplyCompleted, "")
			case "gitlab":
				// //* create teams and repositories in gitlab
				telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitTerraformApplyStarted, "")

				log.Info("Creating gitlab resources with terraform")

				tfEntrypoint := clctrl.ProviderConfig.(*vultr.VultrConfig).GitopsDir + "/terraform/gitlab"
				tfEnvs := map[string]string{}
				tfEnvs = vultrext.GetGitlabTerraformEnvs(tfEnvs, clctrl.GitlabOwnerGroupID, &cl)
				err := terraformext.InitApplyAutoApprove(clctrl.ProviderConfig.(*vultr.VultrConfig).TerraformClient, tfEntrypoint, tfEnvs)
				if err != nil {
					msg := fmt.Sprintf("error creating gitlab resources with terraform %s: %s", tfEntrypoint, err)
					log.Error(msg)
					telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitTerraformApplyFailed, msg)
					return fmt.Errorf(msg)
				}

				log.Infof("created git projects and groups for gitlab.com/%s", clctrl.GitOwner)
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
