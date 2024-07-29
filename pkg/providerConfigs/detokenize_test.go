package providerConfigs

import (
	"context"
	"fmt"
	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argo-cd/v2/test"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"io/fs"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	templateRepositoryURL = "https://github.com/kubefirst/gitops-template.git"
	tempDir               = "./"
)

var (
	gitOpsDirList = []string{
		"akamai-github",
		"aws-github",
		"aws-gitlab",
		"civo-github",
		"civo-gitlab",
		"digitalocean-github",
		"digitalocean-gitlab",
		"google-github",
		"google-gitlab",
		"vultr-github",
		"vultr-gitlab",
	}

	k3dDirList = []string{
		"k3d-github",
		"k3d-gitlab",
		"k3s-gitlab",
	}
)

func setGitopsDirectoryValues() *GitopsDirectoryValues {
	return &GitopsDirectoryValues{
		AlertsEmail:          "alerts@example.com",
		AtlantisAllowList:    "192.168.0.0/16",
		CloudProvider:        "aws",
		CloudRegion:          "us-east-1",
		ClusterId:            "test-cluster-id",
		ClusterName:          "test-cluster-name",
		ClusterType:          "eks",
		ContainerRegistryURL: "https://container-registry.example.com",
		CustomTemplateValues: map[string]interface{}{
			"repo_name":          "testKubeFirstCustomTemplating",
			"archive_on_destroy": false,
			"clusters": map[string]string{
				"looper": "coolString",
			},
			"example-cm": "example-cm-name",
			"namespace":  "test-namespace",
			"exampleCmData": map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
		DomainName:                        "example.com",
		SubdomainName:                     "test",
		DNSProvider:                       "route53",
		Kubeconfig:                        "kubeconfig-data",
		KubeconfigPath:                    "/path/to/kubeconfig",
		KubefirstArtifactsBucket:          "kubefirst-artifacts-bucket",
		KubefirstStateStoreBucket:         "kubefirst-state-store-bucket",
		KubefirstTeam:                     "test-team",
		KubefirstVersion:                  "1.2.3",
		StateStoreBucketHostname:          "state-store-bucket.example.com",
		NodeType:                          "m5.large",
		NodeCount:                         3,
		ArgoCDIngressURL:                  "https://argocd.example.com",
		ArgoCDIngressNoHTTPSURL:           "http://argocd.example.com",
		ArgoWorkflowsIngressURL:           "https://argo-workflows.example.com",
		ArgoWorkflowsIngressNoHTTPSURL:    "http://argo-workflows.example.com",
		ArgoWorkflowsDir:                  "/path/to/argo-workflows",
		AtlantisIngressURL:                "https://atlantis.example.com",
		AtlantisIngressNoHTTPSURL:         "http://atlantis.example.com",
		AtlantisWebhookURL:                "https://atlantis-webhook.example.com",
		ChartMuseumIngressURL:             "https://chart-museum.example.com",
		VaultIngressURL:                   "https://vault.example.com",
		VaultIngressNoHTTPSURL:            "http://vault.example.com",
		VaultDataBucketName:               "vault-data-bucket",
		VouchIngressURL:                   "https://vouch.example.com",
		RegistryPath:                      "/path/to/registry",
		SecretStoreRef:                    "secret-store-ref",
		Project:                           "test-project",
		ClusterDestination:                "test-cluster-destination",
		Environment:                       "test-environment",
		AwsIamArnAccountRoot:              "arn:aws:iam::123456789012:root",
		AwsKmsKeyId:                       "1234abcd-12ab-34cd-56ef-123456789abc",
		AwsNodeCapacityType:               "ON_DEMAND",
		AwsAccountID:                      "123456789012",
		GoogleAuth:                        "google-auth-data",
		GoogleProject:                     "test-google-project",
		GoogleUniqueness:                  "test-google-uniqueness",
		ForceDestroy:                      "true",
		K3sServersPrivateIps:              []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"},
		K3sServersPublicIps:               []string{"1.2.3.4", "5.6.7.8", "9.10.11.12"},
		K3sServersArgs:                    []string{"--arg1", "--arg2=value2", "--arg3"},
		SshUser:                           "test-user",
		SshPrivateKey:                     "ssh-private-key-data",
		GitDescription:                    "test-git-description",
		GitNamespace:                      "test-git-namespace",
		GitProvider:                       "github",
		GitProtocol:                       "https",
		GitopsRepoGitURL:                  "https://github.com/test-org/test-repo.git",
		GitopsRepoURL:                     "https://github.com/test-org/test-repo",
		GitRunner:                         "test-git-runner",
		GitRunnerDescription:              "test-git-runner-description",
		GitRunnerNS:                       "test-git-runner-ns",
		GitURL:                            "https://github.com",
		GitFqdn:                           "github.com",
		GitHubHost:                        "github.com",
		GitHubOwner:                       "test-org",
		GitHubUser:                        "test-user",
		GitlabHost:                        "gitlab.com",
		GitlabOwner:                       "test-org",
		GitlabOwnerGroupID:                123,
		GitlabUser:                        "test-user",
		GitopsRepoAtlantisWebhookURL:      "https://atlantis-webhook.example.com/test-repo",
		GitopsRepoNoHTTPSURL:              "http://github.com/test-org/test-repo",
		WorkloadClusterTerraformModuleURL: "https://github.com/test-org/test-module.git",
		WorkloadClusterBootstrapTerraformModuleURL: "https://github.com/test-org/test-bootstrap-module.git",
		ExternalDNSProviderName:                    "route53",
		ExternalDNSProviderTokenEnvName:            "AWS_ACCESS_KEY_ID",
		ExternalDNSProviderSecretName:              "external-dns-provider-secret",
		ExternalDNSProviderSecretKey:               "access-key",
		UseTelemetry:                               "true",
	}
}

type DetokenizeSuite struct {
	k                  *fake.FakeDynamicClient
	t                  *testing.T
	templatesDirectory string

	GitopsTokens   *GitopsDirectoryValues
	MetaphorTokens *MetaphorTokenValues
}

func createRepositoryTempDirs(t *testing.T, d *DetokenizeSuite) {
	var err error
	d.templatesDirectory, err = os.MkdirTemp(tempDir, "templates")
	assert.NoError(t, err)
}

func newFakeClient(t *testing.T, d *DetokenizeSuite) {
	scheme := runtime.NewScheme()
	schemeBuilder := runtime.SchemeBuilder{}
	assert.NoError(t, schemeBuilder.AddToScheme(scheme))

	// Create the mocked kubernetes client and add the ArgoCD Application scheme to it so that
	// we can apply the rendered manifests to the fake client and check for errors.
	d.k = fake.NewSimpleDynamicClient(scheme,
		&v1alpha1.Application{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "argoproj.io/v1alpha1",
				Kind:       "Application",
			},
		})

	assert.NoError(t, v1alpha1.AddToScheme(scheme))
}

