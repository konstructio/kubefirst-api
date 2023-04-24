/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k3d

import (
	"os"
	"strconv"

	"github.com/kubefirst/kubefirst-api/internal/db"
	"github.com/kubefirst/kubefirst-api/internal/types"
	"github.com/kubefirst/runtime/pkg"
	"github.com/kubefirst/runtime/pkg/gitlab"
	"github.com/kubefirst/runtime/pkg/k3d"
	"github.com/kubefirst/runtime/pkg/terraform"
	log "github.com/sirupsen/logrus"
)

// DeleteK3DCluster
func DeleteK3DCluster(cl *types.Cluster) error {
	// Logging handler
	// Logs to stdout to maintain compatibility with event streaming
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "",
	})
	log.SetReportCaller(false)
	log.SetOutput(os.Stdout)

	// Instantiate K3d config
	config := k3d.GetConfig(cl.ClusterName, cl.GitProvider, cl.GitOwner)
	mdbcl := &db.MongoDBClient{}
	err := mdbcl.InitDatabase("api", "clusters")
	if err != nil {
		return err
	}

	switch cl.GitProvider {
	case "github":
		if cl.GitTerraformApplyCheck {
			log.Info("destroying github resources with terraform")

			tfEntrypoint := config.GitopsDir + "/terraform/github"
			tfEnvs := map[string]string{}

			tfEnvs["GITHUB_TOKEN"] = cl.GitToken
			tfEnvs["GITHUB_OWNER"] = cl.GitOwner
			tfEnvs["TF_VAR_atlantis_repo_webhook_secret"] = cl.AtlantisWebhookSecret
			tfEnvs["TF_VAR_atlantis_repo_webhook_url"] = cl.AtlantisWebhookURL
			tfEnvs["TF_VAR_kbot_ssh_public_key"] = cl.PublicKey
			tfEnvs["AWS_ACCESS_KEY_ID"] = pkg.MinioDefaultUsername
			tfEnvs["AWS_SECRET_ACCESS_KEY"] = pkg.MinioDefaultPassword
			tfEnvs["TF_VAR_aws_access_key_id"] = pkg.MinioDefaultUsername
			tfEnvs["TF_VAR_aws_secret_access_key"] = pkg.MinioDefaultPassword

			err := terraform.InitDestroyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs)
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

			tfEnvs["GITLAB_TOKEN"] = cl.GitToken
			tfEnvs["GITLAB_OWNER"] = cl.GitOwner
			tfEnvs["TF_VAR_atlantis_repo_webhook_secret"] = cl.AtlantisWebhookSecret
			tfEnvs["TF_VAR_atlantis_repo_webhook_url"] = cl.AtlantisWebhookURL
			tfEnvs["TF_VAR_owner_group_id"] = strconv.Itoa(gitlabClient.ParentGroupID)

			err = terraform.InitDestroyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs)
			if err != nil {
				log.Printf("error executing terraform destroy %s", tfEntrypoint)
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
		err = k3d.DeleteK3dCluster(cl.ClusterName, config.K1Dir, config.K3dClient)
		if err != nil {
			return err
		}

		log.Info("k3d resources destroyed")
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
