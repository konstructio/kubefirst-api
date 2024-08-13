package utils

import (
	"fmt"
	"os"

	pkg "github.com/kubefirst/kubefirst-api/internal"
	"github.com/kubefirst/kubefirst-api/internal/downloadManager"
	log "github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

func DownloadTools(kubectlClientPath, kubectlClientVersion, localOs, localArchitecture, terraformClientVersion, toolsDirPath string) error {
	log.Info().Msg("starting downloads...")

	// create folder if it doesn't exist
	err := pkg.CreateDirIfNotExist(toolsDirPath)
	if err != nil {
		return err
	}

	group := errgroup.Group{}

	group.Go(func() error {
		kubectlDownloadURL := fmt.Sprintf(
			"https://dl.k8s.io/release/%s/bin/%s/%s/kubectl",
			kubectlClientVersion,
			localOs,
			localArchitecture,
		)
		log.Info().Msgf("Downloading kubectl from: %s", kubectlDownloadURL)
		err = downloadManager.DownloadFile(kubectlClientPath, kubectlDownloadURL)
		if err != nil {
			return fmt.Errorf("error downloading kubectl file: %w", err)
		}

		if err := os.Chmod(kubectlClientPath, 0o755); err != nil {
			return fmt.Errorf("failed to chmod kubectl: %w", err)
		}

		kubectlStdOut, kubectlStdErr, err := pkg.ExecShellReturnStrings(kubectlClientPath, "version", "--client=true", "-oyaml")
		log.Info().Msgf("-> kubectl version:\n\t%s\n\t%s\n", kubectlStdOut, kubectlStdErr)
		if err != nil {
			return fmt.Errorf("failed to call kubectl version: %w", err)
		}

		log.Info().Msg("Kubectl download finished")
		return nil
	})

	group.Go(func() error {
		terraformDownloadURL := fmt.Sprintf(
			"https://releases.hashicorp.com/terraform/%s/terraform_%s_%s_%s.zip",
			terraformClientVersion,
			terraformClientVersion,
			localOs,
			localArchitecture,
		)
		log.Info().Msgf("Downloading terraform from %s", terraformDownloadURL)
		terraformDownloadZipPath := fmt.Sprintf("%s/terraform.zip", toolsDirPath)
		err = downloadManager.DownloadFile(terraformDownloadZipPath, terraformDownloadURL)
		if err != nil {
			return fmt.Errorf("error downloading terraform file: %w", err)
		}

		if err := downloadManager.Unzip(terraformDownloadZipPath, toolsDirPath); err != nil {
			return fmt.Errorf("error unzipping terraform file: %w", err)
		}

		if err := os.Chmod(toolsDirPath, 0o777); err != nil {
			return fmt.Errorf("failed to chmod %q: %w", toolsDirPath, err)
		}

		if err := os.Chmod(fmt.Sprintf("%s/terraform", toolsDirPath), 0o755); err != nil {
			return fmt.Errorf("failed to chmod %q: %w", fmt.Sprintf("%s/terraform", toolsDirPath), err)
		}

		if err := os.RemoveAll(fmt.Sprintf("%s/terraform.zip", toolsDirPath)); err != nil {
			return fmt.Errorf("failed to remove terraform.zip: %w", err)
		}

		// todo output terraform client version to be consistent with others
		log.Info().Msg("Terraform download finished")
		return nil
	})

	if err := group.Wait(); err != nil {
		return err
	}

	return nil
}
