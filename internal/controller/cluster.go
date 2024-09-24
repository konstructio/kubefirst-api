/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	akamaiext "github.com/konstructio/kubefirst-api/extensions/akamai"
	awsext "github.com/konstructio/kubefirst-api/extensions/aws"
	azureext "github.com/konstructio/kubefirst-api/extensions/azure"
	civoext "github.com/konstructio/kubefirst-api/extensions/civo"
	digitaloceanext "github.com/konstructio/kubefirst-api/extensions/digitalocean"
	googleext "github.com/konstructio/kubefirst-api/extensions/google"
	k3sext "github.com/konstructio/kubefirst-api/extensions/k3s"
	terraformext "github.com/konstructio/kubefirst-api/extensions/terraform"
	vultrext "github.com/konstructio/kubefirst-api/extensions/vultr"
	"github.com/konstructio/kubefirst-api/internal/constants"
	"github.com/konstructio/kubefirst-api/internal/env"
	gitShim "github.com/konstructio/kubefirst-api/internal/gitShim"
	"github.com/konstructio/kubefirst-api/internal/k8s"
	"github.com/konstructio/kubefirst-api/internal/secrets"
	"github.com/konstructio/kubefirst-api/pkg/providerConfigs"
	"github.com/kubefirst/metrics-client/pkg/telemetry"
	log "github.com/rs/zerolog/log"
	"github.com/thanhpk/randstr"
	v1 "k8s.io/api/apps/v1"
)

// CreateCluster
func (clctrl *ClusterController) CreateCluster() error {
	cl, err := secrets.GetCluster(clctrl.KubernetesClient, clctrl.ClusterName)
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	if !cl.CloudTerraformApplyCheck || cl.CloudTerraformApplyFailedCheck {
		log.Info().Msg("creating aws cloud resources with terraform")
		tfEntrypoint := clctrl.ProviderConfig.GitopsDir + fmt.Sprintf("/terraform/%s", clctrl.CloudProvider)
		tfEnvs := map[string]string{}

		telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.CloudTerraformApplyStarted, "")

		log.Info().Msgf("creating %s cluster", clctrl.CloudProvider)

		switch clctrl.CloudProvider {
		case "akamai":
			tfEnvs = akamaiext.GetAkamaiTerraformEnvs(tfEnvs, cl)
		case "aws":
			tfEnvs = awsext.GetAwsTerraformEnvs(tfEnvs, cl)
			iamCaller, err := clctrl.AwsClient.GetCallerIdentity()
			if err != nil {
				return fmt.Errorf("error getting AWS caller identity: %w", err)
			}
			tfEnvs["TF_VAR_aws_account_id"] = *iamCaller.Account
			tfEnvs["TF_VAR_use_ecr"] = strconv.FormatBool(clctrl.ECR) // Flag out the ecr terraform

			clctrl.Cluster.AWSAccountID = *iamCaller.Account
			err = secrets.UpdateCluster(clctrl.KubernetesClient, clctrl.Cluster)
			if err != nil {
				return fmt.Errorf("failed to update cluster after getting AWS account ID: %w", err)
			}
		case "azure":
			tfEnvs = azureext.GetAzureTerraformEnvs(tfEnvs, cl)
		case "civo":
			tfEnvs = civoext.GetCivoTerraformEnvs(tfEnvs, cl)
		case "digitalocean":
			tfEnvs = digitaloceanext.GetDigitaloceanTerraformEnvs(tfEnvs, cl)
		case "google":
			tfEnvs = googleext.GetGoogleTerraformEnvs(tfEnvs, cl)
		case "vultr":
			tfEnvs = vultrext.GetVultrTerraformEnvs(tfEnvs, cl)
		case "k3s":
			tfEnvs = k3sext.GetK3sTerraformEnvs(tfEnvs, cl)
		}

		err := terraformext.InitApplyAutoApprove(clctrl.ProviderConfig.TerraformClient, tfEntrypoint, tfEnvs)
		if err != nil {
			log.Error().Msgf("error applying cloud terraform: %s", err)

			log.Info().Msg("sleeping 10 seconds before retrying terraform execution once more")
			time.Sleep(10 * time.Second)

			err = terraformext.InitApplyAutoApprove(clctrl.ProviderConfig.TerraformClient, tfEntrypoint, tfEnvs)
			if err != nil {
				telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.CloudTerraformApplyFailed, err.Error())
				clctrl.Cluster.CloudTerraformApplyFailedCheck = true

				if err := secrets.UpdateCluster(clctrl.KubernetesClient, clctrl.Cluster); err != nil {
					telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.CloudTerraformApplyFailed, err.Error())
					return fmt.Errorf("failed to update cluster after terraform apply failed: %w", err)
				}

				log.Error().Msgf("error creating %s resources with terraform %s: %s", clctrl.CloudProvider, tfEntrypoint, err)
				return fmt.Errorf("error creating %s resources with terraform %s: %w", clctrl.CloudProvider, tfEntrypoint, err)
			}
		}

		log.Info().Msgf("created %s cloud resources", clctrl.CloudProvider)
		telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.CloudTerraformApplyCompleted, "")

		clctrl.Cluster.CloudTerraformApplyCheck = true
		clctrl.Cluster.CloudTerraformApplyFailedCheck = false
		err = secrets.UpdateCluster(clctrl.KubernetesClient, clctrl.Cluster)
		if err != nil {
			return fmt.Errorf("failed to update cluster state after creating cloud resources: %w", err)
		}
	}

	return nil
}