func setMetaphorTokenValues(d *DetokenizeSuite) *MetaphorTokenValues {
	return &MetaphorTokenValues{
		CloudRegion:                   d.GitopsTokens.CloudRegion,
		ClusterName:                   d.GitopsTokens.ClusterName,
		ContainerRegistryURL:          d.GitopsTokens.ContainerRegistryURL,
		CustomTemplateValues:          nil,
		GitProtocol:                   d.GitopsTokens.GitProtocol,
		DomainName:                    d.GitopsTokens.DomainName,
		MetaphorDevelopmentIngressURL: fmt.Sprintf("https://metaphor-development.%s", d.GitopsTokens.DomainName),
		MetaphorProductionIngressURL:  fmt.Sprintf("https://metaphor.%s", d.GitopsTokens.DomainName),
		MetaphorStagingIngressURL:     fmt.Sprintf("https://metaphor-stage.%s", d.GitopsTokens.DomainName),
	}
}

func TestDetokenize(t *testing.T) {
	d := SetupSuite(t)

	clean := os.Getenv("K1_TEST_CLEANUP")

	if clean == "true" || clean == "" {
		defer t.Cleanup(d.TearDownSuite)
	}

	t.Run("DetokenizeGitops", d.TestDetokenizeGitops)
	t.Run("DetokenizeMetaphor", d.TestDetokenizeMetaphor)
	t.Run("DetokenizeK3d", d.TestDetokenizeK3d)
	t.Run("DetokenizeGitopsWithCustomTemplateValues", d.TestDetokenizeGitopsWithCustomTemplateValues)
}

