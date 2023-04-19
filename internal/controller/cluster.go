/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	"fmt"

	civoext "github.com/kubefirst/kubefirst-api/extensions/civo"
	digitaloceanext "github.com/kubefirst/kubefirst-api/extensions/digitalocean"
	vultrext "github.com/kubefirst/kubefirst-api/extensions/vultr"
	gitShim "github.com/kubefirst/kubefirst-api/internal/gitShim"
	"github.com/kubefirst/runtime/configs"
	"github.com/kubefirst/runtime/pkg/civo"
	"github.com/kubefirst/runtime/pkg/digitalocean"
	"github.com/kubefirst/runtime/pkg/gitlab"
	"github.com/kubefirst/runtime/pkg/k3d"
	"github.com/kubefirst/runtime/pkg/k8s"
	"github.com/kubefirst/runtime/pkg/terraform"
	"github.com/kubefirst/runtime/pkg/vultr"
	log "github.com/sirupsen/logrus"
)

// Global Controller Variables
// Civo
// gitlab may have subgroups, so the destination gitops/metaphor repo git urls may be different
var CivoDestinationGitopsRepoGitURL, CivoDestinationMetaphorRepoGitURL string

// Digital Ocean
// gitlab may have subgroups, so the destination gitops/metaphor repo git urls may be different
var DigitaloceanDestinationGitopsRepoGitURL, DigitaloceanDestinationMetaphorRepoGitURL string

// Vultr
// gitlab may have subgroups, so the destination gitops/metaphor repo git urls may be different
var VultrDestinationGitopsRepoGitURL, VultrDestinationMetaphorRepoGitURL string

// CreateCluster
func (clctrl *ClusterController) CreateCluster() error {
	cl, err := clctrl.MdbCl.GetCluster(clctrl.ClusterName)
	if err != nil {
		return err
	}

	if !cl.CloudTerraformApplyCheck || cl.CloudTerraformApplyFailedCheck {
		// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricCloudTerraformApplyStarted, "")

		log.Infof("creating %s cluster", clctrl.CloudProvider)

		switch clctrl.CloudProvider {
		case "k3d":
			err := k3d.ClusterCreate(
				clctrl.ClusterName,
				clctrl.ProviderConfig.(k3d.K3dConfig).K1Dir,
				clctrl.ProviderConfig.(k3d.K3dConfig).K3dClient,
				clctrl.ProviderConfig.(k3d.K3dConfig).Kubeconfig,
			)
			if err != nil {
				msg := fmt.Sprintf("error creating k3d resources: %s", err)
				// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricCloudTerraformApplyFailed, msg)
				err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "cloud_terraform_apply_failed_check", true)
				if err != nil {
					return err
				}
				return fmt.Errorf(msg)
			}
		case "civo":
			// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricCloudTerraformApplyStarted, "")

			log.Info("creating civo cloud resources with terraform")

			tfEntrypoint := clctrl.ProviderConfig.(*civo.CivoConfig).GitopsDir + "/terraform/civo"
			tfEnvs := map[string]string{}
			tfEnvs = civoext.GetCivoTerraformEnvs(tfEnvs, &cl)
			err := terraform.InitApplyAutoApprove(false, tfEntrypoint, tfEnvs)
			if err != nil {
				msg := fmt.Sprintf("error creating civo resources with terraform %s : %s", tfEntrypoint, err)
				err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "cloud_terraform_apply_failed_check", true)
				if err != nil {
					return err
				}
				// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricCloudTerraformApplyFailed, msg)
				return fmt.Errorf(msg)
			}

			log.Info("created civo cloud resources")

			// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricCloudTerraformApplyCompleted, "")
		case "digitalocean":
			// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricCloudTerraformApplyStarted, "")

			log.Info("creating digital ocean cloud resources with terraform")

			tfEntrypoint := clctrl.ProviderConfig.(*digitalocean.DigitaloceanConfig).GitopsDir + "/terraform/digitalocean"
			tfEnvs := map[string]string{}
			tfEnvs = digitaloceanext.GetDigitaloceanTerraformEnvs(tfEnvs, &cl)
			err := terraform.InitApplyAutoApprove(false, tfEntrypoint, tfEnvs)
			if err != nil {
				msg := fmt.Sprintf("error creating digital ocean resources with terraform %s : %s", tfEntrypoint, err)
				err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "cloud_terraform_apply_failed_check", true)
				if err != nil {
					return err
				}
				// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricCloudTerraformApplyFailed, msg)
				return fmt.Errorf(msg)
			}

			log.Info("created digital ocean cloud resources")

			// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricCloudTerraformApplyCompleted, "")
		case "vultr":
			// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricCloudTerraformApplyStarted, "")

			log.Info("creating vultr cloud resources with terraform")

			tfEntrypoint := clctrl.ProviderConfig.(*vultr.VultrConfig).GitopsDir + "/terraform/vultr"
			tfEnvs := map[string]string{}
			tfEnvs = vultrext.GetVultrTerraformEnvs(tfEnvs, &cl)
			err := terraform.InitApplyAutoApprove(false, tfEntrypoint, tfEnvs)
			if err != nil {
				msg := fmt.Sprintf("error creating vultr resources with terraform %s : %s", tfEntrypoint, err)
				err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "cloud_terraform_apply_failed_check", true)
				if err != nil {
					return err
				}
				// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricCloudTerraformApplyFailed, msg)
				return fmt.Errorf(msg)
			}

			log.Info("created vultr cloud resources")

			// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricCloudTerraformApplyCompleted, "")
		}

		log.Infof("successfully created %s cluster", clctrl.CloudProvider)

		// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricCloudTerraformApplyCompleted, "")

		err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "cloud_terraform_apply_failed_check", false)
		if err != nil {
			return err
		}

		err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "cloud_terraform_apply_check", true)
		if err != nil {
			return err
		}
	}

	return nil
}