// CreateTokens
func (clctrl *ClusterController) CreateTokens(kind string) interface{} {
	cl, err := secrets.GetCluster(clctrl.KubernetesClient, clctrl.ClusterName)
	if err != nil {
		return fmt.Errorf("failed to get cluster while creating tokens: %w", err)
	}

	var fullDomainName string

	if clctrl.SubdomainName != "" {
		fullDomainName = fmt.Sprintf("%s.%s", clctrl.SubdomainName, clctrl.DomainName)
	} else {
		fullDomainName = clctrl.DomainName
	}

	// handle set gitops tokens/values
	switch kind {
	case "gitops": // repo name

		var externalDNSProviderTokenEnvName, externalDNSProviderSecretKey string
		if clctrl.DNSProvider == "cloudflare" {
			externalDNSProviderTokenEnvName = "CF_API_TOKEN"
			externalDNSProviderSecretKey = "cf-api-token"
		} else {
			switch clctrl.CloudProvider {
			// provider auth secret gets mapped to these values
			case "aws":
				externalDNSProviderTokenEnvName = "not-used-uses-service-account"
			case "azure":
				externalDNSProviderTokenEnvName = "not-used-uses-managed-service-principal"
			case "google":
				// Normally this would be GOOGLE_APPLICATION_CREDENTIALS but we are using a service account instead and
				// if you set externalDNSProviderTokenEnvName to GOOGLE_APPLICATION_CREDENTIALS then externaldns will overlook the service account
				// if you want to use the provided keyfile instead of a service account then set the var accordingly
				externalDNSProviderTokenEnvName = fmt.Sprintf("%s_auth", strings.ToUpper(clctrl.CloudProvider))
			case "civo":
				externalDNSProviderTokenEnvName = fmt.Sprintf("%s_TOKEN", strings.ToUpper(clctrl.CloudProvider))
			case "vultr":
				externalDNSProviderTokenEnvName = fmt.Sprintf("%s_API_KEY", strings.ToUpper(clctrl.CloudProvider))
			case "digitalocean":
				externalDNSProviderTokenEnvName = "DO_TOKEN"
			}
			externalDNSProviderSecretKey = fmt.Sprintf("%s-auth", clctrl.CloudProvider)
		}

		// switch repo url based on gitProtocol and gitlab group parents.
		destinationGitopsRepoURL, err := clctrl.GetRepoURL()
		if err != nil {
			return fmt.Errorf("failed to get repo URL for gitops tokens: %w", err)
		}

		env, _ := env.GetEnv(constants.SilenceGetEnv)

		// Default gitopsTemplateTokens
		gitopsTemplateTokens := &providerConfigs.GitopsDirectoryValues{
			AlertsEmail:               clctrl.AlertsEmail,
			AtlantisAllowList:         fmt.Sprintf("%s/%s/*", clctrl.GitHost, clctrl.GitAuth.Owner),
			CloudProvider:             clctrl.CloudProvider,
			CloudRegion:               clctrl.CloudRegion,
			ClusterName:               clctrl.ClusterName,
			ClusterType:               clctrl.ClusterType,
			DomainName:                clctrl.DomainName,
			SubdomainName:             clctrl.SubdomainName,
			KubefirstStateStoreBucket: clctrl.KubefirstStateStoreBucketName,
			KubefirstTeam:             clctrl.KubefirstTeam,
			NodeType:                  clctrl.NodeType,
			NodeCount:                 clctrl.NodeCount,
			KubefirstVersion:          env.KubefirstVersion,
			Kubeconfig:                clctrl.ProviderConfig.Kubeconfig, // AWS
			KubeconfigPath:            clctrl.ProviderConfig.Kubeconfig, // Not AWS

			ArgoCDIngressURL:               fmt.Sprintf("https://argocd.%s", fullDomainName),
			ArgoCDIngressNoHTTPSURL:        fmt.Sprintf("argocd.%s", fullDomainName),
			ArgoWorkflowsIngressURL:        fmt.Sprintf("https://argo.%s", fullDomainName),
			ArgoWorkflowsIngressNoHTTPSURL: fmt.Sprintf("argo.%s", fullDomainName),
			AtlantisIngressURL:             fmt.Sprintf("https://atlantis.%s", fullDomainName),
			AtlantisIngressNoHTTPSURL:      fmt.Sprintf("atlantis.%s", fullDomainName),
			ChartMuseumIngressURL:          fmt.Sprintf("https://chartmuseum.%s", fullDomainName),
			VaultIngressURL:                fmt.Sprintf("https://vault.%s", fullDomainName),
			VaultIngressNoHTTPSURL:         fmt.Sprintf("vault.%s", fullDomainName),
			VouchIngressURL:                fmt.Sprintf("https://vouch.%s", fullDomainName),

			GitDescription:       fmt.Sprintf("%s hosted git", clctrl.GitProvider),
			GitNamespace:         "N/A",
			GitProvider:          clctrl.GitProvider,
			GitRunner:            fmt.Sprintf("%s Runner", clctrl.GitProvider),
			GitRunnerDescription: fmt.Sprintf("Self Hosted %s Runner", clctrl.GitProvider),
			GitRunnerNS:          fmt.Sprintf("%s-runner", clctrl.GitProvider),
			GitURL:               clctrl.GitopsTemplateURL,
			GitopsRepoURL:        destinationGitopsRepoURL,

			GitHubHost:  fmt.Sprintf("https://github.com/%s/gitops.git", clctrl.GitAuth.Owner),
			GitHubOwner: clctrl.GitAuth.Owner,
			GitHubUser:  clctrl.GitAuth.User,

			GitlabHost:         clctrl.GitHost,
			GitlabOwner:        clctrl.GitAuth.Owner,
			GitlabOwnerGroupID: clctrl.GitlabOwnerGroupID,
			GitlabUser:         clctrl.GitAuth.User,

			GitopsRepoAtlantisWebhookURL:               clctrl.AtlantisWebhookURL,
			GitopsRepoNoHTTPSURL:                       fmt.Sprintf("%s/%s/gitops.git", clctrl.GitHost, clctrl.GitAuth.Owner),
			WorkloadClusterTerraformModuleURL:          fmt.Sprintf("git::https://%s/%s/gitops.git//terraform/%s/modules/workload-cluster?ref=main", clctrl.GitHost, clctrl.GitAuth.Owner, clctrl.CloudProvider),
			WorkloadClusterBootstrapTerraformModuleURL: fmt.Sprintf("git::https://%s/%s/gitops.git//terraform/%s/modules/bootstrap?ref=main", clctrl.GitHost, clctrl.GitAuth.Owner, clctrl.CloudProvider),
			ClusterID: clctrl.ClusterID,

			// external-dns optionality to provide cloudflare support regardless of cloud provider
			ExternalDNSProviderName:         clctrl.DNSProvider,
			ExternalDNSProviderTokenEnvName: externalDNSProviderTokenEnvName,
			ExternalDNSProviderSecretName:   fmt.Sprintf("%s-auth", clctrl.CloudProvider),
			ExternalDNSProviderSecretKey:    externalDNSProviderSecretKey,

			ContainerRegistryURL: fmt.Sprintf("%s/%s", clctrl.ContainerRegistryHost, clctrl.GitAuth.Owner),
		}

		// Handle provider specific tokens
		switch clctrl.CloudProvider {
		case "vultr":
			gitopsTemplateTokens.StateStoreBucketHostname = cl.StateStoreDetails.Hostname
		case "google":
			gitopsTemplateTokens.GoogleAuth = clctrl.GoogleAuth.KeyFile
			gitopsTemplateTokens.GoogleProject = clctrl.GoogleAuth.ProjectID
			gitopsTemplateTokens.GoogleUniqueness = strings.ToLower(randstr.String(5))
			gitopsTemplateTokens.ForceDestroy = strconv.FormatBool(true) // TODO make this optional
			gitopsTemplateTokens.KubefirstArtifactsBucket = clctrl.KubefirstStateStoreBucketName
			gitopsTemplateTokens.VaultDataBucketName = fmt.Sprintf("%s-vault-data-%s", clctrl.GoogleAuth.ProjectID, clctrl.ClusterName)
		case "aws":
			iamCaller, err := clctrl.AwsClient.GetCallerIdentity()
			if err != nil {
				return fmt.Errorf("error getting AWS caller identity while creating tokens: %w", err)
			}

			// to be added to general tokens struct
			gitopsTemplateTokens.AwsIamArnAccountRoot = fmt.Sprintf("arn:aws:iam::%s:root", *iamCaller.Account)
			gitopsTemplateTokens.AwsNodeCapacityType = "ON_DEMAND" // todo adopt cli flag
			gitopsTemplateTokens.AwsAccountID = *iamCaller.Account
			gitopsTemplateTokens.Kubeconfig = clctrl.ProviderConfig.Kubeconfig
			gitopsTemplateTokens.KubefirstArtifactsBucket = clctrl.KubefirstArtifactsBucketName
			gitopsTemplateTokens.AtlantisWebhookURL = clctrl.AtlantisWebhookURL

			if clctrl.ECR {
				gitopsTemplateTokens.ContainerRegistryURL = fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com", *iamCaller.Account, clctrl.CloudRegion)
				log.Info().Msgf("Using ECR URL %s", gitopsTemplateTokens.ContainerRegistryURL)
			} else {
				// moving commented line below to default behavior
				// gitopsTemplateTokens.ContainerRegistryURL = fmt.Sprintf("%s/%s", clctrl.ContainerRegistryHost, clctrl.GitAuth.Owner)
				log.Info().Msgf("NOT using ECR but instead %s URL %s", clctrl.GitProvider, gitopsTemplateTokens.ContainerRegistryURL)
			}
		case "azure":
			gitopsTemplateTokens.AzureStorageResourceGroup = "kubefirst-state" // @todo(sje): take from resourceGroup var in internal/controller/state.go
			gitopsTemplateTokens.AzureStorageContainerName = "terraform"       // @todo(sje): take from containerName var in internal/controller/state.go
		case "k3s":
			gitopsTemplateTokens.K3sServersPrivateIps = clctrl.K3sAuth.K3sServersPrivateIps
			gitopsTemplateTokens.K3sServersPublicIps = clctrl.K3sAuth.K3sServersPublicIps
			gitopsTemplateTokens.SSHUser = clctrl.K3sAuth.K3sSSHUser
			gitopsTemplateTokens.K3sServersArgs = clctrl.K3sAuth.K3sServersArgs
		}

		return gitopsTemplateTokens
	case "metaphor": // repo name
		metaphorTemplateTokens := &providerConfigs.MetaphorTokenValues{
			ClusterName:                   clctrl.ClusterName,
			CloudRegion:                   clctrl.CloudRegion,
			ContainerRegistryURL:          fmt.Sprintf("%s/%s/metaphor", clctrl.ContainerRegistryHost, clctrl.GitAuth.Owner),
			DomainName:                    fullDomainName,
			MetaphorDevelopmentIngressURL: fmt.Sprintf("metaphor-development.%s", fullDomainName),
			MetaphorStagingIngressURL:     fmt.Sprintf("metaphor-staging.%s", fullDomainName),
			MetaphorProductionIngressURL:  fmt.Sprintf("metaphor-production.%s", fullDomainName),
		}
		return metaphorTemplateTokens
	}

	return nil
}

