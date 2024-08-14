/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package aws

import (
	"fmt"
	"strconv"
	"time"

	awsext "github.com/kubefirst/kubefirst-api/extensions/aws"
	terraformext "github.com/kubefirst/kubefirst-api/extensions/terraform"
	pkg "github.com/kubefirst/kubefirst-api/internal"
	"github.com/kubefirst/kubefirst-api/internal/argocd"
	awsinternal "github.com/kubefirst/kubefirst-api/internal/aws"
	"github.com/kubefirst/kubefirst-api/internal/constants"
	"github.com/kubefirst/kubefirst-api/internal/errors"
	gitlab "github.com/kubefirst/kubefirst-api/internal/gitlab"
	"github.com/kubefirst/kubefirst-api/internal/httpCommon"
	"github.com/kubefirst/kubefirst-api/internal/k8s"
	"github.com/kubefirst/kubefirst-api/internal/secrets"
	"github.com/kubefirst/kubefirst-api/internal/utils"
	"github.com/kubefirst/kubefirst-api/pkg/providerConfigs"
	pkgtypes "github.com/kubefirst/kubefirst-api/pkg/types"
	"github.com/kubefirst/metrics-client/pkg/telemetry"
	log "github.com/rs/zerolog/log"
)

// DeleteAWSCluster
func DeleteAWSCluster(cl *pkgtypes.Cluster, telemetryEvent telemetry.TelemetryEvent) error {
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
		return fmt.Errorf("error getting provider config for cluster %s: %w", cl.ClusterName, err)
	}

	kcfg := utils.GetKubernetesClient(cl.ClusterName)

	cl.Status = constants.ClusterStatusDeleting

	if err := secrets.UpdateCluster(kcfg.Clientset, *cl); err != nil {
		return fmt.Errorf("error updating cluster status for cluster %s: %w", cl.ClusterName, err)
	}

	switch cl.GitProvider {
	case "github":
		if cl.GitTerraformApplyCheck {
			log.Info().Msg("destroying github resources with terraform")

			tfEntrypoint := config.GitopsDir + "/terraform/github"
			tfEnvs := map[string]string{}
			tfEnvs = awsext.GetAwsTerraformEnvs(tfEnvs, cl)
			tfEnvs = awsext.GetGithubTerraformEnvs(tfEnvs, cl)
			err := terraformext.InitDestroyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs)
			if err != nil {
				log.Error().Msgf("error executing terraform destroy %s", tfEntrypoint)
				errors.HandleClusterError(cl, err.Error())
				return fmt.Errorf("failed to execute terraform destroy for GitHub resources at %s: %w", tfEntrypoint, err)
			}
			log.Info().Msg("github resources terraform destroyed")

			kcfg := utils.GetKubernetesClient(cl.ClusterName)

			cl.GitTerraformApplyCheck = false
			err = secrets.UpdateCluster(kcfg.Clientset, *cl)
			if err != nil {
				return fmt.Errorf("error updating cluster after destroying github resources for cluster %s: %w", cl.ClusterName, err)
			}
		}
	case "gitlab":
		if cl.GitTerraformApplyCheck {
			log.Info().Msg("destroying gitlab resources with terraform")
			gitlabClient, err := gitlab.NewGitLabClient(cl.GitAuth.Token, cl.GitAuth.Owner)
			if err != nil {
				return fmt.Errorf("error creating gitlab client for cluster %s: %w", cl.ClusterName, err)
			}

			// Before removing Terraform resources, remove any container registry repositories
			// since failing to remove them beforehand will result in an apply failure
			projectsForDeletion := []string{"gitops", "metaphor"}
			for _, project := range projectsForDeletion {
				projectExists, err := gitlabClient.CheckProjectExists(project)
				if err != nil {
					log.Error().Msgf("could not check for existence of project %s: %s", project, err)
				}
				if projectExists {
					log.Info().Msgf("checking project %s for container registries...", project)
					crr, err := gitlabClient.GetProjectContainerRegistryRepositories(project)
					if err != nil {
						log.Error().Msgf("could not retrieve container registry repositories: %s", err)
					}
					if len(crr) > 0 {
						for _, cr := range crr {
							err := gitlabClient.DeleteContainerRegistryRepository(project, cr.ID)
							if err != nil {
								log.Error().Msgf("error deleting container registry repository: %s", err)
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
			tfEnvs = awsext.GetAwsTerraformEnvs(tfEnvs, cl)
			tfEnvs = awsext.GetGitlabTerraformEnvs(tfEnvs, gitlabClient.ParentGroupID, cl)
			err = terraformext.InitDestroyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs)
			if err != nil {
				log.Error().Msgf("error executing terraform destroy %s", tfEntrypoint)
				errors.HandleClusterError(cl, err.Error())
				return fmt.Errorf("failed to execute terraform destroy for GitLab resources at %s: %w", tfEntrypoint, err)
			}

			log.Info().Msg("gitlab resources terraform destroyed")

			cl.GitTerraformApplyCheck = false
			err = secrets.UpdateCluster(kcfg.Clientset, *cl)
			if err != nil {
				return fmt.Errorf("error updating cluster after destroying gitlab resources for cluster %s: %w", cl.ClusterName, err)
			}
		}
	}

	if cl.CloudTerraformApplyCheck || cl.CloudTerraformApplyFailedCheck {
		if !cl.ArgoCDDeleteRegistryCheck {
			conf, err := awsinternal.NewAwsV3(
				cl.CloudRegion,
				cl.AWSAuth.AccessKeyID,
				cl.AWSAuth.SecretAccessKey,
				cl.AWSAuth.SessionToken,
			)
			if err != nil {
				errors.HandleClusterError(cl, err.Error())
				return fmt.Errorf("error creating aws client for cluster %s: %w", cl.ClusterName, err)
			}

			awsClient := &awsinternal.Configuration{
				Config: conf,
			}
			kcfg := awsext.CreateEKSKubeconfig(&awsClient.Config, cl.ClusterName)

			log.Info().Msg("destroying aws resources with terraform")

			// Only port-forward to ArgoCD and delete registry if ArgoCD was installed
			if cl.ArgoCDInstallCheck {
				removeArgoCDApps := []string{"ingress-nginx-components", "ingress-nginx"}
				err = argocd.ApplicationCleanup(kcfg.Clientset, removeArgoCDApps)
				if err != nil {
					log.Error().Msgf("encountered error during argocd application cleanup: %s", err)
				}

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
					return fmt.Errorf("error reading argocd secret for cluster %s: %w", cl.ClusterName, err)
				}
				argocdPassword := secData["password"]

				argocdAuthToken, err := argocd.GetArgoCDToken("admin", argocdPassword)
				if err != nil {
					return fmt.Errorf("error getting argocd token for cluster %s: %w", cl.ClusterName, err)
				}

				log.Info().Msgf("port-forward to argocd is available at %s", providerConfigs.ArgocdPortForwardURL)

				client := httpCommon.CustomHTTPClient(true)
				log.Info().Msg("deleting the registry application")
				httpCode, _, err := argocd.DeleteApplication(client, config.RegistryAppName, argocdAuthToken, "true")
				if err != nil {
					errors.HandleClusterError(cl, err.Error())
					return fmt.Errorf("failed to delete ArgoCD application %s for cluster %s: %w", config.RegistryAppName, cl.ClusterName, err)
				}
				log.Info().Msgf("http status code %d", httpCode)
			}

			// Pause before cluster destroy to prevent a race condition
			log.Info().Msg("waiting for aws Kubernetes cluster resource removal to finish...")
			time.Sleep(time.Second * 10)

			cl.ArgoCDDeleteRegistryCheck = true
			err = secrets.UpdateCluster(kcfg.Clientset, *cl)
			if err != nil {
				return fmt.Errorf("error updating cluster after ArgoCD cleanup for cluster %s: %w", cl.ClusterName, err)
			}
		}

		log.Info().Msg("destroying aws cloud resources")
		tfEntrypoint := config.GitopsDir + fmt.Sprintf("/terraform/%s", cl.CloudProvider)
		tfEnvs := map[string]string{}
		tfEnvs = awsext.GetAwsTerraformEnvs(tfEnvs, cl)
		tfEnvs["TF_VAR_aws_account_id"] = cl.AWSAccountID

		switch cl.GitProvider {
		case "github":
			tfEnvs = awsext.GetGithubTerraformEnvs(tfEnvs, cl)
		case "gitlab":
			gid, err := strconv.Atoi(fmt.Sprint(cl.GitlabOwnerGroupID))
			if err != nil {
				return fmt.Errorf("couldn't convert gitlab group id to int for cluster %s: %w", cl.ClusterName, err)
			}
			tfEnvs = awsext.GetGitlabTerraformEnvs(tfEnvs, gid, cl)
		}
		err = terraformext.InitDestroyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs)
		if err != nil {
			log.Error().Msgf("error executing terraform destroy %s", tfEntrypoint)
			errors.HandleClusterError(cl, err.Error())
			return fmt.Errorf("failed to execute terraform destroy for AWS resources at %s: %w", tfEntrypoint, err)
		}
		log.Info().Msg("aws resources terraform destroyed")

		cl.CloudTerraformApplyCheck = false
		err = secrets.UpdateCluster(kcfg.Clientset, *cl)
		if err != nil {
			return fmt.Errorf("error updating cluster after destroying aws resources for cluster %s: %w", cl.ClusterName, err)
		}

		cl.CloudTerraformApplyFailedCheck = false
		err = secrets.UpdateCluster(kcfg.Clientset, *cl)
		if err != nil {
			return fmt.Errorf("error updating cluster after marking aws apply as failed for cluster %s: %w", cl.ClusterName, err)
		}
	}

	// remove ssh key provided one was created
	if cl.GitProvider == "gitlab" {
		gitlabClient, err := gitlab.NewGitLabClient(cl.GitAuth.Token, cl.GitAuth.Owner)
		if err != nil {
			return fmt.Errorf("error creating gitlab client for SSH key deletion for cluster %s: %w", cl.ClusterName, err)
		}
		log.Info().Msgf("attempting to delete managed ssh key...")
		err = gitlabClient.DeleteUserSSHKey("kbot-ssh-key")
		if err != nil {
			log.Warn().Msgf("error deleting SSH key for cluster %s: %s", cl.ClusterName, err)
		}
	}

	telemetry.SendEvent(telemetryEvent, telemetry.ClusterDeleteCompleted, "")

	cl.Status = constants.ClusterStatusDeleted
	err = secrets.UpdateCluster(kcfg.Clientset, *cl)
	if err != nil {
		return fmt.Errorf("error updating cluster status to deleted for cluster %s: %w", cl.ClusterName, err)
	}

	err = pkg.ResetK1Dir(config.K1Dir)
	if err != nil {
		return fmt.Errorf("error resetting k1 directory for cluster %s: %w", cl.ClusterName, err)
	}

	return nil
}
