/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package civo

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/civo/civogo"
	civoext "github.com/kubefirst/kubefirst-api/extensions/civo"
	"github.com/kubefirst/kubefirst-api/internal/db"
	"github.com/kubefirst/kubefirst-api/internal/types"
	"github.com/kubefirst/runtime/pkg/argocd"
	"github.com/kubefirst/runtime/pkg/civo"
	gitlab "github.com/kubefirst/runtime/pkg/gitlab"
	"github.com/kubefirst/runtime/pkg/k8s"
	"github.com/kubefirst/runtime/pkg/terraform"
	log "github.com/sirupsen/logrus"
)

// DeleteCivoCluster
func DeleteCivoCluster(cl *types.Cluster) error {
	// Instantiate civo config
	config := civo.GetConfig(cl.ClusterName, cl.DomainName, cl.GitProvider, cl.GitOwner)
	mdbcl := &db.MongoDBClient{}
	err := mdbcl.InitDatabase()
	if err != nil {
		return err
	}

	switch cl.GitProvider {
	case "github":
		if cl.GitTerraformApplyCheck {
			log.Info("destroying github resources with terraform")

			tfEntrypoint := config.GitopsDir + "/terraform/github"
			tfEnvs := map[string]string{}
			tfEnvs = civoext.GetCivoTerraformEnvs(tfEnvs, cl)
			tfEnvs = civoext.GetGithubTerraformEnvs(tfEnvs, cl)
			err := terraform.InitDestroyAutoApprove(false, tfEntrypoint, tfEnvs)
			if err != nil {
				log.Printf("error executing terraform destroy %s", tfEntrypoint)
				return err
			}
			log.Info("github resources terraform destroyed")

			err = mdbcl.UpdateCluster(cl.ClusterName, "git_terraform_apply_check", false)
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
			tfEnvs = civoext.GetCivoTerraformEnvs(tfEnvs, cl)
			tfEnvs = civoext.GetGitlabTerraformEnvs(tfEnvs, gitlabClient.ParentGroupID, cl)
			err = terraform.InitDestroyAutoApprove(false, tfEntrypoint, tfEnvs)
			if err != nil {
				log.Infof("error executing terraform destroy %s", tfEntrypoint)
				return err
			}

			log.Info("gitlab resources terraform destroyed")

			err = mdbcl.UpdateCluster(cl.ClusterName, "git_terraform_apply_check", false)
			if err != nil {
				return err
			}
		}
	}

	if cl.CloudTerraformApplyCheck || cl.CloudTerraformApplyFailedCheck {
		if !cl.CloudTerraformApplyFailedCheck {
			kcfg := k8s.CreateKubeConfig(false, config.Kubeconfig)

			log.Info("destroying civo resources with terraform")

			client, err := civogo.NewClient(os.Getenv("CIVO_TOKEN"), cl.CloudRegion)
			if err != nil {
				return fmt.Errorf(err.Error())
			}

			cluster, err := client.FindKubernetesCluster(cl.ClusterName)
			if err != nil {
				return err
			}
			log.Info("cluster name: " + cluster.ID)

			clusterVolumes, err := client.ListVolumesForCluster(cluster.ID)
			if err != nil {
				return err
			}

			// Only port-forward to ArgoCD and delete registry if ArgoCD was installed
			if cl.ArgoCDInstallCheck {
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

				log.Infof("port-forward to argocd is available at %s", civo.ArgocdPortForwardURL)

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

			for _, vol := range clusterVolumes {
				log.Info("removing volume with name: " + vol.Name)
				_, err := client.DeleteVolume(vol.ID)
				if err != nil {
					return err
				}
				log.Info("volume " + vol.ID + " deleted")
			}

			// Pause before cluster destroy to prevent a race condition
			log.Info("waiting for Civo Kubernetes cluster resource removal to finish...")
			time.Sleep(time.Second * 10)
		}

		log.Info("destroying civo cloud resources")
		tfEntrypoint := config.GitopsDir + "/terraform/civo"
		tfEnvs := map[string]string{}
		tfEnvs = civoext.GetCivoTerraformEnvs(tfEnvs, cl)

		switch cl.GitProvider {
		case "github":
			tfEnvs = civoext.GetGithubTerraformEnvs(tfEnvs, cl)
		case "gitlab":
			gid, err := strconv.Atoi(fmt.Sprint(cl.GitlabOwnerGroupID))
			if err != nil {
				return fmt.Errorf("couldn't convert gitlab group id to int: %s", err)
			}
			tfEnvs = civoext.GetGitlabTerraformEnvs(tfEnvs, gid, cl)
		}
		err = terraform.InitDestroyAutoApprove(false, tfEntrypoint, tfEnvs)
		if err != nil {
			log.Printf("error executing terraform destroy %s", tfEntrypoint)
			return err
		}
		log.Info("civo resources terraform destroyed")

		err = mdbcl.UpdateCluster(cl.ClusterName, "cloud_terraform_apply_check", false)
		if err != nil {
			return err
		}

		err = mdbcl.UpdateCluster(cl.ClusterName, "cloud_terraform_apply_failed_check", false)
		if err != nil {
			return err
		}
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

	return nil
}