// ClusterSecretsBootstrap
func (clctrl *ClusterController) ClusterSecretsBootstrap() error {
	cl, err := secrets.GetCluster(clctrl.KubernetesClient, clctrl.ClusterName)
	if err != nil {
		return fmt.Errorf("failed to get cluster during secrets bootstrap: %w", err)
	}

	var kcfg *k8s.KubernetesClient

	switch clctrl.CloudProvider {
	case "aws":
		kcfg = awsext.CreateEKSKubeconfig(&clctrl.AwsClient.Config, clctrl.ClusterName)
	case "akamai", "azure", "civo", "digitalocean", "k3s", "vultr":
		kcfg, err = k8s.CreateKubeConfig(false, clctrl.ProviderConfig.Kubeconfig)
		if err != nil {
			return fmt.Errorf("failed to create Kubernetes config during secrets bootstrap: %w", err)
		}
	case "google":
		var err error
		kcfg, err = clctrl.GoogleClient.GetContainerClusterAuth(clctrl.ClusterName, []byte(clctrl.GoogleAuth.KeyFile))
		if err != nil {
			return fmt.Errorf("error getting Google container cluster auth during secrets bootstrap: %w", err)
		}
	}
	clientSet := kcfg.Clientset

	// create namespaces
	err = providerConfigs.Namespaces(clientSet)
	if err != nil {
		return fmt.Errorf("failed to create namespaces during secrets bootstrap: %w", err)
	}

	destinationGitopsRepoGitURL, err := clctrl.GetRepoURL()
	if err != nil {
		return fmt.Errorf("failed to get repo URL for gitops during secrets bootstrap: %w", err)
	}

	// TODO Remove specific ext bootstrap functions.
	if !cl.ClusterSecretsCreatedCheck {
		switch clctrl.CloudProvider {
		case "akamai":
			err := akamaiext.BootstrapAkamaiMgmtCluster(clientSet, cl, destinationGitopsRepoGitURL)
			if err != nil {
				log.Error().Msgf("error adding Kubernetes secrets for bootstrap: %s", err)
				return fmt.Errorf("error adding Kubernetes secrets for bootstrap on akamai: %w", err)
			}
		case "aws":
			err := awsext.BootstrapAWSMgmtCluster(
				clientSet,
				cl,
				destinationGitopsRepoGitURL,
				clctrl.AwsClient,
			)
			if err != nil {
				log.Error().Msgf("error adding Kubernetes secrets for bootstrap: %s", err)
				return fmt.Errorf("error adding Kubernetes secrets for bootstrap on aws: %w", err)
			}
		case "azure":
			err := azureext.BootstrapAzureMgmtCluster(clientSet, cl, destinationGitopsRepoGitURL)
			if err != nil {
				log.Error().Msgf("error adding Kubernetes secrets for bootstrap: %s", err)
				return fmt.Errorf("error adding Kubernetes secrets for bootstrap on azure: %w", err)
			}
		case "civo":
			err := civoext.BootstrapCivoMgmtCluster(clientSet, cl, destinationGitopsRepoGitURL)
			if err != nil {
				log.Error().Msgf("error adding Kubernetes secrets for bootstrap: %s", err)
				return fmt.Errorf("error adding Kubernetes secrets for bootstrap on civo: %w", err)
			}
		case "google":
			err := googleext.BootstrapGoogleMgmtCluster(clientSet, cl, destinationGitopsRepoGitURL)
			if err != nil {
				log.Error().Msgf("error adding Kubernetes secrets for bootstrap: %s", err)
				return fmt.Errorf("error adding Kubernetes secrets for bootstrap on google: %w", err)
			}
		case "digitalocean":
			err := digitaloceanext.BootstrapDigitaloceanMgmtCluster(clientSet, cl, destinationGitopsRepoGitURL)
			if err != nil {
				log.Error().Msgf("error adding Kubernetes secrets for bootstrap: %s", err)
				return fmt.Errorf("error adding Kubernetes secrets for bootstrap on digitalocean: %w", err)
			}
		case "vultr":
			err := vultrext.BootstrapVultrMgmtCluster(clientSet, cl, destinationGitopsRepoGitURL)
			if err != nil {
				log.Error().Msgf("error adding Kubernetes secrets for bootstrap: %s", err)
				return fmt.Errorf("error adding Kubernetes secrets for bootstrap on vultr: %w", err)
			}
		case "k3s":
			err := k3sext.BootstrapK3sMgmtCluster(clientSet, cl, destinationGitopsRepoGitURL)
			if err != nil {
				log.Error().Msgf("error adding Kubernetes secrets for bootstrap: %s", err)
				return fmt.Errorf("error adding Kubernetes secrets for bootstrap on k3s: %w", err)
			}
		}

		err = providerConfigs.ServiceAccounts(clientSet)
		if err != nil {
			return fmt.Errorf("failed to create service accounts during secrets bootstrap: %w", err)
		}

		clctrl.Cluster.ClusterSecretsCreatedCheck = true
		err = secrets.UpdateCluster(clctrl.KubernetesClient, clctrl.Cluster)
		if err != nil {
			return fmt.Errorf("failed to update cluster state after creating secrets bootstrap: %w", err)
		}
	}

	return nil
}

