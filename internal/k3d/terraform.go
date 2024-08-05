/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k3d

import (
	"os"
	"path/filepath"
	"strings"

	pkg "github.com/kubefirst/kubefirst-api/internal"
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

func TerraformPrep(config *K3dConfig) error {

	path := config.GitopsDir + "/terraform"
	log.Info().Msgf("Repo is %s", path)
	err := filepath.Walk(path, detokenizeterraform(path, config))
	if err != nil {
		return err
	}
	return nil
}

func detokenizeterraform(path string, config *K3dConfig) filepath.WalkFunc {
	return filepath.WalkFunc(func(path string, fi os.FileInfo, err error) error {

		if fi.IsDir() {
			return nil
		}

		read, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		replacer := strings.NewReplacer(
			"<ADMIN_TEAM>", config.AdminTeamName,
			"<DEVELOPER_TEAM>", config.DeveloperTeamName,
			"<METAPHOR_REPO_NAME>", config.MetaphorRepoName,
			"<GIT_REPO_NAME>", config.GitopsRepoName,
		)

		newContents := replacer.Replace(string(read))

		err = os.WriteFile(path, []byte(newContents), 0)
		if err != nil {
			return err
		}

		return nil
	})

}
