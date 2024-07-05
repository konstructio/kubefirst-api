package providerConfigs

import (
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

const (
	templateRepositoryURL = "https://github.com/dahendel/gitops-template.git"
)

func TestDetokenizeGitops(t *testing.T) {
	t.Run("successful walk function", func(t *testing.T) {
		dirPath := "./gitops-templates"
		g := &GitopsDirectoryValues{
			KubefirstStateStoreBucket: "got-test-bucket",
			CloudRegion:               "us-east-1",
			DomainName:                "example.com",
			ClusterName:               "go-test-cluster",
			AtlantisWebhookURL:        "https://atlantis.example.com",
			GitopsRepoURL:             "https://gitops-repo.example.com/git/template.git",
			ExternalDNSProviderName:   "route53",
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
		}
		
		assert.NoError(t, cloneRepo(dirPath))
		
		err := DetokenizeGitGitops(filepath.Join(dirPath, "templating"), g, "https", false)
		assert.NoError(t, err)
	})
}

func cloneRepo(dirPath string) error {
	
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		_, cloneErr := git.PlainClone(dirPath, false, &git.CloneOptions{
			URL:           templateRepositoryURL,
			SingleBranch:  true,
			ReferenceName: plumbing.NewBranchReferenceName("main"),
		})
		
		return cloneErr
	}
	
	return nil
}

func rmRepo(dirPath string) error {
	fmt.Println("Removing dir for clean clone...")
	return os.RemoveAll(dirPath)
}
