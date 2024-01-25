/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package services

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	v1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	argocdapi "github.com/argoproj/argo-cd/v2/pkg/client/clientset/versioned"
	health "github.com/argoproj/gitops-engine/pkg/health"
	"github.com/go-git/go-git/v5"
	githttps "github.com/go-git/go-git/v5/plumbing/transport/http"
	vaultapi "github.com/hashicorp/vault/api"
	"github.com/kubefirst/kubefirst-api/internal/constants"
	"github.com/kubefirst/kubefirst-api/internal/db"
	"github.com/kubefirst/kubefirst-api/internal/env"
	"github.com/kubefirst/kubefirst-api/internal/gitShim"
	"github.com/kubefirst/kubefirst-api/internal/gitopsCatalog"
	"github.com/kubefirst/kubefirst-api/internal/types"
	pkgtypes "github.com/kubefirst/kubefirst-api/pkg/types"
	"github.com/kubefirst/runtime/pkg/argocd"
	"github.com/kubefirst/runtime/pkg/gitClient"
	"github.com/kubefirst/runtime/pkg/k8s"
	"github.com/kubefirst/runtime/pkg/vault"
	log "github.com/rs/zerolog/log"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateService
func CreateService(cl *pkgtypes.Cluster, serviceName string, appDef *types.GitopsCatalogApp, req *types.GitopsCatalogAppCreateRequest) error {
	switch cl.Status {
	case constants.ClusterStatusDeleted, constants.ClusterStatusDeleting, constants.ClusterStatusError, constants.ClusterStatusProvisioning:
		return fmt.Errorf("cluster %s - unable to deploy service %s to cluster: cannot deploy services to a cluster in %s state", cl.ClusterName, serviceName, cl.Status)
	}

	homeDir, err := os.UserHomeDir()
	tmpGitopsDir := fmt.Sprintf("%s/.k1/%s/%s/gitops", homeDir, cl.ClusterName, serviceName)

	// Remove gitops dir
	err = os.RemoveAll(tmpGitopsDir)
	if err != nil {
		log.Fatal().Msgf("error removing gitops dir %s: %s", tmpGitopsDir, err)
		return err
	}

	err = gitShim.PrepareGitEnvironment(cl, tmpGitopsDir)
	if err != nil {
		log.Fatal().Msgf("an error ocurred preparing git environment %s %s", tmpGitopsDir, err)
	}

	gitopsRepo, _ := git.PlainOpen(tmpGitopsDir)

	var registryPath string
	if cl.CloudProvider == "civo" && cl.GitProvider == "github" {
		registryPath = fmt.Sprintf("registry/clusters/%s", cl.ClusterName)
	} else if cl.CloudProvider == "civo" && cl.GitProvider == "gitlab" {
		registryPath = fmt.Sprintf("registry/clusters/%s", cl.ClusterName)
	} else if cl.CloudProvider == "aws" && cl.GitProvider == "github" {
		registryPath = fmt.Sprintf("registry/clusters/%s", cl.ClusterName)
	} else if cl.CloudProvider == "aws" && cl.GitProvider == "gitlab" {
		registryPath = fmt.Sprintf("registry/clusters/%s", cl.ClusterName)
	} else if cl.CloudProvider == "google" && cl.GitProvider == "github" {
		registryPath = fmt.Sprintf("registry/clusters/%s", cl.ClusterName)
	} else if cl.CloudProvider == "google" && cl.GitProvider == "gitlab" {
		registryPath = fmt.Sprintf("registry/clusters/%s", cl.ClusterName)
	} else if cl.CloudProvider == "digitalocean" && cl.GitProvider == "github" {
		registryPath = fmt.Sprintf("registry/clusters/%s", cl.ClusterName)
	} else if cl.CloudProvider == "digitalocean" && cl.GitProvider == "gitlab" {
		registryPath = fmt.Sprintf("registry/clusters/%s", cl.ClusterName)
	} else if cl.CloudProvider == "vultr" && cl.GitProvider == "github" {
		registryPath = fmt.Sprintf("registry/clusters/%s", cl.ClusterName)
	} else if cl.CloudProvider == "vultr" && cl.GitProvider == "gitlab" {
		registryPath = fmt.Sprintf("registry/clusters/%s", cl.ClusterName)
	} else {
		registryPath = fmt.Sprintf("registry/%s", cl.ClusterName)
	}
	serviceFile := fmt.Sprintf("%s/%s/%s.yaml", tmpGitopsDir, registryPath, serviceName)

	var kcfg *k8s.KubernetesClient

	env, _ := env.GetEnv(constants.SilenceGetEnv)

	var inCluster bool = false
	if env.InCluster == "true" {
		inCluster = true
	}

	kcfg = k8s.CreateKubeConfig(inCluster, fmt.Sprintf("%s/kubeconfig", tmpGitopsDir))

	var fullDomainName string
	if cl.SubdomainName != "" {
		fullDomainName = fmt.Sprintf("%s.%s", cl.SubdomainName, cl.DomainName)
	} else {
		fullDomainName = cl.DomainName
	}

	// If there are secret values, create a vault secret
	if len(req.SecretKeys) > 0 {
		log.Info().Msgf("cluster %s - application %s has secrets, creating vault values", cl.ClusterName, appDef.Name)

		s := make(map[string]interface{}, 0)

		for _, secret := range req.SecretKeys {
			s[secret.Name] = secret.Value
		}

		// Get token
		existingKubernetesSecret, err := k8s.ReadSecretV2(kcfg.Clientset, vault.VaultNamespace, vault.VaultSecretName)
		if err != nil || existingKubernetesSecret == nil {
			return fmt.Errorf("cluster %s - error getting vault token: %s", cl.ClusterName, err)
		}

		vaultClient, err := vaultapi.NewClient(&vaultapi.Config{
			Address: fmt.Sprintf("https://vault.%s", fullDomainName),
		})
		if err != nil {
			return fmt.Errorf("cluster %s - error initializing vault client: %s", cl.ClusterName, err)
		}

		vaultClient.SetToken(existingKubernetesSecret["root-token"])

		resp, err := vaultClient.KVv2("secret").Put(context.Background(), appDef.Name, s)
		if err != nil {
			return fmt.Errorf("cluster %s - error putting vault secret: %s", cl.ClusterName, err)
		}

		log.Info().Msgf("cluster %s - created vault secret data for application %s %s", cl.ClusterName, appDef.Name, resp.VersionMetadata.CreatedTime)
	}

	// Create service files in gitops dir
	err = gitShim.PullWithAuth(
		gitopsRepo,
		"origin",
		"main",
		&githttps.BasicAuth{
			Username: cl.GitAuth.User,
			Password: cl.GitAuth.Token,
		},
	)
	if err != nil {
		log.Warn().Msgf("cluster %s - error pulling gitops repo: %s", cl.ClusterName, err)
	}
	files, err := gitopsCatalog.ReadApplicationDirectory(serviceName)
	if err != nil {
		return err
	}
	_, err = os.Create(serviceFile)
	if err != nil {
		return fmt.Errorf("cluster %s - error creating file: %s", cl.ClusterName, err)
	}
	f, err := os.OpenFile(serviceFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("cluster %s - error opening file: %s", cl.ClusterName, err)
	}
	// Regardless of how many files there are, loop over them and create a single yaml file
	for _, c := range files {
		_, err = f.WriteString(fmt.Sprintf("---\n%s\n", c))
		if err != nil {
			return err
		}
	}
	defer f.Close()

	// Commit to gitops repository
	err = gitClient.Commit(gitopsRepo, fmt.Sprintf("committing files for service %s", serviceName))
	if err != nil {
		return fmt.Errorf("cluster %s - error committing service file: %s", cl.ClusterName, err)
	}
	err = gitopsRepo.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth: &githttps.BasicAuth{
			Username: cl.GitAuth.User,
			Password: cl.GitAuth.Token,
		},
	})
	if err != nil {
		return fmt.Errorf("cluster %s - error pushing commit for service file: %s", cl.ClusterName, err)
	}

	// Add to list
	err = db.Client.InsertClusterServiceListEntry(cl.ClusterName, &types.Service{
		Name:        serviceName,
		Default:     false,
		Description: appDef.Description,
		Image:       appDef.ImageURL,
		Links:       []string{},
		Status:      "",
	})
	if err != nil {
		return err
	}

	// Wait for ArgoCD application sync
	argocdClient, err := argocdapi.NewForConfig(kcfg.RestConfig)
	if err != nil {
		return err
	}

	// Sync registry
	argoCDHost := fmt.Sprintf("https://argocd.%s", fullDomainName)
	if cl.CloudProvider == "k3d" {
		argoCDHost = "http://argocd-server.argocd.svc.cluster.local"
	}

	httpClient := http.Client{Timeout: time.Second * 10}
	argoCDToken, err := argocd.GetArgocdTokenV2(&httpClient, argoCDHost, "admin", cl.ArgoCDPassword)
	if err != nil {
		log.Warn().Msgf("error getting argocd token: %s", err)
		return err
	}
	err = argocd.RefreshRegistryApplication(argoCDHost, argoCDToken)
	if err != nil {
		log.Warn().Msgf("error refreshing registry application: %s", err)
		return err
	}

	// Wait for app to be created
	for i := 0; i < 50; i++ {
		_, err := argocdClient.ArgoprojV1alpha1().Applications("argocd").Get(context.Background(), serviceName, v1.GetOptions{})
		if err != nil {
			log.Info().Msgf("cluster %s - waiting for app %s to be created", cl.ClusterName, serviceName)
			time.Sleep(time.Second * 10)
		} else {
			break
		}
		if i == 50 {
			return fmt.Errorf("cluster %s - error waiting for app %s to be created: %s", cl.ClusterName, serviceName, err)
		}
	}

	// Wait for app to be synchronized and healthy
	for i := 0; i < 50; i++ {
		if i == 50 {
			return fmt.Errorf("cluster %s - error waiting for app %s to synchronize: %s", cl.ClusterName, serviceName, err)
		}
		app, err := argocdClient.ArgoprojV1alpha1().Applications("argocd").Get(context.Background(), serviceName, v1.GetOptions{})
		if err != nil {
			return fmt.Errorf("cluster %s - error getting argocd application %s: %s", cl.ClusterName, serviceName, err)
		}
		if app.Status.Sync.Status == v1alpha1.SyncStatusCodeSynced && app.Status.Health.Status == health.HealthStatusHealthy {
			log.Info().Msgf("cluster %s - app %s synchronized", cl.ClusterName, serviceName)
			break
		}
		log.Info().Msgf("cluster %s - waiting for app %s to sync", cl.ClusterName, serviceName)
		time.Sleep(time.Second * 10)
	}

	return nil
}