// ContainerRegistryAuth
func (clctrl *ClusterController) ContainerRegistryAuth() (string, error) {
	var kcfg *k8s.KubernetesClient

	switch clctrl.CloudProvider {
	case "aws":
		kcfg = awsext.CreateEKSKubeconfig(&clctrl.AwsClient.Config, clctrl.ClusterName)

		// Container registry authentication creation
		containerRegistryAuth := gitShim.ContainerRegistryAuth{
			GitProvider:           clctrl.GitProvider,
			GitUser:               clctrl.GitAuth.User,
			GitToken:              clctrl.GitAuth.Token,
			GitlabGroupFlag:       clctrl.GitAuth.Owner,
			GithubOwner:           clctrl.GitAuth.Owner,
			ContainerRegistryHost: clctrl.ContainerRegistryHost,
			Clientset:             kcfg.Clientset,
		}
		containerRegistryAuthToken, err := gitShim.CreateContainerRegistrySecret(&containerRegistryAuth)
		if err != nil {
			log.Error().Msgf("error generating container registry authentication: %s", err)
			return "", fmt.Errorf("error generating container registry authentication for AWS: %w", err)
		}

		return containerRegistryAuthToken, nil
	case "civo", "digitalocean", "vultr", "k3s":
		var err error
		kcfg, err = k8s.CreateKubeConfig(false, clctrl.ProviderConfig.Kubeconfig)
		if err != nil {
			return "", fmt.Errorf("error creating Kubernetes config during registry auth: %w", err)
		}
	case "google":
		var err error
		kcfg, err = clctrl.GoogleClient.GetContainerClusterAuth(clctrl.ClusterName, []byte(clctrl.GoogleAuth.KeyFile))
		if err != nil {
			return "", fmt.Errorf("error getting google container cluster auth during registry auth: %w", err)
		}
	}

	// Container registry authentication creation
	containerRegistryAuth := gitShim.ContainerRegistryAuth{
		GitProvider:           clctrl.GitProvider,
		GitUser:               clctrl.GitAuth.User,
		GitToken:              clctrl.GitAuth.Token,
		GitlabGroupFlag:       clctrl.GitAuth.Owner,
		GithubOwner:           clctrl.GitAuth.Owner,
		ContainerRegistryHost: clctrl.ContainerRegistryHost,
		Clientset:             kcfg.Clientset,
	}
	containerRegistryAuthToken, err := gitShim.CreateContainerRegistrySecret(&containerRegistryAuth)
	if err != nil {
		log.Error().Msgf("error generating container registry authentication: %s", err)
		return "", fmt.Errorf("error generating container registry authentication for cloud provider %s: %w", clctrl.CloudProvider, err)
	}

	return containerRegistryAuthToken, nil
}

