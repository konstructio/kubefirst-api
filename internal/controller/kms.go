/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	"fmt"

	"github.com/go-git/go-git/v5"
	githttps "github.com/go-git/go-git/v5/plumbing/transport/http"
	pkg "github.com/kubefirst/kubefirst-api/internal"
	"github.com/kubefirst/kubefirst-api/internal/gitClient"
	"github.com/kubefirst/kubefirst-api/internal/secrets"
)

// DetokenizeKMSKeyID
func (clctrl *ClusterController) DetokenizeKMSKeyID() error {
	cl, err := clctrl.GetCurrentClusterRecord()
	if err != nil {
		return err
	}

	if !cl.AWSKMSKeyDetokenizedCheck {
		switch clctrl.CloudProvider {
		case "aws":
			// KMS
			gitopsRepo, err := git.PlainOpen(clctrl.ProviderConfig.GitopsDir)
			if err != nil {
				return err
			}
			awsKmsKeyId, err := clctrl.AwsClient.GetKmsKeyID(fmt.Sprintf("alias/vault_%s", clctrl.ClusterName))
			if err != nil {
				return err
			}

			clctrl.Cluster.AWSKMSKeyId = awsKmsKeyId
			err = secrets.UpdateCluster(clctrl.KubernetesClient, clctrl.Cluster)

			if err != nil {
				return err
			}

			var registryPath string
			if clctrl.CloudProvider == "civo" && clctrl.GitProvider == "github" {
				registryPath = fmt.Sprintf("registry/clusters/%s", clctrl.ClusterName)
			} else if clctrl.CloudProvider == "civo" && clctrl.GitProvider == "gitlab" {
				registryPath = fmt.Sprintf("registry/clusters/%s", clctrl.ClusterName)
			} else if clctrl.CloudProvider == "aws" && clctrl.GitProvider == "github" {
				registryPath = fmt.Sprintf("registry/clusters/%s", clctrl.ClusterName)
			} else if clctrl.CloudProvider == "aws" && clctrl.GitProvider == "gitlab" {
				registryPath = fmt.Sprintf("registry/clusters/%s", clctrl.ClusterName)
			} else if clctrl.CloudProvider == "google" && clctrl.GitProvider == "github" {
				registryPath = fmt.Sprintf("registry/clusters/%s", clctrl.ClusterName)
			} else if clctrl.CloudProvider == "google" && clctrl.GitProvider == "gitlab" {
				registryPath = fmt.Sprintf("registry/clusters/%s", clctrl.ClusterName)
			} else if clctrl.CloudProvider == "digitalocean" && clctrl.GitProvider == "github" {
				registryPath = fmt.Sprintf("registry/clusters/%s", clctrl.ClusterName)
			} else if clctrl.CloudProvider == "digitalocean" && clctrl.GitProvider == "gitlab" {
				registryPath = fmt.Sprintf("registry/clusters/%s", clctrl.ClusterName)
			} else if clctrl.CloudProvider == "vultr" && clctrl.GitProvider == "github" {
				registryPath = fmt.Sprintf("registry/clusters/%s", clctrl.ClusterName)
			} else if clctrl.CloudProvider == "vultr" && clctrl.GitProvider == "gitlab" {
				registryPath = fmt.Sprintf("registry/clusters/%s", clctrl.ClusterName)
			} else {
				registryPath = fmt.Sprintf("registry/%s", clctrl.ClusterName)
			}

			if err := pkg.ReplaceFileContent(
				fmt.Sprintf("%s/%s/components/vault/application.yaml", clctrl.ProviderConfig.GitopsDir, registryPath),
				"<AWS_KMS_KEY_ID>",
				awsKmsKeyId,
			); err != nil {
				return err
			}

			err = gitClient.Commit(gitopsRepo, "committing detokenized kms key")
			if err != nil {
				return err
			}

			err = gitopsRepo.Push(&git.PushOptions{
				RemoteName: clctrl.GitProvider,
				Auth: &githttps.BasicAuth{
					Username: clctrl.GitAuth.User,
					Password: clctrl.GitAuth.Token,
				},
			})
			if err != nil {
				return err
			}

			clctrl.Cluster.AWSKMSKeyDetokenizedCheck = true
			err = secrets.UpdateCluster(clctrl.KubernetesClient, clctrl.Cluster)

			if err != nil {
				return err
			}
		}
	}

	return nil
}
