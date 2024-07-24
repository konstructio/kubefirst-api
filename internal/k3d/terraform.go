/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k3d

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	pkg "github.com/konstructio/kubefirst-api/internal"
	"github.com/rs/zerolog/log"
)

func GetGithubTerraformEnvs(config *K3dConfig, envs map[string]string, githubToken string) map[string]string {
	envs["GITHUB_TOKEN"] = config.GithubToken
	envs["AWS_ACCESS_KEY_ID"] = pkg.MinioDefaultUsername
	envs["AWS_SECRET_ACCESS_KEY"] = pkg.MinioDefaultPassword
	envs["TF_VAR_aws_access_key_id"] = pkg.MinioDefaultUsername
	envs["TF_VAR_aws_secret_access_key"] = pkg.MinioDefaultPassword

	return envs
}

func GetUsersTerraformEnvs(config *K3dConfig, envs map[string]string) map[string]string {
	envs["TF_VAR_email_address"] = "your@email.com"
	envs["TF_VAR_github_token"] = config.GithubToken
	envs["TF_VAR_vault_addr"] = VaultPortForwardURL
	envs["TF_VAR_vault_token"] = "k1_local_vault_token"
	envs["VAULT_ADDR"] = VaultPortForwardURL
	envs["VAULT_TOKEN"] = "k1_local_vault_token"
	envs["GITHUB_TOKEN"] = config.GithubToken

	return envs
}

func GetVaultTerraformEnvs(config *K3dConfig, envs map[string]string) map[string]string {
	envs["TF_VAR_email_address"] = "your@email.com"
	envs["TF_VAR_github_token"] = config.GithubToken
	envs["TF_VAR_vault_addr"] = VaultPortForwardURL
	envs["TF_VAR_vault_token"] = "k1_local_vault_token"
	envs["VAULT_ADDR"] = VaultPortForwardURL
	envs["VAULT_TOKEN"] = "k1_local_vault_token"
	envs["TF_VAR_aws_access_key_id"] = pkg.MinioDefaultUsername
	envs["TF_VAR_aws_secret_access_key"] = pkg.MinioDefaultPassword

	return envs
}

type GithubTerraformEnvs struct {
	GithubToken           string
	GithubOwner           string
	AtlantisWebhookSecret string
	KbotSSHPublicKey      string
	AwsAccessKeyId        string
	AwsSecretAccessKey    string
}

// TerraformPrep prepares Terraform files by detokenizing them.
// It processes files in the specified GitOps directory and the GitHub runner path.
func TerraformPrep(config *K3dConfig) error {
	// Define paths for Terraform and GitHub runner components
	terraformPath := filepath.Join(config.GitopsDir, "terraform")
	githubRunnerPath := filepath.Join(config.GitopsDir, "registry/kubefirst/")

	log.Info().Msgf("Repo is %s", terraformPath)

	// Detokenize Terraform files
	if err := filepath.Walk(terraformPath, detokenizeTerraform(terraformPath, config)); err != nil {
		return fmt.Errorf("error in detokenizing Terraform: %w", err)
	}

	log.Info().Msg("Applying to GitHub runner")

	// Detokenize GitHub runner files
	if err := filepath.Walk(githubRunnerPath, detokenizeTerraform(githubRunnerPath, config)); err != nil {
		return fmt.Errorf("error in detokenizing GitHub runner: %w", err)
	}

	return nil
}

// detokenizeTerraform returns a WalkFunc that detokenizes Terraform configuration files.
func detokenizeTerraform(basePath string, config *K3dConfig) filepath.WalkFunc {
	return func(path string, fi os.FileInfo, err error) error {
		// Skip directories
		if fi.IsDir() {
			return nil
		}

		// Read the file content
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}

		// Create a replacer for token substitution
		replacer := strings.NewReplacer(
			"<ADMIN_TEAM>", config.AdminTeamName,
			"<DEVELOPER_TEAM>", config.DeveloperTeamName,
			"<METAPHOR_REPO_NAME>", config.MetaphorRepoName,
			"<GIT_REPO_NAME>", config.GitopsRepoName,
		)

		// Replace tokens in the file content
		newContents := replacer.Replace(string(content))

		// Write the updated content back to the file
		if err := os.WriteFile(path, []byte(newContents), 0); err != nil {
			return fmt.Errorf("failed to write file %s: %w", path, err)
		}

		return nil
	}
}
