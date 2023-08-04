/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package vultr

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	terraformext "github.com/kubefirst/kubefirst-api/extensions/terraform"
	vultrext "github.com/kubefirst/kubefirst-api/extensions/vultr"
	"github.com/kubefirst/kubefirst-api/internal/constants"
	"github.com/kubefirst/kubefirst-api/internal/db"
	"github.com/kubefirst/kubefirst-api/internal/errors"
	"github.com/kubefirst/kubefirst-api/internal/telemetryShim"
	"github.com/kubefirst/kubefirst-api/internal/types"
	"github.com/kubefirst/runtime/pkg"
	"github.com/kubefirst/runtime/pkg/argocd"
	gitlab "github.com/kubefirst/runtime/pkg/gitlab"
	"github.com/kubefirst/runtime/pkg/k8s"
	"github.com/kubefirst/runtime/pkg/providerConfigs"
	"github.com/kubefirst/runtime/pkg/segment"
	"github.com/kubefirst/runtime/pkg/vultr"
	log "github.com/sirupsen/logrus"
)

// DeleteVultrCluster
func DeleteVultrCluster(cl *types.Cluster) error {
	// Logging handler
	// Logs to stdout to maintain compatibility with event streaming
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "",
	})
	log.SetReportCaller(false)
	log.SetOutput(os.Stdout)

	// Telemetry handler
	segmentClient, err := telemetryShim.SetupTelemetry(*cl)
	if err != nil {
		return err
	}
	defer segmentClient.Client.Close()

	telemetryShim.Transmit(cl.UseTelemetry, segmentClient, segment.MetricClusterDeleteStarted, "")

	// Instantiate vultr config
	config := providerConfigs.GetConfig(cl.ClusterName, cl.DomainName, cl.GitProvider, cl.GitOwner, cl.GitProtocol, cl.CloudflareAuth.Token)

	err = db.Client.UpdateCluster(cl.ClusterName, "status", constants.ClusterStatusDeleting)
	if err != nil {
		return err
	}

	switch cl.GitProvider {
	case "github":
		if cl.GitTerraformApplyCheck {
			log.Info("destroying github resources with terraform")

			tfEntrypoint := config.GitopsDir + "/terraform/github"
			tfEnvs := map[string]string{}
			tfEnvs = vultrext.GetVultrTerraformEnvs(tfEnvs, cl)
			tfEnvs = vultrext.GetGithubTerraformEnvs(tfEnvs, cl)
			err := terraformext.InitDestroyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs)
			if err != nil {
				log.Printf("error executing terraform destroy %s", tfEntrypoint)
				errors.HandleClusterError(cl, err.Error())
				return err
			}
			log.Info("github resources terraform destroyed")

			err = db.Client.UpdateCluster(cl.ClusterName, "git_terraform_apply_check", false)
			if err != nil {
				return err
			}
		}
	case "gitlab":
		if cl.GitTerraformApplyCheck {
			log.Info("destroying gitlab resources with terraform")
			gitlabClient, err := gitlab.NewGitLabClient(cl.GitToken, cl.GitOwner)
			if err != nil {
				return err
			}

			// Before removing Terraform resources, remove any container registry repositories
			// since failing to remove them beforehand will result in an apply failure
			var projectsForDeletion = []string{"gitops", "metaphor"}
			for _, project := range projectsForDeletion {
				projectExists, err := gitlabClient.CheckProjectExists(project)
				if err != nil {
					log.Errorf("could not check for existence of project %s: %s", project, err)
				}
				if projectExists {
					log.Infof("checking project %s for container registries...", project)
					crr, err := gitlabClient.GetProjectContainerRegistryRepositories(project)
					if err != nil {
						log.Errorf("could not retrieve container registry repositories: %s", err)
					}
					if len(crr) > 0 {
						for _, cr := range crr {
							err := gitlabClient.DeleteContainerRegistryRepository(project, cr.ID)
							if err != nil {
								log.Errorf("error deleting container registry repository: %s", err)
							}
						}
					} else {
						log.Infof("project %s does not have any container registries, skipping", project)
					}
				} else {
					log.Infof("project %s does not exist, skipping", project)
				}
			}

			tfEntrypoint := config.GitopsDir + "/terraform/gitlab"
			tfEnvs := map[string]string{}
			tfEnvs = vultrext.GetVultrTerraformEnvs(tfEnvs, cl)
			tfEnvs = vultrext.GetGitlabTerraformEnvs(tfEnvs, gitlabClient.ParentGroupID, cl)
			err = terraformext.InitDestroyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs)
			if err != nil {
				log.Infof("error executing terraform destroy %s", tfEntrypoint)
				errors.HandleClusterError(cl, err.Error())
				return err
			}

			log.Info("gitlab resources terraform destroyed")

			err = db.Client.UpdateCluster(cl.ClusterName, "git_terraform_apply_check", false)
			if err != nil {
				return err
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
		err = argocd.ArgoCDApplicationCleanup(kcfg.Clientset, removeArgoCDApps)
		if err != nil {
			log.Errorf("encountered error during argocd application cleanup: %s", err)
		}
		// Pause before cluster destroy to prevent a race condition
		log.Info("waiting for argocd application deletion to complete...")
		time.Sleep(time.Second * 20)
	}

	//GetKubernetesAssociatedBlockStorage
	vultrConf := vultr.VultrConfiguration{
		Client:  vultr.NewVultr(cl.VultrAuth.Token),
		Context: context.Background(),
	}
	blockStorage, err := vultrConf.GetKubernetesAssociatedBlockStorage("", true)
	if err != nil {
		return err
	}

	if cl.CloudTerraformApplyCheck || cl.CloudTerraformApplyFailedCheck {
		if !cl.CloudTerraformApplyFailedCheck {
			kcfg := k8s.CreateKubeConfig(false, config.Kubeconfig)

			log.Info("destroying vultr resources with terraform")

			// Only port-forward to ArgoCD and delete registry if ArgoCD was installed
			if !cl.ArgoCDDeleteRegistryCheck {
				log.Info("opening argocd port forward")
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
					8080,
					8080,
					argoCDStopChannel,
				)

				log.Info("getting new auth token for argocd")

				secData, err := k8s.ReadSecretV2(kcfg.Clientset, "argocd", "argocd-initial-admin-secret")
				if err != nil {
					return err
				}
				argocdPassword := secData["password"]

				argocdAuthToken, err := argocd.GetArgoCDToken("admin", argocdPassword)
				if err != nil {
					return err
				}

				log.Infof("port-forward to argocd is available at %s", providerConfigs.ArgocdPortForwardURL)

				customTransport := http.DefaultTransport.(*http.Transport).Clone()
				customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
				argocdHttpClient := http.Client{Transport: customTransport}
				log.Info("deleting the registry application")
				httpCode, _, err := argocd.DeleteApplication(&argocdHttpClient, config.RegistryAppName, argocdAuthToken, "true")
				if err != nil {
					return err
				}
				log.Infof("http status code %d", httpCode)
			}

			// Pause before cluster destroy to prevent a race condition
			log.Info("waiting for vultr kubernetes cluster resource removal to finish...")
			time.Sleep(time.Second * 10)

			err = db.Client.UpdateCluster(cl.ClusterName, "argocd_delete_registry_check", true)
			if err != nil {
				return err
			}
		}

		log.Info("destroying vultr cloud resources")
		tfEntrypoint := config.GitopsDir + "/terraform/vultr"
		tfEnvs := map[string]string{}
		tfEnvs = vultrext.GetVultrTerraformEnvs(tfEnvs, cl)

		switch cl.GitProvider {
		case "github":
			tfEnvs = vultrext.GetGithubTerraformEnvs(tfEnvs, cl)
		case "gitlab":
			gid, err := strconv.Atoi(fmt.Sprint(cl.GitlabOwnerGroupID))
			if err != nil {
				return fmt.Errorf("couldn't convert gitlab group id to int: %s", err)
			}
			tfEnvs = vultrext.GetGitlabTerraformEnvs(tfEnvs, gid, cl)
		}
		err = terraformext.InitDestroyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs)
		if err != nil {
			log.Printf("error executing terraform destroy %s", tfEntrypoint)
			errors.HandleClusterError(cl, err.Error())
			return err
		}
		log.Info("vultr resources terraform destroyed")

		err = db.Client.UpdateCluster(cl.ClusterName, "cloud_terraform_apply_check", false)
		if err != nil {
			return err
		}

		err = db.Client.UpdateCluster(cl.ClusterName, "cloud_terraform_apply_failed_check", false)
		if err != nil {
			return err
		}
	}

	// Remove hanging volumes
	// This fails with regularity if done too quickly
	time.Sleep(time.Second * 45)
	err = vultrConf.DeleteBlockStorage(blockStorage)
	if err != nil {
		return err
	}

	// remove ssh key provided one was created
	if cl.GitProvider == "gitlab" {
		gitlabClient, err := gitlab.NewGitLabClient(cl.GitToken, cl.GitOwner)
		if err != nil {
			return err
		}
		log.Info("attempting to delete managed ssh key...")
		err = gitlabClient.DeleteUserSSHKey("kbot-ssh-key")
		if err != nil {
			log.Warn(err.Error())
		}
	}

	telemetryShim.Transmit(cl.UseTelemetry, segmentClient, segment.MetricClusterDeleteCompleted, "")

	err = db.Client.UpdateCluster(cl.ClusterName, "status", constants.ClusterStatusDeleted)
	if err != nil {
		return err
	}

	err = pkg.ResetK1Dir(config.K1Dir)
	if err != nil {
		return err
	}

	return nil
}
