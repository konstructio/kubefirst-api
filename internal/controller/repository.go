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
	githttps "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/konstructio/kubefirst-api/internal/civo"
	"github.com/konstructio/kubefirst-api/internal/digitalocean"
	"github.com/konstructio/kubefirst-api/internal/gitlab"
	"github.com/konstructio/kubefirst-api/internal/secrets"
	"github.com/konstructio/kubefirst-api/internal/vultr"
	google "github.com/konstructio/kubefirst-api/pkg/google"
	"github.com/konstructio/kubefirst-api/pkg/providerConfigs"
	"github.com/kubefirst/metrics-client/pkg/telemetry"
	log "github.com/rs/zerolog/log"
)

// RepositoryPrep
func (clctrl *ClusterController) RepositoryPrep() error {
	cl, err := secrets.GetCluster(clctrl.KubernetesClient, clctrl.ClusterName)
	if err != nil {
		return fmt.Errorf("error getting cluster for %q: %w", clctrl.ClusterName, err)
	}

	useCloudflareOriginIssuer := false
	if cl.CloudflareAuth.OriginCaIssuerKey != "" {
		useCloudflareOriginIssuer = true
	}

	// TODO Implement an interface so we can call GetDomainApexContent on the clustercotroller

	if !cl.GitopsReadyCheck {
		log.Info().Msg("initializing the gitops repository - this may take several minutes")

		switch clctrl.CloudProvider {
		case "akamai":
			err := providerConfigs.PrepareGitRepositories(
				clctrl.CloudProvider,
				clctrl.GitProvider,
				clctrl.ClusterName,
				clctrl.ClusterType,
				clctrl.ProviderConfig.DestinationGitopsRepoURL,
				clctrl.ProviderConfig.GitopsDir,
				clctrl.GitopsTemplateBranch,
				clctrl.GitopsTemplateURL,
				clctrl.ProviderConfig.DestinationMetaphorRepoURL,
				clctrl.ProviderConfig.K1Dir,
				clctrl.CreateTokens("gitops").(*providerConfigs.GitopsDirectoryValues), // tokens created on the fly
				clctrl.ProviderConfig.MetaphorDir,
				clctrl.CreateTokens("metaphor").(*providerConfigs.MetaphorTokenValues), // tokens created on the fly
				civo.GetDomainApexContent(clctrl.DomainName),
				cl.GitProtocol,
				useCloudflareOriginIssuer,
			)
			if err != nil {
				return fmt.Errorf("error preparing git repositories for akamai: %w", err)
			}
		case "aws":
			err := providerConfigs.PrepareGitRepositories(
				clctrl.CloudProvider,
				clctrl.GitProvider,
				clctrl.ClusterName,
				clctrl.ClusterType,
				clctrl.ProviderConfig.DestinationGitopsRepoURL,
				clctrl.ProviderConfig.GitopsDir,
				clctrl.GitopsTemplateBranch,
				clctrl.GitopsTemplateURL,
				clctrl.ProviderConfig.DestinationMetaphorRepoURL,
				clctrl.ProviderConfig.K1Dir,
				clctrl.CreateTokens("gitops").(*providerConfigs.GitopsDirectoryValues), // tokens created on the fly
				clctrl.ProviderConfig.MetaphorDir,
				clctrl.CreateTokens("metaphor").(*providerConfigs.MetaphorTokenValues), // tokens created on the fly
				civo.GetDomainApexContent(clctrl.DomainName),
				cl.GitProtocol,
				useCloudflareOriginIssuer,
			)
			if err != nil {
				return fmt.Errorf("error preparing git repositories for AWS: %w", err)
			}
		case "azure":
			err := providerConfigs.PrepareGitRepositories(
				clctrl.CloudProvider,
				clctrl.GitProvider,
				clctrl.ClusterName,
				clctrl.ClusterType,
				clctrl.ProviderConfig.DestinationGitopsRepoURL,
				clctrl.ProviderConfig.GitopsDir,
				clctrl.GitopsTemplateBranch,
				clctrl.GitopsTemplateURL,
				clctrl.ProviderConfig.DestinationMetaphorRepoURL,
				clctrl.ProviderConfig.K1Dir,
				clctrl.CreateTokens("gitops").(*providerConfigs.GitopsDirectoryValues), // tokens created on the fly
				clctrl.ProviderConfig.MetaphorDir,
				clctrl.CreateTokens("metaphor").(*providerConfigs.MetaphorTokenValues), // tokens created on the fly
				true,
				cl.GitProtocol,
				useCloudflareOriginIssuer,
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
				clctrl.ProviderConfig.DestinationGitopsRepoURL,
				clctrl.ProviderConfig.GitopsDir,
				clctrl.GitopsTemplateBranch,
				clctrl.GitopsTemplateURL,
				clctrl.ProviderConfig.DestinationMetaphorRepoURL,
				clctrl.ProviderConfig.K1Dir,
				clctrl.CreateTokens("gitops").(*providerConfigs.GitopsDirectoryValues), // tokens created on the fly
				clctrl.ProviderConfig.MetaphorDir,
				clctrl.CreateTokens("metaphor").(*providerConfigs.MetaphorTokenValues), // tokens created on the fly
				civo.GetDomainApexContent(clctrl.DomainName),
				cl.GitProtocol,
				useCloudflareOriginIssuer,
			)
			if err != nil {
				return fmt.Errorf("error preparing git repositories for Civo: %w", err)
			}
		case "google":
			err := providerConfigs.PrepareGitRepositories(
				clctrl.CloudProvider,
				clctrl.GitProvider,
				clctrl.ClusterName,
				clctrl.ClusterType,
				clctrl.ProviderConfig.DestinationGitopsRepoURL,
				clctrl.ProviderConfig.GitopsDir,
				clctrl.GitopsTemplateBranch,
				clctrl.GitopsTemplateURL,
				clctrl.ProviderConfig.DestinationMetaphorRepoURL,
				clctrl.ProviderConfig.K1Dir,
				clctrl.CreateTokens("gitops").(*providerConfigs.GitopsDirectoryValues), // tokens created on the fly
				clctrl.ProviderConfig.MetaphorDir,
				clctrl.CreateTokens("metaphor").(*providerConfigs.MetaphorTokenValues), // tokens created on the fly
				google.GetDomainApexContent(clctrl.DomainName),
				cl.GitProtocol,
				useCloudflareOriginIssuer,
			)
			if err != nil {
				return fmt.Errorf("error preparing git repositories for Google: %w", err)
			}
		case "digitalocean":
			err = providerConfigs.PrepareGitRepositories(
				clctrl.CloudProvider,
				clctrl.GitProvider,
				clctrl.ClusterName,
				clctrl.ClusterType,
				clctrl.ProviderConfig.DestinationGitopsRepoURL,
				clctrl.ProviderConfig.GitopsDir,
				clctrl.GitopsTemplateBranch,
				clctrl.GitopsTemplateURL,
				clctrl.ProviderConfig.DestinationMetaphorRepoURL,
				clctrl.ProviderConfig.K1Dir,
				clctrl.CreateTokens("gitops").(*providerConfigs.GitopsDirectoryValues), // tokens created on the fly
				clctrl.ProviderConfig.MetaphorDir,
				clctrl.CreateTokens("metaphor").(*providerConfigs.MetaphorTokenValues), // tokens created on the fly
				digitalocean.GetDomainApexContent(clctrl.DomainName),
				cl.GitProtocol,
				useCloudflareOriginIssuer,
			)
			if err != nil {
				return fmt.Errorf("error preparing git repositories for DigitalOcean: %w", err)
			}
		case "vultr":
			err = providerConfigs.PrepareGitRepositories(
				clctrl.CloudProvider,
				clctrl.GitProvider,
				clctrl.ClusterName,
				clctrl.ClusterType,
				clctrl.ProviderConfig.DestinationGitopsRepoURL,
				clctrl.ProviderConfig.GitopsDir,
				clctrl.GitopsTemplateBranch,
				clctrl.GitopsTemplateURL,
				clctrl.ProviderConfig.DestinationMetaphorRepoURL,
				clctrl.ProviderConfig.K1Dir,
				clctrl.CreateTokens("gitops").(*providerConfigs.GitopsDirectoryValues), // tokens created on the fly
				clctrl.ProviderConfig.MetaphorDir,
				clctrl.CreateTokens("metaphor").(*providerConfigs.MetaphorTokenValues), // tokens created on the fly
				vultr.GetDomainApexContent(clctrl.DomainName),
				cl.GitProtocol,
				useCloudflareOriginIssuer,
			)
			if err != nil {
				return fmt.Errorf("error preparing git repositories for Vultr: %w", err)
			}

		case "k3s":
			err = providerConfigs.PrepareGitRepositories(
				clctrl.CloudProvider,
				clctrl.GitProvider,
				clctrl.ClusterName,
				clctrl.ClusterType,
				clctrl.ProviderConfig.DestinationGitopsRepoURL,
				clctrl.ProviderConfig.GitopsDir,
				clctrl.GitopsTemplateBranch,
				clctrl.GitopsTemplateURL,
				clctrl.ProviderConfig.DestinationMetaphorRepoURL,
				clctrl.ProviderConfig.K1Dir,
				clctrl.CreateTokens("gitops").(*providerConfigs.GitopsDirectoryValues), // tokens created on the fly
				clctrl.ProviderConfig.MetaphorDir,
				clctrl.CreateTokens("metaphor").(*providerConfigs.MetaphorTokenValues), // tokens created on the fly
				vultr.GetDomainApexContent(clctrl.DomainName),
				cl.GitProtocol,
				useCloudflareOriginIssuer,
			)
			if err != nil {
				return fmt.Errorf("error preparing git repositories for K3s: %w", err)
			}
		}

		if !clctrl.InstallKubefirstPro {
			kubefirstComponentsLocation := fmt.Sprintf("%s/registry/clusters/%s/components/kubefirst", clctrl.ProviderConfig.GitopsDir, clctrl.ClusterName)
			kubefirstRegistryLocation := fmt.Sprintf("%s/registry/clusters/%s/kubefirst.yaml", clctrl.ProviderConfig.GitopsDir, clctrl.ClusterName)

			os.RemoveAll(kubefirstComponentsLocation)
			os.Remove(kubefirstRegistryLocation)
		}

		clctrl.Cluster.GitopsReadyCheck = true
		err = secrets.UpdateCluster(clctrl.KubernetesClient, clctrl.Cluster)
		if err != nil {
			return fmt.Errorf("error updating cluster %q: %w", clctrl.ClusterName, err)
		}

		log.Info().Msg("gitops repository initialized")
	}

	return nil
}

