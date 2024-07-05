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
	renderedFileText = `terraform {
  backend "s3" {
    bucket = "got-test-bucket"
    key    = "terraform/github/terraform.tfstate"

    region  = "us-east-1"
    encrypt = true
  }
  required_providers {
    github = {
      source  = "integrations/github"
      version = "~> 5.17.0"
    }
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.0"
    }
  }
}

module "gitops" {
  source = "./modules/repository"

  repo_name          = testKubeFirstCustomTemplating
  archive_on_destroy = false
  auto_init          = false # set to false if importing an existing repository
  team_developers_id = github_team.developers.id
  team_admins_id     = github_team.admins.id
}

locals {}

module "example-loop" {
  source = "./modules/repository"
  repo_name = looper
}

`
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
		
		//renderedFile, err := os.ReadFile(filePath)
		//assert.NoError(t, err)
		//buff := bytes.NewBuffer(renderedFile)
		//assert.Equal(t, buff.Bytes(), []byte(renderedFileText))
		//
		//assert.NoError(t, os.WriteFile(dirPath+"/repos.tf", tmplFile, 0644))
	})
}

func cloneRepo(dirPath string) error {
	
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		_, cloneErr := git.PlainClone(dirPath, false, &git.CloneOptions{
			URL:           "https://github.com/dahendel/gitops-template.git",
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
