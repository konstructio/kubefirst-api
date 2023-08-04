/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/imdario/mergo"
	awsext "github.com/kubefirst/kubefirst-api/extensions/aws"
	civoext "github.com/kubefirst/kubefirst-api/extensions/civo"
	digitaloceanext "github.com/kubefirst/kubefirst-api/extensions/digitalocean"
	terraformext "github.com/kubefirst/kubefirst-api/extensions/terraform"
	vultrext "github.com/kubefirst/kubefirst-api/extensions/vultr"
	gitShim "github.com/kubefirst/kubefirst-api/internal/gitShim"
	"github.com/kubefirst/kubefirst-api/internal/telemetryShim"
	"github.com/kubefirst/runtime/pkg/gitlab"
	"github.com/kubefirst/runtime/pkg/k8s"
	"github.com/kubefirst/runtime/pkg/providerConfigs"
	"github.com/kubefirst/runtime/pkg/segment"
	log "github.com/sirupsen/logrus"
)

// Global Controller Variables
// AWS
// gitlab may have subgroups, so the destination gitops/metaphor repo git urls may be different
var AWSDestinationGitopsRepoURL, AWSDestinationMetaphorRepoURL string

// Civo
// gitlab may have subgroups, so the destination gitops/metaphor repo git urls may be different
var CivoDestinationGitopsRepoURL, CivoDestinationMetaphorRepoURL string

// Digital Ocean
// gitlab may have subgroups, so the destination gitops/metaphor repo git urls may be different
var DigitaloceanDestinationGitopsRepoURL, DigitaloceanDestinationMetaphorRepoURL string

// Vultr
// gitlab may have subgroups, so the destination gitops/metaphor repo git urls may be different
var VultrDestinationGitopsRepoURL, VultrDestinationMetaphorRepoURL string

// CreateCluster
func (clctrl *ClusterController) CreateCluster() error {
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

	if !cl.CloudTerraformApplyCheck || cl.CloudTerraformApplyFailedCheck {
		telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricCloudTerraformApplyStarted, "")

		log.Infof("creating %s cluster", clctrl.CloudProvider)

		switch clctrl.CloudProvider {
		case "aws":
			telemetryShim.Transmit(true, segmentClient, segment.MetricCloudTerraformApplyStarted, "")

			log.Info("creating aws cloud resources with terraform")

			tfEntrypoint := clctrl.ProviderConfig.GitopsDir + "/terraform/aws"
			tfEnvs := map[string]string{}
			tfEnvs = awsext.GetAwsTerraformEnvs(tfEnvs, &cl)
			iamCaller, err := clctrl.AwsClient.GetCallerIdentity()
			if err != nil {
				return err
			}
			tfEnvs["TF_VAR_aws_account_id"] = *iamCaller.Account
			tfEnvs["TF_VAR_use_ecr"] = strconv.FormatBool(clctrl.ECR) //Flag out the ecr terraform

			err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "aws_account_id", *iamCaller.Account)
			if err != nil {
				return err
			}

			err = terraformext.InitApplyAutoApprove(clctrl.ProviderConfig.TerraformClient, tfEntrypoint, tfEnvs)
			if err != nil {
				msg := fmt.Sprintf("error creating aws resources with terraform %s: %s", tfEntrypoint, err)
				log.Error(msg)
				err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "cloud_terraform_apply_failed_check", true)
				if err != nil {
					return err
				}
				telemetryShim.Transmit(true, segmentClient, segment.MetricCloudTerraformApplyFailed, msg)
				return fmt.Errorf(msg)
			}

			log.Info("created aws cloud resources")

			telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricCloudTerraformApplyCompleted, "")
		case "civo":
			telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricCloudTerraformApplyStarted, "")

			log.Info("creating civo cloud resources with terraform")

			tfEntrypoint := clctrl.ProviderConfig.GitopsDir + "/terraform/civo"
			tfEnvs := map[string]string{}
			tfEnvs = civoext.GetCivoTerraformEnvs(tfEnvs, &cl)
			err := terraformext.InitApplyAutoApprove(clctrl.ProviderConfig.TerraformClient, tfEntrypoint, tfEnvs)
			if err != nil {
				msg := fmt.Sprintf("error creating civo resources with terraform %s: %s", tfEntrypoint, err)
				log.Error(msg)
				err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "cloud_terraform_apply_failed_check", true)
				if err != nil {
					return err
				}
				telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricCloudTerraformApplyFailed, msg)
				return fmt.Errorf(msg)
			}

			log.Info("created civo cloud resources")

			telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricCloudTerraformApplyCompleted, "")
		case "digitalocean":
			telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricCloudTerraformApplyStarted, "")

			log.Info("creating digital ocean cloud resources with terraform")

			tfEntrypoint := clctrl.ProviderConfig.GitopsDir + "/terraform/digitalocean"
			tfEnvs := map[string]string{}
			tfEnvs = digitaloceanext.GetDigitaloceanTerraformEnvs(tfEnvs, &cl)
			err := terraformext.InitApplyAutoApprove(clctrl.ProviderConfig.TerraformClient, tfEntrypoint, tfEnvs)
			if err != nil {
				msg := fmt.Sprintf("error creating digital ocean resources with terraform %s: %s", tfEntrypoint, err)
				log.Error(msg)
				err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "cloud_terraform_apply_failed_check", true)
				if err != nil {
					return err
				}
				telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricCloudTerraformApplyFailed, msg)
				return fmt.Errorf(msg)
			}

			log.Info("created digital ocean cloud resources")

			telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricCloudTerraformApplyCompleted, "")
		case "vultr":
			telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricCloudTerraformApplyStarted, "")

			log.Info("creating vultr cloud resources with terraform")

			tfEntrypoint := clctrl.ProviderConfig.GitopsDir + "/terraform/vultr"
			tfEnvs := map[string]string{}
			tfEnvs = vultrext.GetVultrTerraformEnvs(tfEnvs, &cl)
			err := terraformext.InitApplyAutoApprove(clctrl.ProviderConfig.TerraformClient, tfEntrypoint, tfEnvs)
			if err != nil {
				msg := fmt.Sprintf("error creating vultr resources with terraform %s: %s", tfEntrypoint, err)
				log.Error(msg)
				err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "cloud_terraform_apply_failed_check", true)
				if err != nil {
					return err
				}
				telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricCloudTerraformApplyFailed, msg)
				return fmt.Errorf(msg)
			}

			log.Info("created vultr cloud resources")

			telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricCloudTerraformApplyCompleted, "")
		}

		log.Infof("successfully created %s cluster", clctrl.CloudProvider)

		telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricCloudTerraformApplyCompleted, "")

		err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "cloud_terraform_apply_failed_check", false)
		if err != nil {
			return err
		}

		err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "cloud_terraform_apply_check", true)
		if err != nil {
			return err
		}
	}

	return nil
}