// RepositoryPush
func (clctrl *ClusterController) RepositoryPush() error {
	cl, err := secrets.GetCluster(clctrl.KubernetesClient, clctrl.ClusterName)
	if err != nil {
		return fmt.Errorf("error getting cluster %q: %w", clctrl.ClusterName, err)
	}

	if !cl.GitopsPushedCheck {
		gitopsDir := clctrl.ProviderConfig.GitopsDir
		metaphorDir := clctrl.ProviderConfig.MetaphorDir

		telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.GitopsRepoPushStarted, "")
		gitopsRepo, err := git.PlainOpen(gitopsDir)
		if err != nil {
			return fmt.Errorf("error opening gitops repo at %q: %w", gitopsDir, err)
		}

		metaphorRepo, err := git.PlainOpen(metaphorDir)
		if err != nil {
			return fmt.Errorf("error opening metaphor repo at %q: %w", metaphorDir, err)
		}

		// For GitLab, we currently need to add an ssh key to the authenticating user
		if clctrl.GitProvider == "gitlab" {
			gitlabClient, err := gitlab.NewGitLabClient(clctrl.GitAuth.Token, clctrl.GitAuth.Owner)
			if err != nil {
				return fmt.Errorf("error creating gitlab client for %q: %w", clctrl.GitAuth.Owner, err)
			}

			keys, err := gitlabClient.GetUserSSHKeys()
			if err != nil {
				log.Error().Msgf("unable to check for ssh keys in gitlab: %s", err.Error())
			}

			keyName := "kbot-ssh-key"
			keyFound := false
			for _, key := range keys {
				if key.Title == keyName {
					if strings.Contains(key.Key, strings.TrimSuffix(clctrl.GitAuth.PublicKey, "\n")) {
						log.Info().Msgf("ssh key %s already exists and key is up to date, continuing", keyName)
						keyFound = true
					} else {
						log.Error().Msgf("ssh key %s already exists and key data has drifted - please remove before continuing", keyName)
					}
				}
			}
			if !keyFound {
				log.Info().Msgf("creating ssh key %s...", keyName)
				err := gitlabClient.AddUserSSHKey(keyName, clctrl.GitAuth.PublicKey)
				if err != nil {
					log.Error().Msgf("error adding ssh key %q: %s", keyName, err.Error())
				}
			}
		}

		// push metaphor repo to remote
		err = gitopsRepo.Push(
			&git.PushOptions{
				RemoteName: clctrl.GitProvider,
				Auth: &githttps.BasicAuth{
					Username: clctrl.GitAuth.User,
					Password: clctrl.GitAuth.Token,
				},
			},
		)
		if err != nil {
			telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.GitopsRepoPushFailed, err.Error())
			return fmt.Errorf("error pushing detokenized gitops repository to remote %s: %w", clctrl.ProviderConfig.DestinationGitopsRepoURL, err)
		}

		// push metaphor repo to remote
		err = metaphorRepo.Push(
			&git.PushOptions{
				RemoteName: "origin",
				Auth: &githttps.BasicAuth{
					Username: clctrl.GitAuth.User,
					Password: clctrl.GitAuth.Token,
				},
			},
		)
		if err != nil {
			telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.GitopsRepoPushFailed, err.Error())
			return fmt.Errorf("error pushing detokenized metaphor repository to remote %s: %w", clctrl.ProviderConfig.DestinationMetaphorRepoURL, err)
		}

		log.Info().Msgf("successfully pushed gitops and metaphor repositories to git@%s/%s", clctrl.GitHost, clctrl.GitAuth.Owner)
		// todo delete the local gitops repo and re-clone it
		// todo that way we can stop worrying about which origin we're going to push to
		telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.GitopsRepoPushCompleted, "")

		clctrl.Cluster.GitopsPushedCheck = true
		err = secrets.UpdateCluster(clctrl.KubernetesClient, clctrl.Cluster)
		if err != nil {
			return fmt.Errorf("error updating cluster %q: %w", clctrl.ClusterName, err)
		}
	}

	return nil
}
