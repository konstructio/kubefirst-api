/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-git/go-git/v5"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/kubefirst/kubefirst-api/internal/telemetryShim"
	"github.com/kubefirst/runtime/pkg/civo"
	"github.com/kubefirst/runtime/pkg/digitalocean"
	"github.com/kubefirst/runtime/pkg/gitlab"
	"github.com/kubefirst/runtime/pkg/providerConfigs"
	"github.com/kubefirst/runtime/pkg/segment"
	"github.com/kubefirst/runtime/pkg/vultr"
	log "github.com/sirupsen/logrus"
)

// RepositoryPrep
func (clctrl *ClusterController) RepositoryPrep() error {
	// Logging handler
	// Logs to stdout to maintain compatibility with event streaming
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "",
	})
	log.SetReportCaller(false)
	log.SetOutput(os.Stdout)

	cl, err := clctrl.MdbCl.GetCluster(clctrl.ClusterName)
	if err != nil {
		return err
	}

	if !cl.GitopsReadyCheck {
		log.Info("initializing the gitops repository - this may take several minutes")

		switch clctrl.CloudProvider {
		case "aws":
			err := providerConfigs.PrepareGitRepositories(
				clctrl.CloudProvider,
				clctrl.GitProvider,
				clctrl.ClusterName,
				clctrl.ClusterType,
				AWSDestinationGitopsRepoURL,
				clctrl.ProviderConfig.GitopsDir,
				clctrl.GitopsTemplateBranch,
				clctrl.GitopsTemplateURL,
				AWSDestinationMetaphorRepoURL,
				clctrl.ProviderConfig.K1Dir,
				clctrl.CreateTokens("gitops").(*providerConfigs.GitopsDirectoryValues),
				clctrl.ProviderConfig.MetaphorDir,
				clctrl.CreateTokens("metaphor").(*providerConfigs.MetaphorTokenValues),
				true,
				cl.GitProtocol,
			)
			if err != nil {
				return err
			}
		case "civo":
			err := providerConfigs.PrepareGitRepositories(
				clctrl.CloudProvider,
				clctrl.GitProvider,
				clctrl.ClusterName,
				clctrl.ClusterType,
				CivoDestinationGitopsRepoURL,
				clctrl.ProviderConfig.GitopsDir,
				clctrl.GitopsTemplateBranch,
				clctrl.GitopsTemplateURL,
				CivoDestinationMetaphorRepoURL,
				clctrl.ProviderConfig.K1Dir,
				clctrl.CreateTokens("gitops").(*providerConfigs.GitopsDirectoryValues),
				clctrl.ProviderConfig.MetaphorDir,
				clctrl.CreateTokens("metaphor").(*providerConfigs.MetaphorTokenValues),
				civo.GetDomainApexContent(clctrl.DomainName),
				cl.GitProtocol,
			)
			if err != nil {
				return err
			}
		case "digitalocean":
			err = providerConfigs.PrepareGitRepositories(
				clctrl.CloudProvider,
				clctrl.GitProvider,
				clctrl.ClusterName,
				clctrl.ClusterType,
				DigitaloceanDestinationGitopsRepoURL,
				clctrl.ProviderConfig.GitopsDir,
				clctrl.GitopsTemplateBranch,
				clctrl.GitopsTemplateURL,
				DigitaloceanDestinationMetaphorRepoURL,
				clctrl.ProviderConfig.K1Dir,
				clctrl.CreateTokens("gitops").(*providerConfigs.GitopsDirectoryValues),
				clctrl.ProviderConfig.MetaphorDir,
				clctrl.CreateTokens("metaphor").(*providerConfigs.MetaphorTokenValues),
				digitalocean.GetDomainApexContent(clctrl.DomainName),
				cl.GitProtocol,
			)
			if err != nil {
				return err
			}
		case "vultr":
			err = providerConfigs.PrepareGitRepositories(
				clctrl.CloudProvider,
				clctrl.GitProvider,
				clctrl.ClusterName,
				clctrl.ClusterType,
				VultrDestinationGitopsRepoURL,
				clctrl.ProviderConfig.GitopsDir,
				clctrl.GitopsTemplateBranch,
				clctrl.GitopsTemplateURL,
				VultrDestinationMetaphorRepoURL,
				clctrl.ProviderConfig.K1Dir,
				clctrl.CreateTokens("gitops").(*providerConfigs.GitopsDirectoryValues),
				clctrl.ProviderConfig.MetaphorDir,
				clctrl.CreateTokens("metaphor").(*providerConfigs.MetaphorTokenValues),
				vultr.GetDomainApexContent(clctrl.DomainName),
				cl.GitProtocol,
			)
			if err != nil {
				return err
			}
		}

		err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "gitops_ready_check", true)
		if err != nil {
			return err
		}

		log.Info("gitops repository initialized")
	}

	return nil
}