// DeleteService
func DeleteService(cl *pkgtypes.Cluster, serviceName string) error {
	var gitopsRepo *git.Repository

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal().Msgf("error getting home path: %s", err)
	}
	gitopsDir := fmt.Sprintf("%s/.k1/%s/gitops", homeDir, cl.ClusterName)
	gitopsRepo, _ = git.PlainOpen(gitopsDir)

	var registryPath string
	if cl.CloudProvider == "civo" && cl.GitProvider == "github" {
		registryPath = fmt.Sprintf("registry/clusters/%s", cl.ClusterName)
	} else if cl.CloudProvider == "civo" && cl.GitProvider == "gitlab" {
		registryPath = fmt.Sprintf("registry/clusters/%s", cl.ClusterName)
	} else if cl.CloudProvider == "aws" && cl.GitProvider == "github" {
		registryPath = fmt.Sprintf("registry/clusters/%s", cl.ClusterName)
	} else if cl.CloudProvider == "aws" && cl.GitProvider == "gitlab" {
		registryPath = fmt.Sprintf("registry/clusters/%s", cl.ClusterName)
	} else if cl.CloudProvider == "google" && cl.GitProvider == "github" {
		registryPath = fmt.Sprintf("registry/clusters/%s", cl.ClusterName)
	} else if cl.CloudProvider == "google" && cl.GitProvider == "gitlab" {
		registryPath = fmt.Sprintf("registry/clusters/%s", cl.ClusterName)
	} else if cl.CloudProvider == "vultr" && cl.GitProvider == "github" {
		registryPath = fmt.Sprintf("registry/clusters/%s", cl.ClusterName)
	} else if cl.CloudProvider == "vultr" && cl.GitProvider == "gitlab" {
		registryPath = fmt.Sprintf("registry/clusters/%s", cl.ClusterName)
	} else {
		registryPath = fmt.Sprintf("registry/%s", cl.ClusterName)
	}

	serviceFile := fmt.Sprintf("%s/%s/%s/%s.yaml", gitopsDir, registryPath, cl.ClusterName, serviceName)

	// Delete service files from gitops dir
	err = gitShim.PullWithAuth(
		gitopsRepo,
		cl.GitProvider,
		"main",
		&githttps.BasicAuth{
			Username: cl.GitAuth.User,
			Password: cl.GitAuth.Token,
		},
	)
	if err != nil {
		log.Warn().Msgf("cluster %s - error pulling gitops repo: %s", cl.ClusterName, err)
	}
	_, err = os.Stat(serviceFile)
	if err != nil {
		return fmt.Errorf("file %s does not exist in repository", serviceFile)
	} else {
		err := os.Remove(serviceFile)
		if err != nil {
			return fmt.Errorf("cluster %s - error deleting file: %s", cl.ClusterName, err)
		}
	}

	// Commit to gitops repository
	err = gitClient.Commit(gitopsRepo, fmt.Sprintf("deleting files for service %s", serviceName))
	if err != nil {
		return fmt.Errorf("cluster %s - error deleting service file: %s", cl.ClusterName, err)
	}
	err = gitopsRepo.Push(&git.PushOptions{
		RemoteName: cl.GitProvider,
		Auth: &githttps.BasicAuth{
			Username: cl.GitAuth.User,
			Password: cl.GitAuth.Token,
		},
	})
	if err != nil {
		return fmt.Errorf("cluster %s - error pushing commit for service file: %s", cl.ClusterName, err)
	}

	// Remove from list
	svc, err := db.Client.GetService(cl.ClusterName, serviceName)
	if err != nil {
		return fmt.Errorf("cluster %s - error finding service: %s", cl.ClusterName, err)
	}
	err = db.Client.DeleteClusterServiceListEntry(cl.ClusterName, &svc)
	if err != nil {
		return err
	}

	return nil
}