// CreateTokens
func (clctrl *ClusterController) CreateTokens(kind string) interface{} {
	var gitopsTemplateTokens interface{}

	switch kind {
	case "gitops":
		switch clctrl.CloudProvider {
		case "k3d":
			gitopsTemplateTokens = &k3d.GitopsTokenValues{
				GithubOwner:                   clctrl.GitOwner,
				GithubUser:                    clctrl.GitUser,
				GitlabOwner:                   clctrl.GitOwner,
				GitlabOwnerGroupID:            clctrl.GitlabOwnerGroupID,
				GitlabUser:                    clctrl.GitUser,
				DomainName:                    clctrl.DomainName,
				AtlantisAllowList:             fmt.Sprintf("%s/%s/*", clctrl.GitHost, clctrl.GitOwner),
				AlertsEmail:                   "REMOVE_THIS_VALUE",
				ClusterName:                   clctrl.ClusterName,
				ClusterType:                   clctrl.ClusterType,
				GithubHost:                    clctrl.GitHost,
				GitlabHost:                    clctrl.GitHost,
				ArgoWorkflowsIngressURL:       fmt.Sprintf("https://argo.%s", clctrl.DomainName),
				VaultIngressURL:               fmt.Sprintf("https://vault.%s", clctrl.DomainName),
				ArgocdIngressURL:              fmt.Sprintf("https://argocd.%s", clctrl.DomainName),
				AtlantisIngressURL:            fmt.Sprintf("https://atlantis.%s", clctrl.DomainName),
				MetaphorDevelopmentIngressURL: fmt.Sprintf("https://metaphor-development.%s", clctrl.DomainName),
				MetaphorStagingIngressURL:     fmt.Sprintf("https://metaphor-staging.%s", clctrl.DomainName),
				MetaphorProductionIngressURL:  fmt.Sprintf("https://metaphor-production.%s", clctrl.DomainName),
				KubefirstVersion:              configs.K1Version,
				KubefirstTeam:                 clctrl.KubefirstTeam,
				KubeconfigPath:                clctrl.ProviderConfig.(k3d.K3dConfig).Kubeconfig,
				GitopsRepoGitURL:              clctrl.ProviderConfig.(k3d.K3dConfig).DestinationGitopsRepoGitURL,
				GitProvider:                   clctrl.ProviderConfig.(k3d.K3dConfig).GitProvider,
				ClusterId:                     clctrl.ClusterID,
				CloudProvider:                 clctrl.CloudProvider,
			}
		case "civo":
			gitopsTemplateTokens = &civo.GitOpsDirectoryValues{
				AlertsEmail:               clctrl.AlertsEmail,
				AtlantisAllowList:         fmt.Sprintf("%s/%s/*", clctrl.GitHost, clctrl.GitOwner),
				CloudProvider:             clctrl.CloudProvider,
				CloudRegion:               clctrl.CloudRegion,
				ClusterName:               clctrl.ClusterName,
				ClusterType:               clctrl.ClusterType,
				DomainName:                clctrl.DomainName,
				KubeconfigPath:            clctrl.ProviderConfig.(*civo.CivoConfig).Kubeconfig,
				KubefirstStateStoreBucket: clctrl.KubefirstStateStoreBucketName,
				KubefirstTeam:             clctrl.KubefirstTeam,
				KubefirstVersion:          configs.K1Version,

				ArgoCDIngressURL:               fmt.Sprintf("https://argocd.%s", clctrl.DomainName),
				ArgoCDIngressNoHTTPSURL:        fmt.Sprintf("argocd.%s", clctrl.DomainName),
				ArgoWorkflowsIngressURL:        fmt.Sprintf("https://argo.%s", clctrl.DomainName),
				ArgoWorkflowsIngressNoHTTPSURL: fmt.Sprintf("argo.%s", clctrl.DomainName),
				AtlantisIngressURL:             fmt.Sprintf("https://atlantis.%s", clctrl.DomainName),
				AtlantisIngressNoHTTPSURL:      fmt.Sprintf("atlantis.%s", clctrl.DomainName),
				ChartMuseumIngressURL:          fmt.Sprintf("https://chartmuseum.%s", clctrl.DomainName),
				VaultIngressURL:                fmt.Sprintf("https://vault.%s", clctrl.DomainName),
				VaultIngressNoHTTPSURL:         fmt.Sprintf("vault.%s", clctrl.DomainName),
				VouchIngressURL:                fmt.Sprintf("https://vouch.%s", clctrl.DomainName),

				GitDescription:       fmt.Sprintf("%s hosted git", clctrl.GitProvider),
				GitNamespace:         "N/A",
				GitProvider:          clctrl.GitProvider,
				GitRunner:            fmt.Sprintf("%s Runner", clctrl.GitProvider),
				GitRunnerDescription: fmt.Sprintf("Self Hosted %s Runner", clctrl.GitProvider),
				GitRunnerNS:          fmt.Sprintf("%s-runner", clctrl.GitProvider),
				GitURL:               clctrl.GitopsTemplateURLFlag,

				GitHubHost:  fmt.Sprintf("https://github.com/%s/gitops.git", clctrl.GitOwner),
				GitHubOwner: clctrl.GitOwner,
				GitHubUser:  clctrl.GitUser,

				GitlabHost:         clctrl.GitHost,
				GitlabOwner:        clctrl.GitOwner,
				GitlabOwnerGroupID: clctrl.GitlabOwnerGroupID,
				GitlabUser:         clctrl.GitUser,

				GitOpsRepoAtlantisWebhookURL: clctrl.AtlantisWebhookURL,
				GitOpsRepoNoHTTPSURL:         fmt.Sprintf("%s.com/%s/gitops.git", clctrl.GitHost, clctrl.GitOwner),
				ClusterId:                    clctrl.ClusterID,
			}

			switch clctrl.GitProvider {
			case "github":
				CivoDestinationGitopsRepoGitURL = clctrl.ProviderConfig.(*civo.CivoConfig).DestinationGitopsRepoGitURL
				CivoDestinationMetaphorRepoGitURL = clctrl.ProviderConfig.(*civo.CivoConfig).DestinationMetaphorRepoGitURL
			case "gitlab":
				gitlabClient, err := gitlab.NewGitLabClient(clctrl.GitToken, clctrl.GitOwner)
				if err != nil {
					return err
				}
				// Format git url based on full path to group
				CivoDestinationGitopsRepoGitURL = fmt.Sprintf("git@gitlab.com:%s/gitops.git", gitlabClient.ParentGroupPath)
				CivoDestinationMetaphorRepoGitURL = fmt.Sprintf("git@gitlab.com:%s/metaphor.git", gitlabClient.ParentGroupPath)

			}
			gitopsTemplateTokens.(*civo.GitOpsDirectoryValues).GitOpsRepoGitURL = CivoDestinationGitopsRepoGitURL
		case "digitalocean":
			gitopsTemplateTokens = &digitalocean.GitOpsDirectoryValues{
				AlertsEmail:               clctrl.AlertsEmail,
				AtlantisAllowList:         fmt.Sprintf("%s/%s/*", clctrl.GitHost, clctrl.GitOwner),
				CloudProvider:             clctrl.CloudProvider,
				CloudRegion:               clctrl.CloudRegion,
				ClusterName:               clctrl.ClusterName,
				ClusterType:               clctrl.ClusterType,
				DomainName:                clctrl.DomainName,
				KubeconfigPath:            clctrl.ProviderConfig.(*digitalocean.DigitaloceanConfig).Kubeconfig,
				KubefirstStateStoreBucket: clctrl.KubefirstStateStoreBucketName,
				KubefirstTeam:             clctrl.KubefirstTeam,
				KubefirstVersion:          configs.K1Version,

				ArgoCDIngressURL:               fmt.Sprintf("https://argocd.%s", clctrl.DomainName),
				ArgoCDIngressNoHTTPSURL:        fmt.Sprintf("argocd.%s", clctrl.DomainName),
				ArgoWorkflowsIngressURL:        fmt.Sprintf("https://argo.%s", clctrl.DomainName),
				ArgoWorkflowsIngressNoHTTPSURL: fmt.Sprintf("argo.%s", clctrl.DomainName),
				AtlantisIngressURL:             fmt.Sprintf("https://atlantis.%s", clctrl.DomainName),
				AtlantisIngressNoHTTPSURL:      fmt.Sprintf("atlantis.%s", clctrl.DomainName),
				ChartMuseumIngressURL:          fmt.Sprintf("https://chartmuseum.%s", clctrl.DomainName),
				VaultIngressURL:                fmt.Sprintf("https://vault.%s", clctrl.DomainName),
				VaultIngressNoHTTPSURL:         fmt.Sprintf("vault.%s", clctrl.DomainName),
				VouchIngressURL:                fmt.Sprintf("https://vouch.%s", clctrl.DomainName),

				GitDescription:       fmt.Sprintf("%s hosted git", clctrl.GitProvider),
				GitNamespace:         "N/A",
				GitProvider:          clctrl.GitProvider,
				GitRunner:            fmt.Sprintf("%s Runner", clctrl.GitProvider),
				GitRunnerDescription: fmt.Sprintf("Self Hosted %s Runner", clctrl.GitProvider),
				GitRunnerNS:          fmt.Sprintf("%s-runner", clctrl.GitProvider),
				GitURL:               clctrl.GitopsTemplateURLFlag,

				GitHubHost:  fmt.Sprintf("https://github.com/%s/gitops.git", clctrl.GitOwner),
				GitHubOwner: clctrl.GitOwner,
				GitHubUser:  clctrl.GitUser,

				GitlabHost:         clctrl.GitHost,
				GitlabOwner:        clctrl.GitOwner,
				GitlabOwnerGroupID: clctrl.GitlabOwnerGroupID,
				GitlabUser:         clctrl.GitUser,

				GitOpsRepoAtlantisWebhookURL: clctrl.AtlantisWebhookURL,
				GitOpsRepoNoHTTPSURL:         fmt.Sprintf("%s.com/%s/gitops.git", clctrl.GitHost, clctrl.GitOwner),
				ClusterId:                    clctrl.ClusterID,
			}

			switch clctrl.GitProvider {
			case "github":
				DigitaloceanDestinationGitopsRepoGitURL = clctrl.ProviderConfig.(*digitalocean.DigitaloceanConfig).DestinationGitopsRepoGitURL
				DigitaloceanDestinationMetaphorRepoGitURL = clctrl.ProviderConfig.(*digitalocean.DigitaloceanConfig).DestinationMetaphorRepoGitURL
			case "gitlab":
				gitlabClient, err := gitlab.NewGitLabClient(clctrl.GitToken, clctrl.GitOwner)
				if err != nil {
					return err
				}
				// Format git url based on full path to group
				DigitaloceanDestinationGitopsRepoGitURL = fmt.Sprintf("git@gitlab.com:%s/gitops.git", gitlabClient.ParentGroupPath)
				DigitaloceanDestinationMetaphorRepoGitURL = fmt.Sprintf("git@gitlab.com:%s/metaphor.git", gitlabClient.ParentGroupPath)

			}
			gitopsTemplateTokens.(*digitalocean.GitOpsDirectoryValues).GitOpsRepoGitURL = DigitaloceanDestinationGitopsRepoGitURL
			gitopsTemplateTokens.(*digitalocean.GitOpsDirectoryValues).StateStoreBucketHostname = DigitaloceanStateStoreBucketName
		case "vultr":
			gitopsTemplateTokens = &vultr.GitOpsDirectoryValues{
				AlertsEmail:               clctrl.AlertsEmail,
				AtlantisAllowList:         fmt.Sprintf("%s/%s/*", clctrl.GitHost, clctrl.GitOwner),
				CloudProvider:             clctrl.CloudProvider,
				CloudRegion:               clctrl.CloudRegion,
				ClusterName:               clctrl.ClusterName,
				ClusterType:               clctrl.ClusterType,
				DomainName:                clctrl.DomainName,
				KubeconfigPath:            clctrl.ProviderConfig.(*vultr.VultrConfig).Kubeconfig,
				KubefirstStateStoreBucket: clctrl.KubefirstStateStoreBucketName,
				KubefirstTeam:             clctrl.KubefirstTeam,
				KubefirstVersion:          configs.K1Version,

				ArgoCDIngressURL:               fmt.Sprintf("https://argocd.%s", clctrl.DomainName),
				ArgoCDIngressNoHTTPSURL:        fmt.Sprintf("argocd.%s", clctrl.DomainName),
				ArgoWorkflowsIngressURL:        fmt.Sprintf("https://argo.%s", clctrl.DomainName),
				ArgoWorkflowsIngressNoHTTPSURL: fmt.Sprintf("argo.%s", clctrl.DomainName),
				AtlantisIngressURL:             fmt.Sprintf("https://atlantis.%s", clctrl.DomainName),
				AtlantisIngressNoHTTPSURL:      fmt.Sprintf("atlantis.%s", clctrl.DomainName),
				ChartMuseumIngressURL:          fmt.Sprintf("https://chartmuseum.%s", clctrl.DomainName),
				VaultIngressURL:                fmt.Sprintf("https://vault.%s", clctrl.DomainName),
				VaultIngressNoHTTPSURL:         fmt.Sprintf("vault.%s", clctrl.DomainName),
				VouchIngressURL:                fmt.Sprintf("https://vouch.%s", clctrl.DomainName),

				GitDescription:       fmt.Sprintf("%s hosted git", clctrl.GitProvider),
				GitNamespace:         "N/A",
				GitProvider:          clctrl.GitProvider,
				GitRunner:            fmt.Sprintf("%s Runner", clctrl.GitProvider),
				GitRunnerDescription: fmt.Sprintf("Self Hosted %s Runner", clctrl.GitProvider),
				GitRunnerNS:          fmt.Sprintf("%s-runner", clctrl.GitProvider),
				GitURL:               clctrl.GitopsTemplateURLFlag,

				GitHubHost:  fmt.Sprintf("https://github.com/%s/gitops.git", clctrl.GitOwner),
				GitHubOwner: clctrl.GitOwner,
				GitHubUser:  clctrl.GitUser,

				GitlabHost:         clctrl.GitHost,
				GitlabOwner:        clctrl.GitOwner,
				GitlabOwnerGroupID: clctrl.GitlabOwnerGroupID,
				GitlabUser:         clctrl.GitUser,

				GitOpsRepoAtlantisWebhookURL: clctrl.AtlantisWebhookURL,
				GitOpsRepoNoHTTPSURL:         fmt.Sprintf("%s.com/%s/gitops.git", clctrl.GitHost, clctrl.GitOwner),
				ClusterId:                    clctrl.ClusterID,
			}

			switch clctrl.GitProvider {
			case "github":
				VultrDestinationGitopsRepoGitURL = clctrl.ProviderConfig.(*vultr.VultrConfig).DestinationGitopsRepoGitURL
				VultrDestinationMetaphorRepoGitURL = clctrl.ProviderConfig.(*vultr.VultrConfig).DestinationMetaphorRepoGitURL
			case "gitlab":
				gitlabClient, err := gitlab.NewGitLabClient(clctrl.GitToken, clctrl.GitOwner)
				if err != nil {
					return err
				}
				// Format git url based on full path to group
				VultrDestinationGitopsRepoGitURL = fmt.Sprintf("git@gitlab.com:%s/gitops.git", gitlabClient.ParentGroupPath)
				VultrDestinationMetaphorRepoGitURL = fmt.Sprintf("git@gitlab.com:%s/metaphor.git", gitlabClient.ParentGroupPath)

			}
			gitopsTemplateTokens.(*vultr.GitOpsDirectoryValues).GitOpsRepoGitURL = VultrDestinationGitopsRepoGitURL
			gitopsTemplateTokens.(*vultr.GitOpsDirectoryValues).StateStoreBucketHostname = VultrStateStoreBucketHostname
		}

		return gitopsTemplateTokens
	case "metaphor":
		var metaphorTemplateTokens interface{}

		switch clctrl.CloudProvider {
		case "k3d":
			metaphorTemplateTokens = &k3d.MetaphorTokenValues{
				ClusterName:                   clctrl.ClusterName,
				CloudRegion:                   clctrl.CloudRegion,
				ContainerRegistryURL:          fmt.Sprintf("%s/%s/metaphor", clctrl.ContainerRegistryHost, clctrl.GitOwner),
				DomainName:                    clctrl.DomainName,
				MetaphorDevelopmentIngressURL: fmt.Sprintf("metaphor-development.%s", clctrl.DomainName),
				MetaphorStagingIngressURL:     fmt.Sprintf("metaphor-staging.%s", clctrl.DomainName),
				MetaphorProductionIngressURL:  fmt.Sprintf("metaphor-production.%s", clctrl.DomainName),
			}
		case "civo":
			metaphorTemplateTokens = &civo.MetaphorTokenValues{
				ClusterName:                   clctrl.ClusterName,
				CloudRegion:                   clctrl.CloudRegion,
				ContainerRegistryURL:          fmt.Sprintf("%s/%s/metaphor", clctrl.ContainerRegistryHost, clctrl.GitOwner),
				DomainName:                    clctrl.DomainName,
				MetaphorDevelopmentIngressURL: fmt.Sprintf("metaphor-development.%s", clctrl.DomainName),
				MetaphorStagingIngressURL:     fmt.Sprintf("metaphor-staging.%s", clctrl.DomainName),
				MetaphorProductionIngressURL:  fmt.Sprintf("metaphor-production.%s", clctrl.DomainName),
			}
		case "digitalocean":
			metaphorTemplateTokens = &digitalocean.MetaphorTokenValues{
				ClusterName:                   clctrl.ClusterName,
				CloudRegion:                   clctrl.CloudRegion,
				ContainerRegistryURL:          fmt.Sprintf("%s/%s/metaphor", clctrl.ContainerRegistryHost, clctrl.GitOwner),
				DomainName:                    clctrl.DomainName,
				MetaphorDevelopmentIngressURL: fmt.Sprintf("metaphor-development.%s", clctrl.DomainName),
				MetaphorStagingIngressURL:     fmt.Sprintf("metaphor-staging.%s", clctrl.DomainName),
				MetaphorProductionIngressURL:  fmt.Sprintf("metaphor-production.%s", clctrl.DomainName),
			}
		case "vultr":
			metaphorTemplateTokens = &vultr.MetaphorTokenValues{
				ClusterName:                   clctrl.ClusterName,
				CloudRegion:                   clctrl.CloudRegion,
				ContainerRegistryURL:          fmt.Sprintf("%s/%s/metaphor", clctrl.ContainerRegistryHost, clctrl.GitOwner),
				DomainName:                    clctrl.DomainName,
				MetaphorDevelopmentIngressURL: fmt.Sprintf("metaphor-development.%s", clctrl.DomainName),
				MetaphorStagingIngressURL:     fmt.Sprintf("metaphor-staging.%s", clctrl.DomainName),
				MetaphorProductionIngressURL:  fmt.Sprintf("metaphor-production.%s", clctrl.DomainName),
			}
		}

		return metaphorTemplateTokens
	}

	return nil
}

