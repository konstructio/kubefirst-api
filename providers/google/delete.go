/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package google

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strconv"
	"time"

	googleext "github.com/kubefirst/kubefirst-api/extensions/google"
	terraformext "github.com/kubefirst/kubefirst-api/extensions/terraform"
	"github.com/kubefirst/kubefirst-api/internal/constants"
	"github.com/kubefirst/kubefirst-api/internal/errors"
	"github.com/kubefirst/kubefirst-api/internal/secrets"
	"github.com/kubefirst/kubefirst-api/internal/utils"
	"github.com/kubefirst/kubefirst-api/pkg/google"
	"github.com/kubefirst/kubefirst-api/pkg/providerConfigs"
	pkgtypes "github.com/kubefirst/kubefirst-api/pkg/types"
	"github.com/kubefirst/metrics-client/pkg/telemetry"
	"github.com/kubefirst/runtime/pkg"
	"github.com/kubefirst/runtime/pkg/argocd"
	gitlab "github.com/kubefirst/runtime/pkg/gitlab"
	"github.com/kubefirst/runtime/pkg/k8s"
	log "github.com/rs/zerolog/log"
)

// DeleteGoogleCluster
func DeleteGoogleCluster(cl *pkgtypes.Cluster, telemetryEvent telemetry.TelemetryEvent) error {

	// Instantiate google config
	config := providerConfigs.GetConfig(cl.ClusterName, cl.DomainName, cl.GitProvider, cl.GitAuth.Owner, cl.GitProtocol, cl.CloudflareAuth.Token, "")

	kcfg := utils.GetKubernetesClient(cl.ClusterName)

	cl.Status = constants.ClusterStatusDeleting
	err := secrets.UpdateCluster(kcfg.Clientset, *cl)
	if err != nil {
		return err
	}

	switch cl.GitProvider {
	case "github":
		if cl.GitTerraformApplyCheck {
			log.Info().Msg("destroying github resources with terraform")

			tfEntrypoint := config.GitopsDir + "/terraform/github"
			tfEnvs := map[string]string{}
			tfEnvs = googleext.GetGoogleTerraformEnvs(tfEnvs, cl)
			tfEnvs = googleext.GetGithubTerraformEnvs(tfEnvs, cl)
			err := terraformext.InitDestroyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs)
			if err != nil {
				log.Error().Msgf("error executing terraform destroy %s", tfEntrypoint)
				errors.HandleClusterError(cl, err.Error())
				return err
			}
			log.Info().Msg("github resources terraform destroyed")

			cl.GitTerraformApplyCheck = false
			err = secrets.UpdateCluster(kcfg.Clientset, *cl)
			if err != nil {
				return err
			}
		}
	case "gitlab":
		if cl.GitTerraformApplyCheck {
			log.Info().Msg("destroying gitlab resources with terraform")
			gitlabClient, err := gitlab.NewGitLabClient(cl.GitAuth.Token, cl.GitAuth.Owner)
			if err != nil {
				return err
			}

			// Before removing Terraform resources, remove any container registry repositories
			// since failing to remove them beforehand will result in an apply failure
			var projectsForDeletion = []string{"gitops", "metaphor"}
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
			tfEnvs = googleext.GetGoogleTerraformEnvs(tfEnvs, cl)
			tfEnvs = googleext.GetGitlabTerraformEnvs(tfEnvs, gitlabClient.ParentGroupID, cl)
			err = terraformext.InitDestroyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs)
			if err != nil {
				log.Error().Msgf("error executing terraform destroy %s", tfEntrypoint)
				errors.HandleClusterError(cl, err.Error())
				return err
			}

			log.Info().Msg("gitlab resources terraform destroyed")

			cl.GitTerraformApplyCheck = false
			err = secrets.UpdateCluster(kcfg.Clientset, *cl)
			if err != nil {
				return err
			}
		}
	}

	if cl.CloudTerraformApplyCheck || cl.CloudTerraformApplyFailedCheck {
		if !cl.ArgoCDDeleteRegistryCheck {
			googleConf := google.GoogleConfiguration{
				Context: context.Background(),
				Project: cl.GoogleAuth.ProjectId,
				Region:  cl.CloudRegion,
			}
			kcfg, _ := googleConf.GetContainerClusterAuth(cl.ClusterName, []byte(cl.GoogleAuth.KeyFile))

			log.Info().Msg("destroying google resources with terraform")

			// Only port-forward to ArgoCD and delete registry if ArgoCD was installed
			if cl.ArgoCDInstallCheck {
				removeArgoCDApps := []string{"ingress-nginx-components", "ingress-nginx"}
				err = argocd.ArgoCDApplicationCleanup(kcfg.Clientset, removeArgoCDApps)
				if err != nil {
					log.Error().Msgf("encountered error during argocd application cleanup: %s", err)
				}

				log.Info().Msg("opening argocd port forward")
				//* ArgoCD port-forward
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
					return err
				}
				argocdPassword := secData["password"]

				argocdAuthToken, err := argocd.GetArgoCDToken("admin", argocdPassword)
				if err != nil {
					return err
				}

				log.Info().Msgf("port-forward to argocd is available at %s", providerConfigs.ArgocdPortForwardURL)

				customTransport := http.DefaultTransport.(*http.Transport).Clone()
				customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
				argocdHttpClient := http.Client{Transport: customTransport}
				log.Info().Msg("deleting the registry application")
				httpCode, _, err := argocd.DeleteApplication(&argocdHttpClient, config.RegistryAppName, argocdAuthToken, "true")
				if err != nil {
					errors.HandleClusterError(cl, err.Error())
					return err
				}
				log.Info().Msgf("http status code %d", httpCode)
			}

			// Pause before cluster destroy to prevent a race condition
			log.Info().Msg("waiting for google Kubernetes cluster resource removal to finish...")
			time.Sleep(time.Second * 10)

			cl.ArgoCDDeleteRegistryCheck = true
			err = secrets.UpdateCluster(kcfg.Clientset, *cl)
			if err != nil {
				return err
			}
		}

		log.Info().Msg("destroying google cloud resources")
		tfEntrypoint := config.GitopsDir + fmt.Sprintf("/terraform/%s", cl.CloudProvider)
		tfEnvs := map[string]string{}
		tfEnvs = googleext.GetGoogleTerraformEnvs(tfEnvs, cl)
		tfEnvs["TF_VAR_project"] = cl.GoogleAuth.ProjectId

		switch cl.GitProvider {
		case "github":
			tfEnvs = googleext.GetGithubTerraformEnvs(tfEnvs, cl)
		case "gitlab":
			gid, err := strconv.Atoi(fmt.Sprint(cl.GitlabOwnerGroupID))
			if err != nil {
				return fmt.Errorf("couldn't convert gitlab group id to int: %s", err)
			}
			tfEnvs = googleext.GetGitlabTerraformEnvs(tfEnvs, gid, cl)
		}
		err = terraformext.InitDestroyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs)
		if err != nil {
			log.Error().Msgf("error executing terraform destroy %s", tfEntrypoint)
			errors.HandleClusterError(cl, err.Error())
			return err
		}
		log.Info().Msg("google resources terraform destroyed")

		cl.CloudTerraformApplyCheck = false
		cl.CloudTerraformApplyFailedCheck = false
		err = secrets.UpdateCluster(kcfg.Clientset, *cl)

		if err != nil {
			return err
		}
	}

	// remove ssh key provided one was created
	if cl.GitProvider == "gitlab" {
		gitlabClient, err := gitlab.NewGitLabClient(cl.GitAuth.Token, cl.GitAuth.Owner)
		if err != nil {
			return err
		}
		log.Info().Msgf("attempting to delete managed ssh key...")
		err = gitlabClient.DeleteUserSSHKey("kbot-ssh-key")
		if err != nil {
			log.Warn().Msg(err.Error())
		}
	}

	telemetry.SendEvent(telemetryEvent, telemetry.ClusterDeleteCompleted, "")

	cl.Status = constants.ClusterStatusDeleted
	err = secrets.UpdateCluster(kcfg.Clientset, *cl)
	if err != nil {
		return err
	}

	err = pkg.ResetK1Dir(config.K1Dir)
	if err != nil {
		return err
	}

	return nil
}
