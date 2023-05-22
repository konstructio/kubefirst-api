/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package services

import (
	"fmt"
	"log"
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/kubefirst/kubefirst-api/internal/db"
	"github.com/kubefirst/kubefirst-api/internal/marketplace"
	"github.com/kubefirst/kubefirst-api/internal/types"
	"github.com/kubefirst/runtime/pkg/gitClient"
)

// CreateService
func CreateService(cl *types.Cluster, serviceName string, appDef *types.MarketplaceApp, req *types.MarketplaceAppCreateRequest) error {
	var gitopsRepo *git.Repository

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("error getting home path: %s", err)
	}
	gitopsDir := fmt.Sprintf("%s/.k1/%s/gitops", homeDir, cl.ClusterName)
	gitopsRepo, _ = git.PlainOpen(gitopsDir)

	// Create service files in gitops dir
	f, err := marketplace.ReadApplicationDirectory(serviceName)
	if err != nil {
		return err
	}

	for _, c := range f {
		err = os.WriteFile(fmt.Sprintf("%s/registry/%s/%s.yaml", gitopsDir, cl.ClusterName, serviceName), c, 0644)
		if err != nil {
			return err
		}
	}

	err = gitClient.Commit(gitopsRepo, fmt.Sprintf("committing files for service %s", serviceName))
	if err != nil {
		return err
	}

	publicKeys, err := ssh.NewPublicKeys("git", []byte(cl.PrivateKey), "")
	if err != nil {
		return err
	}
	err = gitopsRepo.Push(&git.PushOptions{
		RemoteName: cl.GitProvider,
		Auth:       publicKeys,
	})
	if err != nil {
		return err
	}

	// Update services to feature new service
	// Wait for ArgoCD success in here somewhere?

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

	return nil
}

// AddDefaultServices
func AddDefaultServices(cl *types.Cluster) error {
	err := db.Client.CreateClusterServiceList(cl)
	if err != nil {
		return err
	}

	defaults := []types.Service{
		{
			Name:        "Argo CD",
			Default:     true,
			Description: "A GitOps oriented continuous delivery tool for managing all of our applications across our Kubernetes clusters.",
			Image:       "https://assets.kubefirst.com/console/argocd.svg",
			Links:       []string{fmt.Sprintf("https://argocd.%s", cl.DomainName)},
			Status:      "",
		},
		{
			Name:        "Argo Workflows",
			Default:     true,
			Description: "The workflow engine for orchestrating parallel jobs on Kubernetes.",
			Image:       "https://assets.kubefirst.com/console/argocd.svg",
			Links:       []string{fmt.Sprintf("https://argo.%s/workflows", cl.DomainName)},
			Status:      "",
		},
		{
			Name:        "Atlantis",
			Default:     true,
			Description: "Kubefirst manages Terraform workflows with Atlantis automation.",
			Image:       "https://assets.kubefirst.com/console/atlantis.svg",
			Links:       []string{fmt.Sprintf("https://atlantis.%s", cl.DomainName)},
			Status:      "",
		},
		{
			Name:        cl.GitProvider,
			Default:     true,
			Description: "The git repositories contain all the Infrastructure as Code and GitOps configurations.",
			Image:       fmt.Sprintf("https://assets.kubefirst.com/console/%s.svg", cl.GitProvider),
			Links: []string{fmt.Sprintf("https://%s/%s/gitops", cl.GitHost, cl.GitOwner),
				fmt.Sprintf("https://%s/%s/metaphor", cl.GitHost, cl.GitOwner)},
			Status: "",
		},
		{
			Name:        "Metaphor",
			Default:     true,
			Description: "A multi-environment demonstration space for frontend application best practices that's easy to apply to other projects.",
			Image:       "https://assets.kubefirst.com/console/metaphor.svg",
			Links: []string{fmt.Sprintf("https://metaphor-development.%s", cl.DomainName),
				fmt.Sprintf("https://metaphor-staging.%s", cl.DomainName),
				fmt.Sprintf("https://metaphor-production.%s", cl.DomainName)},
			Status: "",
		},
		{
			Name:        "Vault",
			Default:     true,
			Description: "Kubefirst's secrets manager and identity provider.",
			Image:       "https://assets.kubefirst.com/console/vault.svg",
			Links:       []string{fmt.Sprintf("https://vault.%s", cl.DomainName)},
			Status:      "",
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