// WaitForClusterReady
func (clctrl *ClusterController) WaitForClusterReady() error {
	var kcfg *k8s.KubernetesClient

	switch clctrl.CloudProvider {
	case "aws":
		kcfg = awsext.CreateEKSKubeconfig(&clctrl.AwsClient.Config, clctrl.ClusterName)
	case "civo", "digitalocean", "vultr", "k3s":
		var err error
		kcfg, err = k8s.CreateKubeConfig(false, clctrl.ProviderConfig.Kubeconfig)
		if err != nil {
			return fmt.Errorf("error creating Kubernetes config while waiting for cluster ready: %w", err)
		}
	case "google":
		var err error
		kcfg, err = clctrl.GoogleClient.GetContainerClusterAuth(clctrl.ClusterName, []byte(clctrl.GoogleAuth.KeyFile))
		if err != nil {
			return fmt.Errorf("error getting google container cluster auth while waiting for cluster ready: %w", err)
		}
	}

	var dnsDeployment *v1.Deployment
	var err error
	switch clctrl.CloudProvider {
	case "aws", "civo", "digitalocean", "vultr", "k3s":
		dnsDeployment, err = k8s.ReturnDeploymentObject(
			kcfg.Clientset,
			"kubernetes.io/name",
			"CoreDNS",
			"kube-system",
			300,
		)
		if err != nil {
			log.Error().Msgf("error finding CoreDNS deployment: %s", err)
			return fmt.Errorf("error finding CoreDNS deployment while waiting for cluster to be ready: %w", err)
		}
	case "google":
		dnsDeployment, err = k8s.ReturnDeploymentObject(
			kcfg.Clientset,
			"k8s-app",
			"kube-dns",
			"kube-system",
			300,
		)
		if err != nil {
			log.Error().Msgf("error finding CoreDNS deployment: %s", err)
			return fmt.Errorf("error finding CoreDNS deployment while waiting for cluster to be ready: %w", err)
		}
	}

	_, err = k8s.WaitForDeploymentReady(kcfg.Clientset, dnsDeployment, 120)
	if err != nil {
		log.Error().Msgf("error waiting for CoreDNS deployment ready state: %s", err)
		return fmt.Errorf("error waiting for CoreDNS deployment ready state while waiting for cluster to be ready: %w", err)
	}

	return nil
}