// SetupTest initializes the necessary dependencies and configurations for the DetokenizeSuite test suite.
func SetupSuite(t *testing.T) *DetokenizeSuite {
	d := &DetokenizeSuite{t: t}
	createRepositoryTempDirs(t, d)
	newFakeClient(t, d)
	d.GitopsTokens = setGitopsDirectoryValues()
	d.MetaphorTokens = setMetaphorTokenValues(d)

	err := cloneRepo(templateRepositoryURL, d.templatesDirectory)
	assert.NoError(t, err)

	return d
}

func (d *DetokenizeSuite) TearDownSuite() {
	err := os.RemoveAll(d.templatesDirectory)
	assert.NoError(d.t, err)
}

func (d *DetokenizeSuite) TestDetokenizeGitops(t *testing.T) {
	for _, dir := range gitOpsDirList {
		currentPath := filepath.Join(d.templatesDirectory, dir)
		assert.NoError(t, Detokenize(currentPath, d.GitopsTokens,
			"https", false))
		assert.NoError(t, filepath.Walk(currentPath, testManifestValidity(currentPath, d)))
	}
}

// TestDetokenizeK3d tests the detokenization of the k3d cluster-types directory.
// Leaving this separate from the other cluster-types directories because it is slightly different,
// and we may want to test it separately in the future.
func (d *DetokenizeSuite) TestDetokenizeK3d(t *testing.T) {
	// Set CloudProvider to k3s to properly render the k3d TF
	d.GitopsTokens.CloudProvider = "k3s"
	for _, dir := range k3dDirList {
		currentPath := filepath.Join(d.templatesDirectory, dir)
		assert.NoError(t, Detokenize(currentPath, d.GitopsTokens,
			"https", false))
		assert.NoError(t, filepath.Walk(currentPath, testManifestValidity(currentPath, d)))
	}
	// Reset cloud provider for remainder of tests
	d.GitopsTokens.CloudProvider = "aws"
}

func (d *DetokenizeSuite) TestDetokenizeMetaphor(t *testing.T) {
	err := Detokenize(filepath.Join(d.templatesDirectory, "metaphor", ".argo"), d.MetaphorTokens, d.GitopsTokens.GitProtocol, false)
	assert.NoError(t, err)
}

func (d *DetokenizeSuite) TestDetokenizeGitopsWithCustomTemplateValues(t *testing.T) {
	templatingDir := filepath.Join(d.templatesDirectory, "templating")
	if _, err := os.Stat(templatingDir); os.IsNotExist(err) {
		fmt.Println("Skipping TestDetokenizeGitopsWithCustomTemplateValues because the templating directory does not exist")
		return
	}
	assert.NoError(t, Detokenize(filepath.Join(d.templatesDirectory, "templating"),
		d.GitopsTokens, "https", false))

	assert.NoError(t, filepath.Walk(templatingDir, testManifestValidity(templatingDir, d)))
}

func cloneRepo(src string, dirPath string) error {
	_, cloneErr := git.PlainClone(dirPath, false, &git.CloneOptions{
		URL:           src,
		SingleBranch:  true,
		ReferenceName: plumbing.NewBranchReferenceName("main"),
	})

	return cloneErr
}

func applyFile(path string, k *fake.FakeDynamicClient) error {
	f, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	us := test.YamlToUnstructured(string(f))

	_, err = k.Resource(v1alpha1.SchemeGroupVersion.WithResource(us.GetResourceVersion())).
		Namespace(us.GetNamespace()).
		Create(context.Background(), us, metav1.CreateOptions{})

	return err
}

func delApp(path string, k *fake.FakeDynamicClient) error {
	f, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	app := &v1alpha1.Application{}
	if err = yaml.Unmarshal(f, app); err != nil {
		return err
	}

	return k.Resource(v1alpha1.SchemeGroupVersion.WithResource("Application")).Namespace(app.Namespace).
		DeleteCollection(context.Background(), metav1.DeleteOptions{}, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("app.kubernetes.io/instance=%s", app.Name),
		})
}

func testManifestValidity(currentPath string, d *DetokenizeSuite) filepath.WalkFunc {
	return func(path string, info fs.FileInfo, err error) error {
		if !info.IsDir() {
			if strings.Contains(info.Name(), ".yaml") {
				// Apply the manifests to the fake client and check for errors.
				assert.NoError(d.t, applyFile(path, d.k))
			}
			// Delete the manifests from the fake client so that we don't get duplicate resource errors
			return delApp(currentPath, d.k)
		}
		return filepath.SkipDir
	}
}
