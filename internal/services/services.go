/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package services

import (
	"context"
	"fmt"
	"os"
	"time"

	v1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	argocdapi "github.com/argoproj/argo-cd/v2/pkg/client/clientset/versioned"
	health "github.com/argoproj/gitops-engine/pkg/health"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	awsext "github.com/kubefirst/kubefirst-api/extensions/aws"
	"github.com/kubefirst/kubefirst-api/internal/db"
	"github.com/kubefirst/kubefirst-api/internal/marketplace"
	"github.com/kubefirst/kubefirst-api/internal/types"
	awsinternal "github.com/kubefirst/runtime/pkg/aws"
	"github.com/kubefirst/runtime/pkg/gitClient"
	"github.com/kubefirst/runtime/pkg/k8s"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateService
func CreateService(cl *types.Cluster, serviceName string, appDef *types.MarketplaceApp, req *types.MarketplaceAppCreateRequest) error {
	var gitopsRepo *git.Repository

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("error getting home path: %s", err)
	}
	clusterDir := fmt.Sprintf("%s/.k1/%s", homeDir, cl.ClusterName)
	gitopsDir := fmt.Sprintf("%s/.k1/%s/gitops", homeDir, cl.ClusterName)
	gitopsRepo, _ = git.PlainOpen(gitopsDir)
	serviceFile := fmt.Sprintf("%s/registry/%s/%s.yaml", gitopsDir, cl.ClusterName, serviceName)

	// Create service files in gitops dir
	err = gitClient.Pull(gitopsRepo, cl.GitProvider, "main")
	if err != nil {
		log.Warnf("cluster %s - error pulling gitops repo: %s", cl.ClusterName, err)
	}
	files, err := marketplace.ReadApplicationDirectory(serviceName)
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
	f.Close()

	// Commit to gitops repository
	err = gitClient.Commit(gitopsRepo, fmt.Sprintf("committing files for service %s", serviceName))
	if err != nil {
		return fmt.Errorf("cluster %s - error committing service file: %s", cl.ClusterName, err)
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
	var kcfg *k8s.KubernetesClient

	switch cl.CloudProvider {
	case "aws":
		awscfg := awsinternal.NewAwsV3(
			cl.CloudRegion,
			cl.AWSAuth.AccessKeyID,
			cl.AWSAuth.SecretAccessKey,
			cl.AWSAuth.SessionToken,
		)
		kcfg = awsext.CreateEKSKubeconfig(&awscfg, cl.ClusterName)
	case "civo", "digitalocean", "vultr":
		kcfg = k8s.CreateKubeConfig(false, fmt.Sprintf("%s/kubeconfig", clusterDir))
	}

	argocdClient, err := argocdapi.NewForConfig(kcfg.RestConfig)
	if err != nil {
		return err
	}

	// Sync registry
	registryApplication, err := argocdClient.ArgoprojV1alpha1().Applications("argocd").Get(context.Background(), "registry", v1.GetOptions{})
	if err != nil {
		log.Warnf("cluster %s - could not get registry application data: %s", cl.ClusterName, err)
	}
	registryApplication.SetAnnotations(map[string]string{"argocd.argoproj.io/refresh": "hard"})

	// Wait for app to be created
	for i := 0; i < 50; i++ {
		_, err := argocdClient.ArgoprojV1alpha1().Applications("argocd").Get(context.Background(), serviceName, v1.GetOptions{})
		if err != nil {
			log.Infof("cluster %s - waiting for app %s to be created", cl.ClusterName, serviceName)
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
			log.Infof("cluster %s - app %s synchronized", cl.ClusterName, serviceName)
			break
		}
		log.Infof("cluster %s - waiting for app %s to sync", cl.ClusterName, serviceName)
		time.Sleep(time.Second * 10)
	}

	return nil
}

// DeleteService
func DeleteService(cl *types.Cluster, serviceName string) error {
	var gitopsRepo *git.Repository

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("error getting home path: %s", err)
	}
	gitopsDir := fmt.Sprintf("%s/.k1/%s/gitops", homeDir, cl.ClusterName)
	gitopsRepo, _ = git.PlainOpen(gitopsDir)
	serviceFile := fmt.Sprintf("%s/registry/%s/%s.yaml", gitopsDir, cl.ClusterName, serviceName)

	// Delete service files from gitops dir
	err = gitClient.Pull(gitopsRepo, cl.GitProvider, "main")
	if err != nil {
		log.Warnf("cluster %s - error pulling gitops repo: %s", cl.ClusterName, err)
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
	publicKeys, err := ssh.NewPublicKeys("git", []byte(cl.PrivateKey), "")
	if err != nil {
		return err
	}
	err = gitopsRepo.Push(&git.PushOptions{
		RemoteName: cl.GitProvider,
		Auth:       publicKeys,
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
