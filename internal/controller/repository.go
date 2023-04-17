/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	"fmt"
	"strings"

	"github.com/go-git/go-git/v5"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/kubefirst/runtime/pkg/gitlab"
	"github.com/kubefirst/runtime/pkg/k3d"
	log "github.com/sirupsen/logrus"
)

// RepositoryPrep
func (clctrl *ClusterController) RepositoryPrep() error {
	cl, err := clctrl.MdbCl.GetCluster(clctrl.ClusterName)
	if err != nil {
		return err
	}

	if !cl.GitopsReadyCheck {
		err := k3d.PrepareGitRepositories(
			clctrl.GitProvider,
			clctrl.ClusterName,
			clctrl.ClusterType,
			clctrl.ProviderConfig.DestinationGitopsRepoGitURL,
			clctrl.ProviderConfig.GitopsDir,
			clctrl.GitopsTemplateBranchFlag,
			clctrl.GitopsTemplateURLFlag,
			clctrl.ProviderConfig.DestinationMetaphorRepoGitURL,
			clctrl.ProviderConfig.K1Dir,
			clctrl.CreateTokens("gitops").(*k3d.GitopsTokenValues),
			clctrl.ProviderConfig.MetaphorDir,
			clctrl.CreateTokens("metaphor").(*k3d.MetaphorTokenValues),
		)
		if err != nil {
			return err
		}

		err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "gitops_ready_check", true)
		if err != nil {
			return err
		}
	}

	return nil
}

// RepositoryPush
func (clctrl *ClusterController) RepositoryPush() error {
	cl, err := clctrl.MdbCl.GetCluster(clctrl.ClusterName)
	if err != nil {
		return err
	}

	if !cl.GitopsPushedCheck {
		publicKeys, err := gitssh.NewPublicKeys("git", []byte(cl.PrivateKey), "")
		if err != nil {
			log.Infof("generate public keys failed: %s\n", err.Error())
		}

		// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricGitopsRepoPushStarted, "")
		gitopsRepo, err := git.PlainOpen(clctrl.ProviderConfig.GitopsDir)
		if err != nil {
			log.Infof("error opening repo at: %s", clctrl.ProviderConfig.GitopsDir)
		}

		metaphorRepo, err := git.PlainOpen(clctrl.ProviderConfig.MetaphorDir)
		if err != nil {
			log.Infof("error opening repo at: %s", clctrl.ProviderConfig.MetaphorDir)
		}

		// For GitLab, we currently need to add an ssh key to the authenticating user
		if clctrl.GitProvider == "gitlab" {
			gitlabClient, err := gitlab.NewGitLabClient(clctrl.GitToken, clctrl.GitOwner)
			if err != nil {
				return err
			}
			keys, err := gitlabClient.GetUserSSHKeys()
			if err != nil {
				log.Fatalf("unable to check for ssh keys in gitlab: %s", err.Error())
			}

			var keyName = "kubefirst-k3d-ssh-key"
			var keyFound bool = false
			for _, key := range keys {
				if key.Title == keyName {
					if strings.Contains(key.Key, strings.TrimSuffix(clctrl.PublicKey, "\n")) {
						log.Infof("ssh key %s already exists and key is up to date, continuing", keyName)
						keyFound = true
					} else {
						log.Fatalf("ssh key %s already exists and key data has drifted - please remove before continuing", keyName)
					}
				}
			}
			if !keyFound {
				log.Infof("creating ssh key %s...", keyName)
				err := gitlabClient.AddUserSSHKey(keyName, clctrl.PublicKey)
				if err != nil {
					log.Fatalf("error adding ssh key %s: %s", keyName, err.Error())
				}
			}
		}

		// Push gitops repo to remote
		err = gitopsRepo.Push(
			&git.PushOptions{
				RemoteName: clctrl.GitProvider,
				Auth:       publicKeys,
			},
		)
		if err != nil {
			msg := fmt.Sprintf("error pushing detokenized gitops repository to remote %s: %s", clctrl.ProviderConfig.DestinationGitopsRepoGitURL, err)
			// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricGitopsRepoPushFailed, msg)
			log.Fatal(msg)
		}

		// push metaphor repo to remote
		err = metaphorRepo.Push(
			&git.PushOptions{
				RemoteName: "origin",
				Auth:       publicKeys,
			},
		)
		if err != nil {
			msg := fmt.Sprintf("error pushing detokenized metaphor repository to remote %s: %s", clctrl.ProviderConfig.DestinationMetaphorRepoGitURL, err)
			// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricGitopsRepoPushFailed, msg)
			log.Fatal(msg)
		}

		log.Infof("successfully pushed gitops and metaphor repositories to git@%s/%s", clctrl.GitHost, clctrl.GitOwner)
		// todo delete the local gitops repo and re-clone it
		// todo that way we can stop worrying about which origin we're going to push to
		// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricGitopsRepoPushCompleted, "")

		err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "gitops_pushed_check", true)
		if err != nil {
			return err
		}
	}

	return nil
}