// RepositoryPush
func (clctrl *ClusterController) RepositoryPush() error {
	// Logging handler
	// Logs to stdout to maintain compatibility with event streaming
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "",
	})
	log.SetReportCaller(false)
	log.SetOutput(os.Stdout)

	cl, err := clctrl.MdbCl.GetCluster(clctrl.ClusterName)
	if err != nil {
		return err
	}

	// Telemetry handler
	segmentClient, err := telemetryShim.SetupTelemetry(cl)
	if err != nil {
		return err
	}
	defer segmentClient.Client.Close()

	if !cl.GitopsPushedCheck {
		publicKeys, err := gitssh.NewPublicKeys("git", []byte(cl.PrivateKey), "")
		if err != nil {
			log.Infof("generate public keys failed: %s\n", err.Error())
		}

		gitopsDir := clctrl.ProviderConfig.GitopsDir
		metaphorDir := clctrl.ProviderConfig.MetaphorDir
		DestinationGitopsRepoURL := clctrl.ProviderConfig.DestinationGitopsRepoURL
		destinationMetaphorRepoGitURL := clctrl.ProviderConfig.DestinationMetaphorRepoGitURL

		telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitopsRepoPushStarted, "")
		gitopsRepo, err := git.PlainOpen(gitopsDir)
		if err != nil {
			log.Infof("error opening repo at: %s", gitopsDir)
		}

		metaphorRepo, err := git.PlainOpen(metaphorDir)
		if err != nil {
			log.Infof("error opening repo at: %s", metaphorDir)
		}

		// For GitLab, we currently need to add an ssh key to the authenticating user
		if clctrl.GitProvider == "gitlab" {
			gitlabClient, err := gitlab.NewGitLabClient(clctrl.GitToken, clctrl.GitOwner)
			if err != nil {
				return err
			}

			keys, err := gitlabClient.GetUserSSHKeys()
			if err != nil {
				log.Errorf("unable to check for ssh keys in gitlab: %s", err.Error())
			}

			var keyName = "kbot-ssh-key"
			var keyFound bool = false
			for _, key := range keys {
				if key.Title == keyName {
					if strings.Contains(key.Key, strings.TrimSuffix(clctrl.PublicKey, "\n")) {
						log.Infof("ssh key %s already exists and key is up to date, continuing", keyName)
						keyFound = true
					} else {
						log.Errorf("ssh key %s already exists and key data has drifted - please remove before continuing", keyName)
					}
				}
			}
			if !keyFound {
				log.Infof("creating ssh key %s...", keyName)
				err := gitlabClient.AddUserSSHKey(keyName, clctrl.PublicKey)
				if err != nil {
					log.Errorf("error adding ssh key %s: %s", keyName, err.Error())
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
			msg := fmt.Sprintf("error pushing detokenized gitops repository to remote %s: %s", DestinationGitopsRepoURL, err)
			telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitopsRepoPushFailed, msg)
			return fmt.Errorf(msg)
		}

		// push metaphor repo to remote
		err = metaphorRepo.Push(
			&git.PushOptions{
				RemoteName: "origin",
				Auth:       publicKeys,
			},
		)
		if err != nil {
			msg := fmt.Sprintf("error pushing detokenized metaphor repository to remote %s: %s", destinationMetaphorRepoGitURL, err)
			telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitopsRepoPushFailed, msg)
			return fmt.Errorf(msg)
		}

		log.Infof("successfully pushed gitops and metaphor repositories to git@%s/%s", clctrl.GitHost, clctrl.GitOwner)
		// todo delete the local gitops repo and re-clone it
		// todo that way we can stop worrying about which origin we're going to push to
		telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricGitopsRepoPushCompleted, "")

		err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "gitops_pushed_check", true)
		if err != nil {
			return err
		}
	}

	return nil
}
