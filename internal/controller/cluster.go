/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	"fmt"

	gitShim "github.com/kubefirst/kubefirst-api/internal/gitShim"
	"github.com/kubefirst/runtime/configs"
	"github.com/kubefirst/runtime/pkg/k3d"
	"github.com/kubefirst/runtime/pkg/k8s"
	log "github.com/sirupsen/logrus"
)

// CreateK3DCluster
func (clctrl *ClusterController) CreateK3DCluster() error {
	cl, err := clctrl.MdbCl.GetCluster(clctrl.ClusterName)
	if err != nil {
		return err
	}

	if !cl.CloudTerraformApplyCheck || cl.CloudTerraformApplyFailedCheck {
		// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricCloudTerraformApplyStarted, "")

		log.Info("Creating k3d cluster")

		err := k3d.ClusterCreate(clctrl.ClusterName, clctrl.ProviderConfig.K1Dir, clctrl.ProviderConfig.K3dClient, clctrl.ProviderConfig.Kubeconfig)
		if err != nil {
			msg := fmt.Sprintf("error creating k3d resources: %s", err)
			// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricCloudTerraformApplyFailed, msg)
			err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "cloud_terraform_apply_failed_check", true)
			if err != nil {
				return err
			}
			return fmt.Errorf(msg)
		}

		log.Info("successfully created k3d cluster")

		// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricCloudTerraformApplyCompleted, "")

		err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "cloud_terraform_apply_check", true)
		if err != nil {
			return err
		}
	}

	return nil
}

// CreateTokens
func (clctrl *ClusterController) CreateTokens(kind string) interface{} {
	switch kind {
	case "gitops":
		gitopsTemplateTokens := &k3d.GitopsTokenValues{
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
			KubeconfigPath:                clctrl.ProviderConfig.Kubeconfig,
			GitopsRepoGitURL:              clctrl.ProviderConfig.DestinationGitopsRepoGitURL,
			GitProvider:                   clctrl.ProviderConfig.GitProvider,
			ClusterId:                     clctrl.ClusterID,
			CloudProvider:                 clctrl.CloudProvider,
		}

		return gitopsTemplateTokens
	case "metaphor":
		metaphorTemplateTokens := &k3d.MetaphorTokenValues{
			ClusterName:                   clctrl.ClusterName,
			CloudRegion:                   clctrl.CloudRegion,
			ContainerRegistryURL:          fmt.Sprintf("%s/%s/metaphor", clctrl.ContainerRegistryHost, clctrl.GitOwner),
			DomainName:                    clctrl.DomainName,
			MetaphorDevelopmentIngressURL: fmt.Sprintf("metaphor-development.%s", clctrl.DomainName),
			MetaphorStagingIngressURL:     fmt.Sprintf("metaphor-staging.%s", clctrl.DomainName),
			MetaphorProductionIngressURL:  fmt.Sprintf("metaphor-production.%s", clctrl.DomainName),
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
		kcfg := k8s.CreateKubeConfig(false, clctrl.ProviderConfig.Kubeconfig)
		err := k3d.GenerateTLSSecrets(kcfg.Clientset, *clctrl.ProviderConfig)
		if err != nil {
			return err
		}

		err = k3d.AddK3DSecrets(
			clctrl.AtlantisWebhookSecret,
			clctrl.PublicKey,
			clctrl.ProviderConfig.DestinationGitopsRepoGitURL,
			clctrl.PrivateKey,
			false,
			clctrl.GitProvider,
			clctrl.GitUser,
			clctrl.GitOwner,
			clctrl.ProviderConfig.Kubeconfig,
			clctrl.GitToken,
		)
		if err != nil {
			log.Info("Error adding kubernetes secrets for bootstrap")
			return err
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
	kcfg := k8s.CreateKubeConfig(false, clctrl.ProviderConfig.Kubeconfig)
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
