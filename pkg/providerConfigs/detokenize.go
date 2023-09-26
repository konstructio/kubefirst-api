/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package providerConfigs

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// DetokenizeGitGitops - Translate tokens by values on a given path
func DetokenizeGitGitops(path string, tokens *GitopsDirectoryValues, gitProtocol string) error {
	err := filepath.Walk(path, detokenizeGitops(path, tokens, gitProtocol))
	if err != nil {
		return err
	}

	return nil
}

func detokenizeGitops(path string, tokens *GitopsDirectoryValues, gitProtocol string) filepath.WalkFunc {
	return filepath.WalkFunc(func(path string, fi os.FileInfo, err error) error {

		if fi.IsDir() && fi.Name() == ".git" {
			return filepath.SkipDir
		}
		if err != nil {
			return err
		}

		if fi.IsDir() {
			return nil
		}

		metaphorDevelopmentIngressURL := fmt.Sprintf("https://metaphor-development.%s", tokens.DomainName)
		metaphorStagingIngressURL := fmt.Sprintf("https://metaphor-staging.%s", tokens.DomainName)
		metaphorProductionIngressURL := fmt.Sprintf("https://metaphor-production.%s", tokens.DomainName)

		// var matched bool
		matched, _ := filepath.Match("*", fi.Name())

		if matched {
			// ignore .git files
			if !strings.Contains(path, "/.git/") {

				read, err := ioutil.ReadFile(path)
				if err != nil {
					return err
				}

				var fullDomainName string

				if tokens.SubdomainName != "" {
					fullDomainName = fmt.Sprintf("%s.%s", tokens.SubdomainName, tokens.DomainName)
				} else {
					fullDomainName = tokens.DomainName
				}

				newContents := string(read)
				newContents = strings.Replace(newContents, "<ALERTS_EMAIL>", tokens.AlertsEmail, -1)
				newContents = strings.Replace(newContents, "<ATLANTIS_ALLOW_LIST>", tokens.AtlantisAllowList, -1)
				newContents = strings.Replace(newContents, "<CLUSTER_NAME>", tokens.ClusterName, -1)
				newContents = strings.Replace(newContents, "<CLOUD_PROVIDER>", tokens.CloudProvider, -1)
				newContents = strings.Replace(newContents, "<CLOUD_REGION>", tokens.CloudRegion, -1)
				newContents = strings.Replace(newContents, "<CLUSTER_NAME>", tokens.ClusterName, -1)
				newContents = strings.Replace(newContents, "<CLUSTER_ID>", tokens.ClusterId, -1)
				newContents = strings.Replace(newContents, "<CLUSTER_TYPE>", tokens.ClusterType, -1)
				newContents = strings.Replace(newContents, "<CONTAINER_REGISTRY_URL>", tokens.ContainerRegistryURL, -1)
				newContents = strings.Replace(newContents, "<DOMAIN_NAME>", fullDomainName, -1)
				newContents = strings.Replace(newContents, "<KUBE_CONFIG_PATH>", tokens.KubeconfigPath, -1)
				newContents = strings.Replace(newContents, "<KUBEFIRST_ARTIFACTS_BUCKET>", tokens.KubefirstArtifactsBucket, -1)
				newContents = strings.Replace(newContents, "<KUBEFIRST_STATE_STORE_BUCKET>", tokens.KubefirstStateStoreBucket, -1)
				newContents = strings.Replace(newContents, "<KUBEFIRST_TEAM>", tokens.KubefirstTeam, -1)
				newContents = strings.Replace(newContents, "<KUBEFIRST_VERSION>", tokens.KubefirstVersion, -1)
				newContents = strings.Replace(newContents, "<KUBEFIRST_STATE_STORE_BUCKET_HOSTNAME>", tokens.StateStoreBucketHostname, -1)

				// AWS
				newContents = strings.Replace(newContents, "<AWS_ACCOUNT_ID>", tokens.AwsAccountID, -1)
				newContents = strings.Replace(newContents, "<AWS_IAM_ARN_ACCOUNT_ROOT>", tokens.AwsIamArnAccountRoot, -1)
				newContents = strings.Replace(newContents, "<AWS_NODE_CAPACITY_TYPE>", tokens.AwsNodeCapacityType, -1)

				// google
				newContents = strings.Replace(newContents, "<GOOGLE_PROJECT>", tokens.GoogleProject, -1)
				newContents = strings.Replace(newContents, "<TERRAFORM_FORCE_DESTROY>", tokens.ForceDestroy, -1)
				newContents = strings.Replace(newContents, "<GOOGLE_UNIQUENESS>", tokens.GoogleUniqueness, -1)

				newContents = strings.Replace(newContents, "<ARGOCD_INGRESS_URL>", tokens.ArgoCDIngressURL, -1)
				newContents = strings.Replace(newContents, "<ARGOCD_INGRESS_NO_HTTP_URL>", tokens.ArgoCDIngressNoHTTPSURL, -1)
				newContents = strings.Replace(newContents, "<ARGO_WORKFLOWS_INGRESS_URL>", tokens.ArgoWorkflowsIngressURL, -1)
				newContents = strings.Replace(newContents, "<ARGO_WORKFLOWS_INGRESS_NO_HTTPS_URL>", tokens.ArgoWorkflowsIngressNoHTTPSURL, -1)
				newContents = strings.Replace(newContents, "<ATLANTIS_INGRESS_URL>", tokens.AtlantisIngressURL, -1)
				newContents = strings.Replace(newContents, "<ATLANTIS_INGRESS_NO_HTTPS_URL>", tokens.AtlantisIngressNoHTTPSURL, -1)
				newContents = strings.Replace(newContents, "<CHARTMUSEUM_INGRESS_URL>", tokens.ChartMuseumIngressURL, -1)
				newContents = strings.Replace(newContents, "<VAULT_INGRESS_URL>", tokens.VaultIngressURL, -1)
				newContents = strings.Replace(newContents, "<VAULT_INGRESS_NO_HTTPS_URL>", tokens.VaultIngressNoHTTPSURL, -1)
				newContents = strings.Replace(newContents, "<VAULT_DATA_BUCKET>", tokens.VaultDataBucketName, -1)
				newContents = strings.Replace(newContents, "<VOUCH_INGRESS_URL>", tokens.VouchIngressURL, -1)

				newContents = strings.Replace(newContents, "<GIT_DESCRIPTION>", tokens.GitDescription, -1)
				newContents = strings.Replace(newContents, "<GIT_NAMESPACE>", tokens.GitNamespace, -1)
				newContents = strings.Replace(newContents, "<GIT_PROVIDER>", tokens.GitProvider, -1)
				newContents = strings.Replace(newContents, "<GIT-PROTOCOL>", gitProtocol, -1)
				newContents = strings.Replace(newContents, "<GIT_RUNNER>", tokens.GitRunner, -1)
				newContents = strings.Replace(newContents, "<GIT_RUNNER_DESCRIPTION>", tokens.GitRunnerDescription, -1)
				newContents = strings.Replace(newContents, "<GIT_RUNNER_NS>", tokens.GitRunnerNS, -1)
				newContents = strings.Replace(newContents, "<GIT_URL>", tokens.GitURL, -1)

				// GitHub
				newContents = strings.Replace(newContents, "<GITHUB_HOST>", tokens.GitHubHost, -1)
				newContents = strings.Replace(newContents, "<GITHUB_OWNER>", strings.ToLower(tokens.GitHubOwner), -1)
				newContents = strings.Replace(newContents, "<GITHUB_USER>", tokens.GitHubUser, -1)

				// GitLab
				newContents = strings.Replace(newContents, "<GITLAB_HOST>", tokens.GitlabHost, -1)
				newContents = strings.Replace(newContents, "<GITLAB_OWNER>", tokens.GitlabOwner, -1)
				newContents = strings.Replace(newContents, "<GITLAB_OWNER_GROUP_ID>", strconv.Itoa(tokens.GitlabOwnerGroupID), -1)
				newContents = strings.Replace(newContents, "<GITLAB_USER>", tokens.GitlabUser, -1)

				newContents = strings.Replace(newContents, "<GITOPS_REPO_ATLANTIS_WEBHOOK_URL>", tokens.GitopsRepoAtlantisWebhookURL, -1)
				newContents = strings.Replace(newContents, "<GITOPS_REPO_GIT_URL>", tokens.GitopsRepoGitURL, -1)
				newContents = strings.Replace(newContents, "<GITOPS_REPO_NO_HTTPS_URL>", tokens.GitopsRepoNoHTTPSURL, -1)

				newContents = strings.Replace(newContents, "<METAPHOR_DEVELOPMENT_INGRESS_URL>", metaphorDevelopmentIngressURL, -1)
				newContents = strings.Replace(newContents, "<METAPHOR_PRODUCTION_INGRESS_URL>", metaphorProductionIngressURL, -1)
				newContents = strings.Replace(newContents, "<METAPHOR_STAGING_INGRESS_URL>", metaphorStagingIngressURL, -1)

				// external-dns optionality to provide cloudflare support regardless of cloud provider
				newContents = strings.Replace(newContents, "<EXTERNAL_DNS_PROVIDER_NAME>", tokens.ExternalDNSProviderName, -1)
				newContents = strings.Replace(newContents, "<EXTERNAL_DNS_PROVIDER_TOKEN_ENV_NAME>", tokens.ExternalDNSProviderTokenEnvName, -1)
				newContents = strings.Replace(newContents, "<EXTERNAL_DNS_PROVIDER_SECRET_NAME>", tokens.ExternalDNSProviderSecretName, -1)
				newContents = strings.Replace(newContents, "<EXTERNAL_DNS_PROVIDER_SECRET_KEY>", tokens.ExternalDNSProviderSecretKey, -1)
				newContents = strings.Replace(newContents, "<EXTERNAL_DNS_DOMAIN_NAME>", tokens.DomainName, -1)

				newContents = strings.Replace(newContents, "<USE_TELEMETRY>", tokens.UseTelemetry, -1)

				// Switch the repo url based on https flag
				newContents = strings.Replace(newContents, "<GITOPS_REPO_URL>", tokens.GitopsRepoURL, -1)

				//The fqdn is used by metaphor/argo to choose the appropriate url for cicd operations.
				if gitProtocol == "https" {
					newContents = strings.Replace(newContents, "<GIT_FQDN>", fmt.Sprintf("https://%v.com/", tokens.GitProvider), -1)
				} else {
					newContents = strings.Replace(newContents, "<GIT_FQDN>", fmt.Sprintf("git@%v.com:", tokens.GitProvider), -1)
				}

				err = ioutil.WriteFile(path, []byte(newContents), 0)
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// DetokenizeAdditionalPath - Translate tokens by values on a given path
func DetokenizeAdditionalPath(path string, tokens *GitopsDirectoryValues) error {
	err := filepath.Walk(path, detokenizeAdditionalPath(path, tokens))
	if err != nil {
		return err
	}

	return nil
}

// detokenizeAdditionalPath temporary addition to handle detokenizing additional files
func detokenizeAdditionalPath(path string, tokens *GitopsDirectoryValues) filepath.WalkFunc {
	return filepath.WalkFunc(func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !!fi.IsDir() {
			return nil
		}

		// var matched bool
		matched, _ := filepath.Match("*", fi.Name())

		if matched {
			// ignore .git files
			if !strings.Contains(path, "/.git/") {
				read, err := ioutil.ReadFile(path)
				if err != nil {
					return err
				}

				newContents := string(read)
				newContents = strings.Replace(newContents, "<GITLAB_OWNER>", tokens.GitlabOwner, -1)

				err = ioutil.WriteFile(path, []byte(newContents), 0)
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// DetokenizeGithubMetaphor - Translate tokens by values on a given path
func DetokenizeGitMetaphor(path string, tokens *MetaphorTokenValues) error {
	err := filepath.Walk(path, detokenizeGitopsMetaphor(path, tokens))
	if err != nil {
		return err
	}
	return nil
}

// DetokenizeDirectoryGithubMetaphor - Translate tokens by values on a directory level.
func detokenizeGitopsMetaphor(path string, tokens *MetaphorTokenValues) filepath.WalkFunc {
	return filepath.WalkFunc(func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !!fi.IsDir() {
			return nil
		}

		// var matched bool
		matched, _ := filepath.Match("*", fi.Name())

		if matched {
			// ignore .git files
			if !strings.Contains(path, "/.git/") {

				read, err := ioutil.ReadFile(path)
				if err != nil {
					return err
				}

				// todo reduce to terraform tokens by moving to helm chart?
				newContents := string(read)
				newContents = strings.Replace(newContents, "<CLOUD_REGION>", tokens.CloudRegion, -1)
				newContents = strings.Replace(newContents, "<CLUSTER_NAME>", tokens.ClusterName, -1)
				newContents = strings.Replace(newContents, "<CONTAINER_REGISTRY_URL>", tokens.ContainerRegistryURL, -1) // todo need to fix metaphor repo names
				newContents = strings.Replace(newContents, "<DOMAIN_NAME>", tokens.DomainName, -1)
				newContents = strings.Replace(newContents, "<METAPHOR_DEVELOPMENT_INGRESS_URL>", tokens.MetaphorDevelopmentIngressURL, -1)
				newContents = strings.Replace(newContents, "<METAPHOR_PRODUCTION_INGRESS_URL>", tokens.MetaphorProductionIngressURL, -1)
				newContents = strings.Replace(newContents, "<METAPHOR_STAGING_INGRESS_URL>", tokens.MetaphorStagingIngressURL, -1)

				err = ioutil.WriteFile(path, []byte(newContents), 0)
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
}