// ClusterSecretsBootstrap
func (clctrl *ClusterController) ClusterSecretsBootstrap() error {
	cl, err := clctrl.MdbCl.GetCluster(clctrl.ClusterName)
	if err != nil {
		return err
	}

	if !cl.ClusterSecretsCreatedCheck {
		switch clctrl.CloudProvider {
		case "k3d":
			kcfg := k8s.CreateKubeConfig(false, clctrl.ProviderConfig.(k3d.K3dConfig).Kubeconfig)
			err := k3d.GenerateTLSSecrets(kcfg.Clientset, clctrl.ProviderConfig.(k3d.K3dConfig))
			if err != nil {
				return err
			}

			err = k3d.AddK3DSecrets(
				clctrl.AtlantisWebhookSecret,
				clctrl.PublicKey,
				clctrl.ProviderConfig.(k3d.K3dConfig).DestinationGitopsRepoGitURL,
				clctrl.PrivateKey,
				false,
				clctrl.GitProvider,
				clctrl.GitUser,
				clctrl.GitOwner,
				clctrl.ProviderConfig.(k3d.K3dConfig).Kubeconfig,
				clctrl.GitToken,
			)
			if err != nil {
				log.Info("Error adding kubernetes secrets for bootstrap")
				return err
			}
		case "civo":
			err := civoext.BootstrapCivoMgmtCluster(false, clctrl.ProviderConfig.(*civo.CivoConfig).Kubeconfig, &cl)
			if err != nil {
				log.Info("Error adding kubernetes secrets for bootstrap")
				return err
			}
		case "digitalocean":
			err := digitaloceanext.BootstrapDigitaloceanMgmtCluster(false, clctrl.ProviderConfig.(*digitalocean.DigitaloceanConfig).Kubeconfig, &cl)
			if err != nil {
				log.Info("Error adding kubernetes secrets for bootstrap")
				return err
			}
		case "vultr":
			err := vultrext.BootstrapVultrMgmtCluster(false, clctrl.ProviderConfig.(*vultr.VultrConfig).Kubeconfig, &cl)
			if err != nil {
				log.Info("Error adding kubernetes secrets for bootstrap")
				return err
			}
		}

		err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "cluster_secrets_created_check", true)
		if err != nil {
			return err
		}
	}

	return nil
}

