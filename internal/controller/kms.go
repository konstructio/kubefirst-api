/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	"fmt"

	"github.com/go-git/go-git/v5"
	"github.com/kubefirst/runtime/pkg"
	"github.com/kubefirst/runtime/pkg/gitClient"
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

			err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "aws_kms_key_id", awsKmsKeyId)
			if err != nil {
				return err
			}

			if err := pkg.ReplaceFileContent(
				fmt.Sprintf("%s/registry/%s/components/vault/application.yaml", clctrl.ProviderConfig.GitopsDir, clctrl.ClusterName),
				"<AWS_KMS_KEY_ID>",
				awsKmsKeyId,
			); err != nil {
				return err
			}

			err = gitClient.Commit(gitopsRepo, "committing detokenized kms key")
			if err != nil {
				return err
			}

			//Leaving this here until I know where else it is used and caching the keys for now

			err = gitopsRepo.Push(&git.PushOptions{
				RemoteName: clctrl.GitProvider,
				Auth:       HttpAuth,
			})
			if err != nil {
				return err
			}

			err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "aws_kms_key_detokenized_check", true)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
