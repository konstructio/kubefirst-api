/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k3s

import (
	"fmt"
	"os"

	"github.com/konstructio/kubefirst-api/internal/controller"
	"github.com/konstructio/kubefirst-api/internal/k8s"
	"github.com/konstructio/kubefirst-api/internal/secrets"
	"github.com/konstructio/kubefirst-api/internal/ssl"
	pkgtypes "github.com/konstructio/kubefirst-api/pkg/types"
	log "github.com/rs/zerolog/log"
)

// Createk3sCluster
func CreateK3sCluster(definition *pkgtypes.ClusterDefinition) error {
	ctrl := controller.ClusterController{}
	err := ctrl.InitController(definition)
	if err != nil {
		return fmt.Errorf("error initializing controller: %w", err)
	}

	ctrl.Cluster.InProgress = true
	err = secrets.UpdateCluster(ctrl.KubernetesClient, ctrl.Cluster)
	if err != nil {
		return fmt.Errorf("error updating cluster secrets: %w", err)
	}

	err = ctrl.DownloadTools(ctrl.ProviderConfig.ToolsDir)
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error downloading tools: %w", err)
	}

	err = ctrl.DomainLivenessTest()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error in domain liveness test: %w", err)
	}

	err = ctrl.StateStoreCredentials()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error in state store credentials: %w", err)
	}

	err = ctrl.GitInit()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error initializing git: %w", err)
	}

	err = ctrl.InitializeBot()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error initializing bot: %w", err)
	}

	err = ctrl.RepositoryPrep()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error in repository preparation: %w", err)
	}

	err = ctrl.RunGitTerraform()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error running git terraform: %w", err)
	}

	err = ctrl.RepositoryPush()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error pushing repository: %w", err)
	}

	err = ctrl.CreateCluster()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error creating cluster: %w", err)
	}

	err = ctrl.WaitForClusterReady()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error waiting for cluster to be ready: %w", err)
	}

	err = ctrl.ClusterSecretsBootstrap()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error in cluster secrets bootstrap: %w", err)
	}

	// * check for ssl restore
	log.Info().Msg("checking for tls secrets to restore")
	secretsFilesToRestore, err := os.ReadDir(ctrl.ProviderConfig.SSLBackupDir + "/secrets")
	if err != nil {
		if os.IsNotExist(err) {
			log.Info().Msg("no files found in secrets directory, continuing")
		} else {
			log.Info().Msgf("unable to check for TLS secrets to restore: %s", err.Error())
		}
	}

	if len(secretsFilesToRestore) != 0 {
		// todo would like these but requires CRD's and is not currently supported
		// add crds ( use execShellReturnErrors? )
		// https://raw.githubusercontent.com/cert-manager/cert-manager/v1.11.0/deploy/crds/crd-clusterissuers.yaml
		// https://raw.githubusercontent.com/cert-manager/cert-manager/v1.11.0/deploy/crds/crd-certificates.yaml
		// add certificates, and clusterissuers
		log.Info().Msgf("found %d tls secrets to restore", len(secretsFilesToRestore))
		ssl.Restore(ctrl.ProviderConfig.SSLBackupDir, ctrl.ProviderConfig.Kubeconfig)
	} else {
		log.Info().Msg("no files found in secrets directory, continuing")
	}

	err = ctrl.InstallArgoCD()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error installing ArgoCD: %w", err)
	}

	err = ctrl.InitializeArgoCD()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error initializing ArgoCD: %w", err)
	}

	err = ctrl.DeployRegistryApplication()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error deploying registry application: %w", err)
	}

	err = ctrl.WaitForVault()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error waiting for Vault: %w", err)
	}

	err = ctrl.InitializeVault()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error initializing Vault: %w", err)
	}

	// Create kubeconfig client
	kcfg, err := k8s.CreateKubeConfig(false, ctrl.ProviderConfig.Kubeconfig)
	if err != nil {
		return fmt.Errorf("error creating kubeconfig: %w", err)
	}

	// SetupMinioStorage(kcfg, ctrl.ProviderConfig.K1Dir, ctrl.GitProvider)

	// * configure vault with terraform
	// * vault port-forward
	vaultStopChannel := make(chan struct{}, 1)
	defer func() {
		close(vaultStopChannel)
	}()
	k8s.OpenPortForwardPodWrapper(
		kcfg.Clientset,
		kcfg.RestConfig,
		"vault-0",
		"vault",
		8200,
		8200,
		vaultStopChannel,
	)

	err = ctrl.RunVaultTerraform()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error running Vault terraform: %w", err)
	}

	err = ctrl.WriteVaultSecrets()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error writing Vault secrets: %w", err)
	}

	err = ctrl.RunUsersTerraform()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error running users terraform: %w", err)
	}

	if err := ctrl.FinalCheck(); err != nil {
		log.Error().Msgf("error doing final check: %s", err)
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error doing final check: %w", err)
	}

	return nil
}