// ContainerRegistryAuth
func (clctrl *ClusterController) ContainerRegistryAuth() (string, error) {
	var kcfg *k8s.KubernetesClient

	switch clctrl.CloudProvider {
	case "k3d":
		kcfg = k8s.CreateKubeConfig(false, clctrl.ProviderConfig.(k3d.K3dConfig).Kubeconfig)
	case "civo":
		kcfg = k8s.CreateKubeConfig(false, clctrl.ProviderConfig.(*civo.CivoConfig).Kubeconfig)
	case "digitalocean":
		kcfg = k8s.CreateKubeConfig(false, clctrl.ProviderConfig.(*digitalocean.DigitaloceanConfig).Kubeconfig)
	case "vultr":
		kcfg = k8s.CreateKubeConfig(false, clctrl.ProviderConfig.(*vultr.VultrConfig).Kubeconfig)
	}

	// Container registry authentication creation
	containerRegistryAuth := gitShim.ContainerRegistryAuth{
		GitProvider:           clctrl.GitProvider,
		GitUser:               clctrl.GitUser,
		GitToken:              clctrl.GitToken,
		GitlabGroupFlag:       clctrl.GitOwner,
		GithubOwner:           clctrl.GitOwner,
		ContainerRegistryHost: clctrl.ContainerRegistryHost,
		Clientset:             kcfg.Clientset,
	}
	containerRegistryAuthToken, err := gitShim.CreateContainerRegistrySecret(&containerRegistryAuth)
	if err != nil {
		return "", err
	}

	return containerRegistryAuthToken, nil
}
