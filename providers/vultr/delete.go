/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package vultr

import (
	"context"
	"fmt"
	"time"

	terraformext "github.com/kubefirst/kubefirst-api/extensions/terraform"
	vultrext "github.com/kubefirst/kubefirst-api/extensions/vultr"
	runtime "github.com/kubefirst/kubefirst-api/internal"
	"github.com/kubefirst/kubefirst-api/internal/argocd"
	"github.com/kubefirst/kubefirst-api/internal/constants"
	"github.com/kubefirst/kubefirst-api/internal/errors"
	gitlab "github.com/kubefirst/kubefirst-api/internal/gitlab"
	"github.com/kubefirst/kubefirst-api/internal/httpCommon"
	"github.com/kubefirst/kubefirst-api/internal/k8s"
	"github.com/kubefirst/kubefirst-api/internal/secrets"
	"github.com/kubefirst/kubefirst-api/internal/utils"
	"github.com/kubefirst/kubefirst-api/internal/vultr"
	"github.com/kubefirst/kubefirst-api/pkg/providerConfigs"
	pkgtypes "github.com/kubefirst/kubefirst-api/pkg/types"
	"github.com/kubefirst/metrics-client/pkg/telemetry"
	log "github.com/rs/zerolog/log"
)

// DeleteVultrCluster
func DeleteVultrCluster(cl *pkgtypes.Cluster, telemetryEvent telemetry.TelemetryEvent) error {
	telemetry.SendEvent(telemetryEvent, telemetry.ClusterDeleteStarted, "")

	// Instantiate provider config
	config, err := providerConfigs.GetConfig(
		cl.ClusterName,
		cl.DomainName,
		cl.GitProvider,
		cl.GitAuth.Owner,
		cl.GitProtocol,
		cl.CloudflareAuth.APIToken,
		cl.CloudflareAuth.OriginCaIssuerKey,
	)
	if err != nil {
		return fmt.Errorf("error getting provider config for cluster %q: %w", cl.ClusterName, err)
	}

	kcfg := utils.GetKubernetesClient(cl.ClusterName)

	cl.Status = constants.ClusterStatusDeleting

	if err := secrets.UpdateCluster(kcfg.Clientset, *cl); err != nil {
		return fmt.Errorf("error updating cluster secrets for cluster %q: %w", cl.ClusterName, err)
	}

	switch cl.GitProvider {
	case "github":
		if cl.GitTerraformApplyCheck {
			log.Info().Msg("destroying github resources with terraform")

			tfEntrypoint := config.GitopsDir + "/terraform/github"
			tfEnvs := map[string]string{}
			tfEnvs = vultrext.GetVultrTerraformEnvs(tfEnvs, cl)
			tfEnvs = vultrext.GetGithubTerraformEnvs(tfEnvs, cl)
			err := terraformext.InitDestroyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs)
			if err != nil {
				log.Printf("error executing terraform destroy %s", tfEntrypoint)
				errors.HandleClusterError(cl, err.Error())
				return fmt.Errorf("error executing terraform destroy %q: %w", tfEntrypoint, err)
			}
			log.Info().Msg("github resources terraform destroyed")

			cl.GitTerraformApplyCheck = false
			err = secrets.UpdateCluster(kcfg.Clientset, *cl)
			if err != nil {
				return fmt.Errorf("error updating cluster secrets after destroying github resources for cluster %q: %w", cl.ClusterName, err)
			}
		}
	case "gitlab":
		if cl.GitTerraformApplyCheck {
			log.Info().Msg("destroying gitlab resources with terraform")
			gitlabClient, err := gitlab.NewGitLabClient(cl.GitAuth.Token, cl.GitAuth.Owner)
			if err != nil {
				return fmt.Errorf("error creating gitlab client for cluster %q: %w", cl.ClusterName, err)
			}

			// Before removing Terraform resources, remove any container registry repositories
			// since failing to remove them beforehand will result in an apply failure
			projectsForDeletion := []string{"gitops", "metaphor"}
			for _, project := range projectsForDeletion {
				projectExists, err := gitlabClient.CheckProjectExists(project)
				if err != nil {
					log.Error().Msgf("could not check for existence of project %s: %s", project, err)
					return fmt.Errorf("could not check for existence of project %q: %w", project, err)
				}
				if projectExists {
					log.Info().Msgf("checking project %s for container registries...", project)
					crr, err := gitlabClient.GetProjectContainerRegistryRepositories(project)
					if err != nil {
						log.Error().Msgf("could not retrieve container registry repositories: %s", err)
						return fmt.Errorf("could not retrieve container registry repositories for project %q: %w", project, err)
					}
					if len(crr) > 0 {
						for _, cr := range crr {
							err := gitlabClient.DeleteContainerRegistryRepository(project, cr.ID)
							if err != nil {
								log.Error().Msgf("error deleting container registry repository: %s", err)
								return fmt.Errorf("error deleting container registry repository for project %q: %w", project, err)
							}
						}
					} else {
						log.Info().Msgf("project %s does not have any container registries, skipping", project)
					}
				} else {
					log.Info().Msgf("project %s does not exist, skipping", project)
				}
			}

			tfEntrypoint := config.GitopsDir + "/terraform/gitlab"
			tfEnvs := map[string]string{}
			tfEnvs = vultrext.GetVultrTerraformEnvs(tfEnvs, cl)
			tfEnvs = vultrext.GetGitlabTerraformEnvs(tfEnvs, gitlabClient.ParentGroupID, cl)
			err = terraformext.InitDestroyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs)
			if err != nil {
				log.Info().Msgf("error executing terraform destroy %s", tfEntrypoint)
				errors.HandleClusterError(cl, err.Error())
				return fmt.Errorf("error executing terraform destroy %q: %w", tfEntrypoint, err)
			}

			log.Info().Msg("gitlab resources terraform destroyed")

			cl.GitTerraformApplyCheck = false
			err = secrets.UpdateCluster(kcfg.Clientset, *cl)
			if err != nil {
				return fmt.Errorf("error updating cluster secrets after destroying gitlab resources for cluster %q: %w", cl.ClusterName, err)
			}
		}
	}

	if !cl.ArgoCDDeleteRegistryCheck {
		kcfg := k8s.CreateKubeConfig(false, config.Kubeconfig)

		// Remove applications with external dependencies
		removeArgoCDApps := []string{
			"ingress-nginx-components",
			"ingress-nginx",
			"argo-components",
			"argo",
			"atlantis-components",
			"atlantis",
			"vault-components",
			"vault",
		}
		err = argocd.ApplicationCleanup(kcfg.Clientset, removeArgoCDApps)
		if err != nil {
			log.Error().Msgf("encountered error during argocd application cleanup: %s", err)
			return fmt.Errorf("encountered error during argocd application cleanup for cluster %q: %w", cl.ClusterName, err)
		}
		// Pause before cluster destroy to prevent a race condition
		log.Info().Msg("waiting for argocd application deletion to complete...")
		time.Sleep(time.Second * 20)
	}

	// GetKubernetesAssociatedBlockStorage
	vultrConf := vultr.Configuration{
		Client:  vultr.NewVultr(cl.VultrAuth.Token),
		Context: context.Background(),
	}
	blockStorage, err := vultrConf.GetKubernetesAssociatedBlockStorage("", true)
	if err != nil {
		return fmt.Errorf("error getting associated block storage for cluster %q: %w", cl.ClusterName, err)
	}

	if cl.CloudTerraformApplyCheck || cl.CloudTerraformApplyFailedCheck {
		if !cl.CloudTerraformApplyFailedCheck {
			kcfg := k8s.CreateKubeConfig(false, config.Kubeconfig)

			log.Info().Msg("destroying vultr resources with terraform")

			// Only port-forward to ArgoCD and delete registry if ArgoCD was installed
			if !cl.ArgoCDDeleteRegistryCheck {
				log.Info().Msg("opening argocd port forward")
				// * ArgoCD port-forward
				argoCDStopChannel := make(chan struct{}, 1)
				defer func() {
					close(argoCDStopChannel)
				}()
				k8s.OpenPortForwardPodWrapper(
					kcfg.Clientset,
					kcfg.RestConfig,
					"argocd-server",
					"argocd",
					80,
					8080,
					argoCDStopChannel,
				)

				log.Info().Msg("getting new auth token for argocd")

				secData, err := k8s.ReadSecretV2(kcfg.Clientset, "argocd", "argocd-initial-admin-secret")
				if err != nil {
					return fmt.Errorf("error reading argocd secret for cluster %q: %w", cl.ClusterName, err)
				}
				argocdPassword := secData["password"]

				argocdAuthToken, err := argocd.GetArgoCDToken("admin", argocdPassword)
				if err != nil {
					return fmt.Errorf("error getting argocd token for cluster %q: %w", cl.ClusterName, err)
				}

				log.Info().Msgf("port-forward to argocd is available at %s", providerConfigs.ArgocdPortForwardURL)

				client := httpCommon.CustomHTTPClient(true)
				log.Info().Msg("deleting the registry application")
				httpCode, _, err := argocd.DeleteApplication(client, config.RegistryAppName, argocdAuthToken, "true")
				if err != nil {
					return fmt.Errorf("error deleting registry application for cluster %q: %w", cl.ClusterName, err)
				}
				log.Info().Msgf("http status code %d", httpCode)
			}

			// Pause before cluster destroy to prevent a race condition
			log.Info().Msg("waiting for vultr kubernetes cluster resource removal to finish...")
			time.Sleep(time.Second * 10)

			cl.ArgoCDDeleteRegistryCheck = true
			err = secrets.UpdateCluster(kcfg.Clientset, *cl)
			if err != nil {
				return fmt.Errorf("error updating cluster secrets after waiting for resource removal for cluster %q: %w", cl.ClusterName, err)
			}
		}

		log.Info().Msg("destroying vultr cloud resources")
		tfEntrypoint := config.GitopsDir + fmt.Sprintf("/terraform/%s", cl.CloudProvider)
		tfEnvs := map[string]string{}
		tfEnvs = vultrext.GetVultrTerraformEnvs(tfEnvs, cl)

		switch cl.GitProvider {
		case "github":
			tfEnvs = vultrext.GetGithubTerraformEnvs(tfEnvs, cl)
		case "gitlab":
			tfEnvs = vultrext.GetGitlabTerraformEnvs(tfEnvs, cl.GitlabOwnerGroupID, cl)
		}
		err = terraformext.InitDestroyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs)
		if err != nil {
			log.Printf("error executing terraform destroy %s", tfEntrypoint)
			errors.HandleClusterError(cl, err.Error())
			return fmt.Errorf("error executing terraform destroy %q: %w", tfEntrypoint, err)
		}
		log.Info().Msg("vultr resources terraform destroyed")

		cl.CloudTerraformApplyCheck = false
		cl.CloudTerraformApplyFailedCheck = false
		err = secrets.UpdateCluster(kcfg.Clientset, *cl)
		if err != nil {
			return fmt.Errorf("error updating cluster secrets after destroying vultr resources for cluster %q: %w", cl.ClusterName, err)
		}
	}

	// Remove hanging volumes
	// This fails with regularity if done too quickly
	time.Sleep(time.Second * 45)
	err = vultrConf.DeleteBlockStorage(blockStorage)
	if err != nil {
		return fmt.Errorf("error deleting block storage for cluster %q: %w", cl.ClusterName, err)
	}

	// remove ssh key provided one was created
	if cl.GitProvider == "gitlab" {
		gitlabClient, err := gitlab.NewGitLabClient(cl.GitAuth.Token, cl.GitAuth.Owner)
		if err != nil {
			return fmt.Errorf("error creating gitlab client for deleting ssh key for cluster %q: %w", cl.ClusterName, err)
		}

		log.Info().Msg("attempting to delete managed ssh key...")
		err = gitlabClient.DeleteUserSSHKey("kbot-ssh-key")
		if err != nil {
			log.Error().Msg(err.Error())
			return fmt.Errorf("error deleting managed ssh key for cluster %q: %w", cl.ClusterName, err)
		}
	}

	telemetry.SendEvent(telemetryEvent, telemetry.ClusterDeleteCompleted, "")

	cl.Status = constants.ClusterStatusDeleted
	err = secrets.UpdateCluster(kcfg.Clientset, *cl)
	if err != nil {
		return fmt.Errorf("error updating cluster status for cluster %q: %w", cl.ClusterName, err)
	}

	err = runtime.ResetK1Dir(config.K1Dir)
	if err != nil {
		return fmt.Errorf("error resetting k1 directory for cluster %q: %w", cl.ClusterName, err)
	}

	return nil
}
