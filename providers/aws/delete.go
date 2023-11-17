/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package aws

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	awsext "github.com/kubefirst/kubefirst-api/extensions/aws"
	terraformext "github.com/kubefirst/kubefirst-api/extensions/terraform"
	"github.com/kubefirst/kubefirst-api/internal/constants"
	"github.com/kubefirst/kubefirst-api/internal/db"
	"github.com/kubefirst/kubefirst-api/internal/errors"
	"github.com/kubefirst/kubefirst-api/pkg/providerConfigs"
	pkgtypes "github.com/kubefirst/kubefirst-api/pkg/types"
	"github.com/kubefirst/metrics-client/pkg/telemetry"
	"github.com/kubefirst/runtime/pkg"
	"github.com/kubefirst/runtime/pkg/argocd"
	awsinternal "github.com/kubefirst/runtime/pkg/aws"
	gitlab "github.com/kubefirst/runtime/pkg/gitlab"
	"github.com/kubefirst/runtime/pkg/k8s"
	log "github.com/sirupsen/logrus"
)

// DeleteAWSCluster
func DeleteAWSCluster(cl *pkgtypes.Cluster, telemetryEvent telemetry.TelemetryEvent) error {

	// Logging handler
	// Logs to stdout to maintain compatibility with event streaming
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "",
	})
	log.SetReportCaller(false)
	log.SetOutput(os.Stdout)

	telemetry.SendEvent(telemetryEvent, telemetry.ClusterDeleteStarted, "")

	// Instantiate aws config
	config := providerConfigs.GetConfig(cl.ClusterName, cl.DomainName, cl.GitProvider, cl.GitAuth.Owner, cl.GitProtocol, cl.CloudflareAuth.APIToken, cl.CloudflareAuth.OriginCaIssuerKey)

	err := db.Client.UpdateCluster(cl.ClusterName, "status", constants.ClusterStatusDeleting)
	if err != nil {
		return err
	}

	switch cl.GitProvider {
	case "github":
		if cl.GitTerraformApplyCheck {
			log.Info("destroying github resources with terraform")

			tfEntrypoint := config.GitopsDir + "/terraform/github"
			tfEnvs := map[string]string{}
			tfEnvs = awsext.GetAwsTerraformEnvs(tfEnvs, cl)
			tfEnvs = awsext.GetGithubTerraformEnvs(tfEnvs, cl)
			err := terraformext.InitDestroyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs)
			if err != nil {
				log.Errorf("error executing terraform destroy %s", tfEntrypoint)
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
			tfEnvs = awsext.GetAwsTerraformEnvs(tfEnvs, cl)
			tfEnvs = awsext.GetGitlabTerraformEnvs(tfEnvs, gitlabClient.ParentGroupID, cl)
			err = terraformext.InitDestroyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs)
			if err != nil {
				log.Errorf("error executing terraform destroy %s", tfEntrypoint)
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

	if cl.CloudTerraformApplyCheck || cl.CloudTerraformApplyFailedCheck {
		if !cl.ArgoCDDeleteRegistryCheck {
			awsClient := &awsinternal.AWSConfiguration{
				Config: awsinternal.NewAwsV3(
					cl.CloudRegion,
					cl.AWSAuth.AccessKeyID,
					cl.AWSAuth.SecretAccessKey,
					cl.AWSAuth.SessionToken,
				),
			}
			kcfg := awsext.CreateEKSKubeconfig(&awsClient.Config, cl.ClusterName)

			log.Info("destroying aws resources with terraform")

			// Only port-forward to ArgoCD and delete registry if ArgoCD was installed
			if cl.ArgoCDInstallCheck {
				removeArgoCDApps := []string{"ingress-nginx-components", "ingress-nginx"}
				err = argocd.ArgoCDApplicationCleanup(kcfg.Clientset, removeArgoCDApps)
				if err != nil {
					log.Errorf("encountered error during argocd application cleanup: %s", err)
				}

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
					80,
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
					errors.HandleClusterError(cl, err.Error())
					return err
				}
				log.Infof("http status code %d", httpCode)
			}

			// Pause before cluster destroy to prevent a race condition
			log.Info("waiting for aws Kubernetes cluster resource removal to finish...")
			time.Sleep(time.Second * 10)

			err = db.Client.UpdateCluster(cl.ClusterName, "argocd_delete_registry_check", true)
			if err != nil {
				return err
			}
		}

		log.Info("destroying aws cloud resources")
		tfEntrypoint := config.GitopsDir + fmt.Sprintf("/terraform/%s", cl.CloudProvider)
		tfEnvs := map[string]string{}
		tfEnvs = awsext.GetAwsTerraformEnvs(tfEnvs, cl)
		tfEnvs["TF_VAR_aws_account_id"] = cl.AWSAccountId

		switch cl.GitProvider {
		case "github":
			tfEnvs = awsext.GetGithubTerraformEnvs(tfEnvs, cl)
		case "gitlab":
			gid, err := strconv.Atoi(fmt.Sprint(cl.GitlabOwnerGroupID))
			if err != nil {
				return fmt.Errorf("couldn't convert gitlab group id to int: %s", err)
			}
			tfEnvs = awsext.GetGitlabTerraformEnvs(tfEnvs, gid, cl)
		}
		err = terraformext.InitDestroyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs)
		if err != nil {
			log.Errorf("error executing terraform destroy %s", tfEntrypoint)
			errors.HandleClusterError(cl, err.Error())
			return err
		}
		log.Info("aws resources terraform destroyed")

		err = db.Client.UpdateCluster(cl.ClusterName, "cloud_terraform_apply_check", false)
		if err != nil {
			return err
		}

		err = db.Client.UpdateCluster(cl.ClusterName, "cloud_terraform_apply_failed_check", false)
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
		log.Info("attempting to delete managed ssh key...")
		err = gitlabClient.DeleteUserSSHKey("kbot-ssh-key")
		if err != nil {
			log.Warn(err.Error())
		}
	}

	telemetry.SendEvent(telemetryEvent, telemetry.ClusterDeleteCompleted, "")

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