// CreateTokens
func (clctrl *ClusterController) CreateTokens(kind string) interface{} {
	kubefirstVersion := os.Getenv("KUBEFIRST_VERSION")
	if kubefirstVersion == "" {
		kubefirstVersion = "development"
	}

	// Handle repo urls for gitops and metaphor
	switch clctrl.GitProvider {
	case "github":
		AWSDestinationGitopsRepoURL = clctrl.ProviderConfig.DestinationGitopsRepoURL
		AWSDestinationMetaphorRepoURL = clctrl.ProviderConfig.DestinationMetaphorRepoURL
		CivoDestinationGitopsRepoURL = clctrl.ProviderConfig.DestinationGitopsRepoURL
		CivoDestinationMetaphorRepoURL = clctrl.ProviderConfig.DestinationMetaphorRepoURL
		DigitaloceanDestinationGitopsRepoURL = clctrl.ProviderConfig.DestinationGitopsRepoURL
		DigitaloceanDestinationMetaphorRepoURL = clctrl.ProviderConfig.DestinationMetaphorRepoURL
		VultrDestinationGitopsRepoURL = clctrl.ProviderConfig.DestinationGitopsRepoURL
		VultrDestinationMetaphorRepoURL = clctrl.ProviderConfig.DestinationMetaphorRepoURL
	case "gitlab":
		gitlabClient, err := gitlab.NewGitLabClient(clctrl.GitToken, clctrl.GitOwner)
		if err != nil {
			return err
		}
		// Format git url based on full path to group
		switch clctrl.ProviderConfig.GitProtocol {
		case "https":
			clctrl.ProviderConfig.DestinationGitopsRepoHttpsURL = fmt.Sprintf("https://gitlab.com/%s/gitops.git", gitlabClient.ParentGroupPath)
			clctrl.ProviderConfig.DestinationMetaphorRepoHttpsURL = fmt.Sprintf("https://gitlab.com/%s/metaphor.git", gitlabClient.ParentGroupPath)
			AWSDestinationGitopsRepoURL = clctrl.ProviderConfig.DestinationGitopsRepoHttpsURL
			AWSDestinationMetaphorRepoURL = clctrl.ProviderConfig.DestinationMetaphorRepoHttpsURL
			CivoDestinationGitopsRepoURL = clctrl.ProviderConfig.DestinationGitopsRepoHttpsURL
			CivoDestinationMetaphorRepoURL = clctrl.ProviderConfig.DestinationMetaphorRepoHttpsURL
			DigitaloceanDestinationGitopsRepoURL = clctrl.ProviderConfig.DestinationGitopsRepoHttpsURL
			DigitaloceanDestinationMetaphorRepoURL = clctrl.ProviderConfig.DestinationMetaphorRepoHttpsURL
			VultrDestinationGitopsRepoURL = clctrl.ProviderConfig.DestinationGitopsRepoHttpsURL
			VultrDestinationMetaphorRepoURL = clctrl.ProviderConfig.DestinationMetaphorRepoHttpsURL
		default:
			clctrl.ProviderConfig.DestinationGitopsRepoGitURL = fmt.Sprintf("git@gitlab.com:%s/gitops.git", gitlabClient.ParentGroupPath)
			clctrl.ProviderConfig.DestinationMetaphorRepoGitURL = fmt.Sprintf("git@gitlab.com:%s/metaphor.git", gitlabClient.ParentGroupPath)
			AWSDestinationGitopsRepoURL = clctrl.ProviderConfig.DestinationGitopsRepoGitURL
			AWSDestinationMetaphorRepoURL = clctrl.ProviderConfig.DestinationMetaphorRepoGitURL
			CivoDestinationGitopsRepoURL = clctrl.ProviderConfig.DestinationGitopsRepoGitURL
			CivoDestinationMetaphorRepoURL = clctrl.ProviderConfig.DestinationMetaphorRepoGitURL
			DigitaloceanDestinationGitopsRepoURL = clctrl.ProviderConfig.DestinationGitopsRepoGitURL
			DigitaloceanDestinationMetaphorRepoURL = clctrl.ProviderConfig.DestinationMetaphorRepoGitURL
			VultrDestinationGitopsRepoURL = clctrl.ProviderConfig.DestinationGitopsRepoGitURL
			VultrDestinationMetaphorRepoURL = clctrl.ProviderConfig.DestinationMetaphorRepoHttpsURL
		}
	}

	//handle set gitops tokens/values
	switch kind {
	case "gitops": //repo name

		var externalDNSProviderTokenEnvName, externalDNSProviderSecretKey string
		if clctrl.DnsProvider == "cloudflare" {
			externalDNSProviderTokenEnvName = "CF_API_TOKEN"
			externalDNSProviderSecretKey = "cf-api-token"
		} else {
			externalDNSProviderTokenEnvName = fmt.Sprintf("%s_TOKEN", strings.ToUpper(clctrl.CloudProvider))
			externalDNSProviderSecretKey = fmt.Sprintf("%s-token", clctrl.CloudProvider)
		}

		// Default gitopsTemplateTokens
		gitopsTemplateTokens := &providerConfigs.GitopsDirectoryValues{
			AlertsEmail:               clctrl.AlertsEmail,
			AtlantisAllowList:         fmt.Sprintf("%s/%s/*", clctrl.GitHost, clctrl.GitOwner),
			CloudProvider:             clctrl.CloudProvider,
			CloudRegion:               clctrl.CloudRegion,
			ClusterName:               clctrl.ClusterName,
			ClusterType:               clctrl.ClusterType,
			DomainName:                clctrl.DomainName,
			KubefirstStateStoreBucket: clctrl.KubefirstStateStoreBucketName,
			KubefirstTeam:             clctrl.KubefirstTeam,
			KubefirstVersion:          kubefirstVersion,
			Kubeconfig:                clctrl.ProviderConfig.Kubeconfig, //AWS
			KubeconfigPath:            clctrl.ProviderConfig.Kubeconfig, //Not AWS

			ArgoCDIngressURL:               fmt.Sprintf("https://argocd.%s", clctrl.DomainName),
			ArgoCDIngressNoHTTPSURL:        fmt.Sprintf("argocd.%s", clctrl.DomainName),
			ArgoWorkflowsIngressURL:        fmt.Sprintf("https://argo.%s", clctrl.DomainName),
			ArgoWorkflowsIngressNoHTTPSURL: fmt.Sprintf("argo.%s", clctrl.DomainName),
			AtlantisIngressURL:             fmt.Sprintf("https://atlantis.%s", clctrl.DomainName),
			AtlantisIngressNoHTTPSURL:      fmt.Sprintf("atlantis.%s", clctrl.DomainName),
			ChartMuseumIngressURL:          fmt.Sprintf("https://chartmuseum.%s", clctrl.DomainName),
			VaultIngressURL:                fmt.Sprintf("https://vault.%s", clctrl.DomainName),
			VaultIngressNoHTTPSURL:         fmt.Sprintf("vault.%s", clctrl.DomainName),
			VouchIngressURL:                fmt.Sprintf("https://vouch.%s", clctrl.DomainName),

			GitDescription:       fmt.Sprintf("%s hosted git", clctrl.GitProvider),
			GitNamespace:         "N/A",
			GitProvider:          clctrl.GitProvider,
			GitRunner:            fmt.Sprintf("%s Runner", clctrl.GitProvider),
			GitRunnerDescription: fmt.Sprintf("Self Hosted %s Runner", clctrl.GitProvider),
			GitRunnerNS:          fmt.Sprintf("%s-runner", clctrl.GitProvider),
			GitURL:               clctrl.GitopsTemplateURL,

			GitHubHost:  fmt.Sprintf("https://github.com/%s/gitops.git", clctrl.GitOwner),
			GitHubOwner: clctrl.GitOwner,
			GitHubUser:  clctrl.GitUser,

			GitlabHost:         clctrl.GitHost,
			GitlabOwner:        clctrl.GitOwner,
			GitlabOwnerGroupID: clctrl.GitlabOwnerGroupID,
			GitlabUser:         clctrl.GitUser,

			GitopsRepoAtlantisWebhookURL: clctrl.AtlantisWebhookURL,
			GitopsRepoNoHTTPSURL:         fmt.Sprintf("%s.com/%s/gitops.git", clctrl.GitHost, clctrl.GitOwner),
			ClusterId:                    clctrl.ClusterID,

			// external-dns optionality to provide cloudflare support regardless of cloud provider
			ExternalDNSProviderName:         clctrl.DnsProvider,
			ExternalDNSProviderTokenEnvName: externalDNSProviderTokenEnvName,
			ExternalDNSProviderSecretName:   fmt.Sprintf("%s-creds", clctrl.CloudProvider),
			ExternalDNSProviderSecretKey:    externalDNSProviderSecretKey,
		}

		//Handle provider specific tokens
		switch clctrl.CloudProvider {
		case "aws":
			iamCaller, err := clctrl.AwsClient.GetCallerIdentity()
			if err != nil {
				return err
			}

			//to be added to general tokens struct
			awsAdditionalTokens := &providerConfigs.GitopsDirectoryValues{
				AwsIamArnAccountRoot: fmt.Sprintf("arn:aws:iam::%s:root", *iamCaller.Account),
				AwsNodeCapacityType:  "ON_DEMAND", // todo adopt cli flag
				AwsAccountID:         *iamCaller.Account,

				Kubeconfig:               clctrl.ProviderConfig.Kubeconfig,
				KubefirstArtifactsBucket: clctrl.KubefirstArtifactsBucketName,

				AtlantisWebhookURL: clctrl.AtlantisWebhookURL,
			}
			// Merge aws additional tokens and gitopsTemplateTokens
			mergo.Merge(&gitopsTemplateTokens, awsAdditionalTokens)

			if clctrl.ECR {
				gitopsTemplateTokens.ContainerRegistryURL = fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com", *iamCaller.Account, clctrl.CloudRegion)
				log.Info("Using ECR URL %s", gitopsTemplateTokens.ContainerRegistryURL)
			} else {
				log.Info("Not using ECR but instead %s", clctrl.GitProvider)
			}

			gitopsTemplateTokens.GitopsRepoURL = AWSDestinationGitopsRepoURL
		case "civo":
			gitopsTemplateTokens.GitopsRepoURL = CivoDestinationGitopsRepoURL
		case "digitalocean":
			gitopsTemplateTokens.GitopsRepoURL = DigitaloceanDestinationGitopsRepoURL
			gitopsTemplateTokens.StateStoreBucketHostname = DigitaloceanStateStoreBucketName
		case "vultr":
			gitopsTemplateTokens.GitopsRepoURL = VultrDestinationGitopsRepoURL
			gitopsTemplateTokens.StateStoreBucketHostname = VultrStateStoreBucketHostname
		}

		return gitopsTemplateTokens
	case "metaphor": //repo name
		metaphorTemplateTokens := &providerConfigs.MetaphorTokenValues{
			ClusterName:                   clctrl.ClusterName,
			CloudRegion:                   clctrl.CloudRegion,
			ContainerRegistryURL:          fmt.Sprintf("%s/%s/metaphor", clctrl.ContainerRegistryHost, clctrl.GitOwner),
			DomainName:                    clctrl.DomainName,
			MetaphorDevelopmentIngressURL: fmt.Sprintf("metaphor-development.%s", clctrl.DomainName),
			MetaphorStagingIngressURL:     fmt.Sprintf("metaphor-staging.%s", clctrl.DomainName),
			MetaphorProductionIngressURL:  fmt.Sprintf("metaphor-production.%s", clctrl.DomainName),
		}
		return metaphorTemplateTokens
	}

	return nil
}

