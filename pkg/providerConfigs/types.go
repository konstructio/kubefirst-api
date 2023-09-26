/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package providerConfigs

type GitopsDirectoryValues struct {
	AlertsEmail               string
	AtlantisAllowList         string
	CloudProvider             string
	CloudRegion               string
	ClusterId                 string
	ClusterName               string
	ClusterType               string
	ContainerRegistryURL      string
	DomainName                string
	SubdomainName             string
	DNSProvider               string
	Kubeconfig                string
	KubeconfigPath            string
	KubefirstArtifactsBucket  string
	KubefirstStateStoreBucket string
	KubefirstTeam             string
	KubefirstVersion          string
	StateStoreBucketHostname  string

	ArgoCDIngressURL               string
	ArgoCDIngressNoHTTPSURL        string
	ArgoWorkflowsIngressURL        string
	ArgoWorkflowsIngressNoHTTPSURL string
	ArgoWorkflowsDir               string
	AtlantisIngressURL             string
	AtlantisIngressNoHTTPSURL      string
	AtlantisWebhookURL             string
	ChartMuseumIngressURL          string
	VaultIngressURL                string
	VaultIngressNoHTTPSURL         string
	VaultDataBucketName            string
	VouchIngressURL                string

	AwsIamArnAccountRoot string
	AwsKmsKeyId          string
	AwsNodeCapacityType  string
	AwsAccountID         string

	GoogleAuth       string
	GoogleProject    string
	GoogleUniqueness string
	ForceDestroy     string

	GitDescription       string
	GitNamespace         string
	GitProvider          string
	GitProtocol          string
	GitopsRepoGitURL     string
	GitopsRepoURL        string
	GitRunner            string
	GitRunnerDescription string
	GitRunnerNS          string
	GitURL               string

	GitHubHost  string
	GitHubOwner string
	GitHubUser  string

	GitlabHost         string
	GitlabOwner        string
	GitlabOwnerGroupID int
	GitlabUser         string

	GitopsRepoAtlantisWebhookURL string
	GitopsRepoNoHTTPSURL         string

	ExternalDNSProviderName         string
	ExternalDNSProviderTokenEnvName string
	ExternalDNSProviderSecretName   string
	ExternalDNSProviderSecretKey    string

	UseTelemetry string
}

type MetaphorTokenValues struct {
	CheckoutCWFTTemplate          string
	CloudRegion                   string
	ClusterName                   string
	CommitCWFTTemplate            string
	ContainerRegistryURL          string
	DomainName                    string
	MetaphorDevelopmentIngressURL string
	MetaphorProductionIngressURL  string
	MetaphorStagingIngressURL     string
}
