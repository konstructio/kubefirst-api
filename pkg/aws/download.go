/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package aws

import (
	"fmt"
	"os"

	pkg "github.com/konstructio/kubefirst-api/internal"
	"github.com/konstructio/kubefirst-api/internal/downloadManager"
	"github.com/konstructio/kubefirst-api/pkg/providerConfigs"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

func DownloadTools(awsConfig *providerConfigs.ProviderConfig, kubectlClientVersion string, terraformClientVersion string) error {
	log.Info().Msg("starting downloads...")

	// create folder if it doesn't exist
	err := pkg.CreateDirIfNotExist(awsConfig.ToolsDir)
	if err != nil {
		return fmt.Errorf("error creating tools dir: %w", err)
	}

	eg := errgroup.Group{}

	eg.Go(func() error {
		kubectlDownloadURL := fmt.Sprintf(
			"https://dl.k8s.io/release/%s/bin/%s/%s/kubectl",
			kubectlClientVersion,
			pkg.LocalhostOS,
			pkg.LocalhostARCH,
		)
		log.Info().Msgf("Downloading kubectl from: %s", kubectlDownloadURL)
		err = downloadManager.DownloadFile(awsConfig.KubectlClient, kubectlDownloadURL)
		if err != nil {
			return fmt.Errorf("error downloading kubectl file: %w", err)
		}

		if err := os.Chmod(awsConfig.KubectlClient, 0o755); err != nil {
			return fmt.Errorf("failed to chmod kubectl: %w", err)
		}

		kubectlStdOut, kubectlStdErr, err := pkg.ExecShellReturnStrings(awsConfig.KubectlClient, "version", "--client=true", "-oyaml")
		log.Info().Msgf("-> kubectl version:\n\t%s\n\t%s\n", kubectlStdOut, kubectlStdErr)
		if err != nil {
			return fmt.Errorf("failed to call kubectl version: %w", err)
		}

		log.Info().Msg("Kubectl download finished")
		return nil
	})

	eg.Go(func() error {
		terraformDownloadURL := fmt.Sprintf(
			"https://releases.hashicorp.com/terraform/%s/terraform_%s_%s_%s.zip",
			terraformClientVersion,
			terraformClientVersion,
			pkg.LocalhostOS,
			pkg.LocalhostARCH,
		)
		log.Info().Msgf("Downloading terraform from %s", terraformDownloadURL)
		terraformDownloadZipPath := fmt.Sprintf("%s/terraform.zip", awsConfig.ToolsDir)
		err = downloadManager.DownloadFile(terraformDownloadZipPath, terraformDownloadURL)
		if err != nil {
			return fmt.Errorf("error downloading terraform file, %w", err)
		}

		downloadManager.Unzip(terraformDownloadZipPath, awsConfig.ToolsDir)

		err = os.Chmod(awsConfig.ToolsDir, 0o777)
		if err != nil {
			return fmt.Errorf("failed to chmod %q: %w", awsConfig.ToolsDir, err)
		}

		err = os.Chmod(fmt.Sprintf("%s/terraform", awsConfig.ToolsDir), 0o755)
		if err != nil {
			return fmt.Errorf("failed to chmod %q: %w", awsConfig.ToolsDir, err)
		}
		err = os.RemoveAll(fmt.Sprintf("%s/terraform.zip", awsConfig.ToolsDir))
		if err != nil {
			return fmt.Errorf("failed to remove terraform.zip: %w", err)
		}

		log.Info().Msg("Terraform download finished")
		return nil
	})

	if err := eg.Wait(); err != nil {
		return fmt.Errorf("error downloading tools: %w", err)
	}

	log.Info().Msg("downloads finished")
	return nil
}
