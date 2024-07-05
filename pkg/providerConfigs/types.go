/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package providerConfigs

type Tokens interface {
	ToTemplateVars(s string) string
}

type GitopsDirectoryValues struct {
	AlertsEmail       string
	AtlantisAllowList string
	CloudProvider     string
	CloudRegion       string
	ClusterId         string
	ClusterName       string
	ClusterType       string
	// <CERT_MANAGER_ISSUER_ANNOTATION_1>
	CertManagerIssuerAnnotation1   string
	CertManagerIssuerAnnotation2   string
	CertManagerIssuerAnnotation3   string
	CertManagerIssuerAnnotation4   string
	ContainerRegistryURL           string
	CustomTemplateValues           map[string]interface{}
	DomainName                     string
	SubdomainName                  string
	DNSProvider                    string
	Kubeconfig                     string
	KubeconfigPath                 string
	KubefirstArtifactsBucket       string
	KubefirstStateStoreBucket      string
	KubefirstTeam                  string
	KubefirstVersion               string
	StateStoreBucketHostname       string
	NodeType                       string
	NodeCount                      int
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
	RegistryPath                   string
	SecretStoreRef                 string
	Project                        string
	ClusterDestination             string
	Environment                    string
	
	AwsIamArnAccountRoot string
	AwsKmsKeyId          string
	AwsNodeCapacityType  string
	AwsAccountID         string
	
	GoogleAuth       string
	GoogleProject    string
	GoogleUniqueness string
	ForceDestroy     string
	
	K3sServersPrivateIps []string
	K3sServersPublicIps  []string
	K3sServersArgs       []string
	SshUser              string
	SshPrivateKey        string
	
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
	GitFqdn              string
	
	GitHubHost  string
	GitHubOwner string
	GitHubUser  string
	
	GitlabHost         string
	GitlabOwner        string
	GitlabOwnerGroupID int
	GitlabUser         string
	
	GitopsRepoAtlantisWebhookURL               string
	GitopsRepoNoHTTPSURL                       string
	WorkloadClusterTerraformModuleURL          string
	WorkloadClusterBootstrapTerraformModuleURL string
	
	ExternalDNSProviderName         string
	ExternalDNSProviderTokenEnvName string
	ExternalDNSProviderSecretName   string
	ExternalDNSProviderSecretKey    string
	
	UseTelemetry string
}

func (v *GitopsDirectoryValues) ToTemplateVars(s string) string {
	return ToTemplateVars(s, v)
}

type MetaphorTokenValues struct {
	CheckoutCWFTTemplate          string
	CloudRegion                   string
	ClusterName                   string
	CommitCWFTTemplate            string
	ContainerRegistryURL          string
	CustomTemplateValues          map[string]interface{}
	DomainName                    string
	MetaphorDevelopmentIngressURL string
	MetaphorProductionIngressURL  string
	MetaphorStagingIngressURL     string
}

func (m *MetaphorTokenValues) ToTemplateVars(s string) string {
	return ToTemplateVars(s, m)
}
