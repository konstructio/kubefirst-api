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
	awsinternal "github.com/kubefirst/runtime/pkg/aws"
	"github.com/kubefirst/runtime/pkg/civo"
	"github.com/kubefirst/runtime/pkg/digitalocean"
	"github.com/kubefirst/runtime/pkg/gitlab"
	"github.com/kubefirst/runtime/pkg/segment"
	"github.com/kubefirst/runtime/pkg/vultr"
	log "github.com/sirupsen/logrus"
)

// RepositoryPrep
func (clctrl *ClusterController) RepositoryPrep() error {
	cl, err := clctrl.MdbCl.GetCluster(clctrl.ClusterName)
	if err != nil {
		return err
	}

	if !cl.GitopsReadyCheck {
		switch clctrl.CloudProvider {
		case "aws":
			err := awsinternal.PrepareGitRepositories(
				clctrl.GitProvider,
				clctrl.ClusterName,
				clctrl.ClusterType,
				AWSDestinationGitopsRepoGitURL,
				clctrl.ProviderConfig.(*awsinternal.AwsConfig).GitopsDir,
				clctrl.GitopsTemplateBranchFlag,
				clctrl.GitopsTemplateURLFlag,
				AWSDestinationMetaphorRepoGitURL,
				clctrl.ProviderConfig.(*awsinternal.AwsConfig).K1Dir,
				clctrl.CreateTokens("gitops").(*awsinternal.GitOpsDirectoryValues),
				clctrl.ProviderConfig.(*awsinternal.AwsConfig).MetaphorDir,
				clctrl.CreateTokens("metaphor").(*awsinternal.MetaphorTokenValues),
			)
			if err != nil {
				return err
			}
		case "civo":
			err = civo.PrepareGitRepositories(
				clctrl.GitProvider,
				clctrl.ClusterName,
				clctrl.ClusterType,
				CivoDestinationGitopsRepoGitURL,
				clctrl.ProviderConfig.(*civo.CivoConfig).GitopsDir,
				clctrl.GitopsTemplateBranchFlag,
				clctrl.GitopsTemplateURLFlag,
				CivoDestinationMetaphorRepoGitURL,
				clctrl.ProviderConfig.(*civo.CivoConfig).K1Dir,
				clctrl.CreateTokens("gitops").(*civo.GitOpsDirectoryValues),
				clctrl.ProviderConfig.(*civo.CivoConfig).MetaphorDir,
				clctrl.CreateTokens("metaphor").(*civo.MetaphorTokenValues),
				civo.GetDomainApexContent(clctrl.DomainName),
			)
			if err != nil {
				return err
			}
		case "digitalocean":
			err = digitalocean.PrepareGitRepositories(
				clctrl.GitProvider,
				clctrl.ClusterName,
				clctrl.ClusterType,
				DigitaloceanDestinationGitopsRepoGitURL,
				clctrl.ProviderConfig.(*digitalocean.DigitaloceanConfig).GitopsDir,
				clctrl.GitopsTemplateBranchFlag,
				clctrl.GitopsTemplateURLFlag,
				DigitaloceanDestinationMetaphorRepoGitURL,
				clctrl.ProviderConfig.(*digitalocean.DigitaloceanConfig).K1Dir,
				clctrl.CreateTokens("gitops").(*digitalocean.GitOpsDirectoryValues),
				clctrl.ProviderConfig.(*digitalocean.DigitaloceanConfig).MetaphorDir,
				clctrl.CreateTokens("metaphor").(*digitalocean.MetaphorTokenValues),
				civo.GetDomainApexContent(clctrl.DomainName),
			)
			if err != nil {
				return err
			}
		case "vultr":
			err = vultr.PrepareGitRepositories(
				clctrl.GitProvider,
				clctrl.ClusterName,
				clctrl.ClusterType,
				VultrDestinationGitopsRepoGitURL,
				clctrl.ProviderConfig.(*vultr.VultrConfig).GitopsDir,
				clctrl.GitopsTemplateBranchFlag,
				clctrl.GitopsTemplateURLFlag,
				VultrDestinationMetaphorRepoGitURL,
				clctrl.ProviderConfig.(*vultr.VultrConfig).K1Dir,
				clctrl.CreateTokens("gitops").(*vultr.GitOpsDirectoryValues),
				clctrl.ProviderConfig.(*vultr.VultrConfig).MetaphorDir,
				clctrl.CreateTokens("metaphor").(*vultr.MetaphorTokenValues),
				civo.GetDomainApexContent(clctrl.DomainName),
			)
			if err != nil {
				return err
			}
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

		var gitopsDir, metaphorDir, destinationGitopsRepoGitURL, destinationMetaphorRepoGitURL string

		switch clctrl.CloudProvider {
		case "aws":
			gitopsDir = clctrl.ProviderConfig.(*awsinternal.AwsConfig).GitopsDir
			metaphorDir = clctrl.ProviderConfig.(*awsinternal.AwsConfig).MetaphorDir
			destinationGitopsRepoGitURL = clctrl.ProviderConfig.(*awsinternal.AwsConfig).DestinationGitopsRepoGitURL
			destinationMetaphorRepoGitURL = clctrl.ProviderConfig.(*awsinternal.AwsConfig).DestinationMetaphorRepoGitURL
		case "civo":
			gitopsDir = clctrl.ProviderConfig.(*civo.CivoConfig).GitopsDir
			metaphorDir = clctrl.ProviderConfig.(*civo.CivoConfig).MetaphorDir
			destinationGitopsRepoGitURL = clctrl.ProviderConfig.(*civo.CivoConfig).DestinationGitopsRepoGitURL
			destinationMetaphorRepoGitURL = clctrl.ProviderConfig.(*civo.CivoConfig).DestinationMetaphorRepoGitURL
		case "digitalocean":
			gitopsDir = clctrl.ProviderConfig.(*digitalocean.DigitaloceanConfig).GitopsDir
			metaphorDir = clctrl.ProviderConfig.(*digitalocean.DigitaloceanConfig).MetaphorDir
			destinationGitopsRepoGitURL = clctrl.ProviderConfig.(*digitalocean.DigitaloceanConfig).DestinationGitopsRepoGitURL
			destinationMetaphorRepoGitURL = clctrl.ProviderConfig.(*digitalocean.DigitaloceanConfig).DestinationMetaphorRepoGitURL
		case "vultr":
			gitopsDir = clctrl.ProviderConfig.(*vultr.VultrConfig).GitopsDir
			metaphorDir = clctrl.ProviderConfig.(*vultr.VultrConfig).MetaphorDir
			destinationGitopsRepoGitURL = clctrl.ProviderConfig.(*vultr.VultrConfig).DestinationGitopsRepoGitURL
			destinationMetaphorRepoGitURL = clctrl.ProviderConfig.(*vultr.VultrConfig).DestinationMetaphorRepoGitURL
		}

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
				log.Error("unable to check for ssh keys in gitlab: %s", err.Error())
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
			msg := fmt.Sprintf("error pushing detokenized gitops repository to remote %s: %s", destinationGitopsRepoGitURL, err)
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