// ClusterSecretsBootstrap
func (clctrl *ClusterController) ClusterSecretsBootstrap() error {
	cl, err := clctrl.MdbCl.GetCluster(clctrl.ClusterName)
	if err != nil {
		return err
	}
	clientSet, err := k8s.GetClientSet(clctrl.ProviderConfig.Kubeconfig)
	if err != nil {
		return err
	}
	//create namespaces
	err = providerConfigs.K8sNamespaces(clientSet)
	if err != nil {
		return err
	}

	if !cl.ClusterSecretsCreatedCheck {
		switch clctrl.CloudProvider {
		case "aws":
			err := awsext.BootstrapAWSMgmtCluster(
				clientSet,
				&cl, &clctrl.ProviderConfig,
				clctrl.AwsClient,
				clctrl.ProviderConfig.GitopsDirectoryValues.ContainerRegistryURL,
			)
			if err != nil {
				log.Errorf("error adding kubernetes secrets for bootstrap: %s", err)
				return err
			}
		case "civo":
			err := civoext.BootstrapCivoMgmtCluster(clientSet, &cl, &clctrl.ProviderConfig)
			if err != nil {
				log.Errorf("error adding kubernetes secrets for bootstrap: %s", err)
				return err
			}
		case "digitalocean":
			err := digitaloceanext.BootstrapDigitaloceanMgmtCluster(clientSet, &cl, &clctrl.ProviderConfig)
			if err != nil {
				log.Errorf("error adding kubernetes secrets for bootstrap: %s", err)
				return err
			}
		case "vultr":
			err := vultrext.BootstrapVultrMgmtCluster(clientSet, &cl, &clctrl.ProviderConfig)
			if err != nil {
				log.Errorf("error adding kubernetes secrets for bootstrap: %s", err)
				return err
			}
		}

		//create service accounts
		err = providerConfigs.ServiceAccounts(clientSet)
		if err != nil {
			return err
		}

		err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "cluster_secrets_created_check", true)
		if err != nil {
			return err
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
	case "civo", "digitalocean", "vultr":
		kcfg = k8s.CreateKubeConfig(false, clctrl.ProviderConfig.Kubeconfig)
	}

	// Container registry authentication creation
	containerRegistryAuth := gitShim.ContainerRegistryAuth{
		GitProvider:           clctrl.GitProvider,
		GitUser:               clctrl.GitUser,
		GitToken:              clctrl.GitToken,
		GitlabGroupFlag:       clctrl.GitOwner,
		GithubOwner:           clctrl.GitOwner,
		ContainerRegistryHost: clctrl.ContainerRegistryHost,
		Clientset:             kcfg.Clientset,
	}
	containerRegistryAuthToken, err := gitShim.CreateContainerRegistrySecret(&containerRegistryAuth)
	if err != nil {
		log.Errorf("error generating container registry authentication: %s", err)
		return "", err
	}

	return containerRegistryAuthToken, nil
}

// WaitForClusterReady
func (clctrl *ClusterController) WaitForClusterReady() error {
	// Logging handler
	// Logs to stdout to maintain compatibility with event streaming
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "",
	})
	log.SetReportCaller(false)
	log.SetOutput(os.Stdout)

	var kcfg *k8s.KubernetesClient

	switch clctrl.CloudProvider {
	case "aws":
		kcfg = awsext.CreateEKSKubeconfig(&clctrl.AwsClient.Config, clctrl.ClusterName)
	case "civo", "digitalocean", "vultr":
		kcfg = k8s.CreateKubeConfig(false, clctrl.ProviderConfig.Kubeconfig)
	}

	coreDNSDeployment, err := k8s.ReturnDeploymentObject(
		kcfg.Clientset,
		"kubernetes.io/name",
		"CoreDNS",
		"kube-system",
		120,
	)
	if err != nil {
		log.Errorf("error finding CoreDNS deployment: %s", err)
		return err
	}
	_, err = k8s.WaitForDeploymentReady(kcfg.Clientset, coreDNSDeployment, 120)
	if err != nil {
		log.Errorf("error waiting for CoreDNS deployment ready state: %s", err)
		return err
	}

	return nil
}