// AddDefaultServices
func AddDefaultServices(cl *pkgtypes.Cluster) error {
	err := db.Client.CreateClusterServiceList(cl)
	if err != nil {
		return err
	}

	var fullDomainName string
	if cl.SubdomainName != "" {
		fullDomainName = fmt.Sprintf("%s.%s", cl.SubdomainName, cl.DomainName)
	} else {
		fullDomainName = cl.DomainName
	}

	defaults := []types.Service{
		{
			Name:        cl.GitProvider,
			Default:     true,
			Description: "The git repositories contain all the Infrastructure as Code and Gitops configurations.",
			Image:       fmt.Sprintf("https://assets.kubefirst.com/console/%s.svg", cl.GitProvider),
			Links: []string{fmt.Sprintf("https://%s/%s/gitops", cl.GitHost, cl.GitAuth.Owner),
				fmt.Sprintf("https://%s/%s/metaphor", cl.GitHost, cl.GitAuth.Owner)},
			Status: "",
		},
		{
			Name:        "Vault",
			Default:     true,
			Description: "Kubefirst's secrets manager and identity provider.",
			Image:       "https://assets.kubefirst.com/console/vault.svg",
			Links:       []string{fmt.Sprintf("https://vault.%s", fullDomainName)},
			Status:      "",
		},
		{
			Name:        "Argo CD",
			Default:     true,
			Description: "A Gitops oriented continuous delivery tool for managing all of our applications across our Kubernetes clusters.",
			Image:       "https://assets.kubefirst.com/console/argocd.svg",
			Links:       []string{fmt.Sprintf("https://argocd.%s", fullDomainName)},
			Status:      "",
		},
		{
			Name:        "Argo Workflows",
			Default:     true,
			Description: "The workflow engine for orchestrating parallel jobs on Kubernetes.",
			Image:       "https://assets.kubefirst.com/console/argocd.svg",
			Links:       []string{fmt.Sprintf("https://argo.%s/workflows", fullDomainName)},
			Status:      "",
		},
		{
			Name:        "Atlantis",
			Default:     true,
			Description: "Kubefirst manages Terraform workflows with Atlantis automation.",
			Image:       "https://assets.kubefirst.com/console/atlantis.svg",
			Links:       []string{fmt.Sprintf("https://atlantis.%s", fullDomainName)},
			Status:      "",
		},
		{
			Name:        "Metaphor",
			Default:     true,
			Description: "A multi-environment demonstration space for frontend application best practices that's easy to apply to other projects.",
			Image:       "https://assets.kubefirst.com/console/metaphor.svg",
			Links: []string{fmt.Sprintf("https://metaphor-development.%s", fullDomainName),
				fmt.Sprintf("https://metaphor-staging.%s", fullDomainName),
				fmt.Sprintf("https://metaphor-production.%s", fullDomainName)},
			Status: "",
		},
	}

	for _, svc := range defaults {
		err := db.Client.InsertClusterServiceListEntry(cl.ClusterName, &svc)
		if err != nil {
			return err
		}
	}

	return nil
}
