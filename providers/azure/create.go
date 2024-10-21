package azure

import (
	"fmt"
	"os"

	"github.com/konstructio/kubefirst-api/internal/constants"
	"github.com/konstructio/kubefirst-api/internal/controller"
	"github.com/konstructio/kubefirst-api/internal/secrets"
	"github.com/konstructio/kubefirst-api/internal/services"
	"github.com/konstructio/kubefirst-api/internal/ssl"
	"github.com/konstructio/kubefirst-api/pkg/k8s"
	pkgtypes "github.com/konstructio/kubefirst-api/pkg/types"
	"github.com/rs/zerolog/log"
)

func CreateAzureCluster(definition *pkgtypes.ClusterDefinition) error {
	ctrl := controller.ClusterController{}

	log.Debug().Msg("initializing controller")
	err := ctrl.InitController(definition)
	if err != nil {
		return fmt.Errorf("error initializing controller: %w", err)
	}

	ctrl.Cluster.InProgress = true
	log.Debug().Msg("updating cluster secrets")
	err = secrets.UpdateCluster(ctrl.KubernetesClient, ctrl.Cluster)
	if err != nil {
		return fmt.Errorf("error updating cluster secrets: %w", err)
	}

	log.Debug().Msg("downling tools")
	err = ctrl.DownloadTools(ctrl.ProviderConfig.ToolsDir)
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error downloading tools: %w", err)
	}

	log.Debug().Msg("checking domain liveness")
	err = ctrl.DomainLivenessTest()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error running domain liveness test: %w", err)
	}

	log.Debug().Msg("creating state store")
	err = ctrl.StateStoreCredentials()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error creating state store: %w", err)
	}

	log.Debug().Msg("initializing git")
	err = ctrl.GitInit()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error initializing git: %w", err)
	}

	log.Debug().Msg("initializing bot")
	err = ctrl.InitializeBot()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error initializing bot: %w", err)
	}

	log.Debug().Msg("repository prep")
	err = ctrl.RepositoryPrep()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error preparing repository: %w", err)
	}

	log.Debug().Msg("running git terraform")
	err = ctrl.RunGitTerraform()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error running git terraform: %w", err)
	}

	log.Debug().Msg("pushing repository")
	err = ctrl.RepositoryPush()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error pushing repository: %w", err)
	}

	log.Debug().Msg("pushing repository")
	err = ctrl.CreateCluster()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error creating cluster: %w", err)
	}

	log.Debug().Msg("creating cluster")
	err = ctrl.CreateCluster()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error creating cluster: %w", err)
	}

	log.Debug().Msg("bootstrapping cluster secrets")
	err = ctrl.ClusterSecretsBootstrap()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error bootstrapping cluster secrets: %w", err)
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

	log.Debug().Msg("installing argocd")
	err = ctrl.InstallArgoCD()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error installing argocd: %w", err)
	}

	log.Debug().Msg("initializing argocd")
	err = ctrl.InitializeArgoCD()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error initializing argocd: %w", err)
	}

	log.Debug().Msg("deploying registry application")
	err = ctrl.DeployRegistryApplication()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error deploying registry application: %w", err)
	}

	log.Debug().Msg("waiting for vault readiness")
	err = ctrl.WaitForVault()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error waiting for vault: %w", err)
	}

	log.Debug().Msg("initializing vault")
	err = ctrl.InitializeVault()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error initializing vault: %w", err)
	}

	// Create kubeconfig client
	log.Debug().Msg("creating kubeconfig")
	kcfg, err := k8s.CreateKubeConfig(false, ctrl.ProviderConfig.Kubeconfig)
	if err != nil {
		return fmt.Errorf("error creating kubeconfig: %w", err)
	}

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

	log.Debug().Msg("running vault terraform")
	err = ctrl.RunVaultTerraform()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error running vault terraform: %w", err)
	}

	log.Debug().Msg("write vault secrets")
	err = ctrl.WriteVaultSecrets()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error writing vault secrets: %w", err)
	}

	log.Debug().Msg("running users terraform")
	err = ctrl.RunUsersTerraform()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error running users terraform: %w", err)
	}

	// Wait for last sync wave app transition to Running
	log.Info().Msg("waiting for final sync wave Deployment to transition to Running")
	crossplaneDeployment, err := k8s.ReturnDeploymentObject(
		kcfg.Clientset,
		"app.kubernetes.io/instance",
		"crossplane",
		"crossplane-system",
		3600,
	)
	if err != nil {
		log.Error().Msgf("Error finding crossplane Deployment: %s", err)
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error finding crossplane Deployment: %w", err)
	}
	log.Info().Msg("waiting on dns, tls certificates from letsencrypt and remaining sync waves.\n this may take up to 60 minutes but regularly completes in under 20 minutes")
	_, err = k8s.WaitForDeploymentReady(kcfg.Clientset, crossplaneDeployment, 3600)
	if err != nil {
		log.Error().Msgf("Error waiting for all Apps to sync ready state: %s", err)
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error waiting for all Apps to sync ready state: %w", err)
	}

	// * export and import cluster
	log.Debug().Msg("exporting cluster record")
	err = ctrl.ExportClusterRecord()
	if err != nil {
		log.Error().Msgf("Error exporting cluster record: %s", err)
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error exporting cluster record: %w", err)
	}

	// Create default service entries
	log.Debug().Msg("getting cluster secrets")
	cl, err := secrets.GetCluster(ctrl.KubernetesClient, ctrl.ClusterName)
	if err != nil {
		log.Error().Msgf("error getting cluster %s: %s", ctrl.ClusterName, err)
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error getting cluster %s: %w", ctrl.ClusterName, err)
	}

	log.Debug().Msg("adding default services")
	err = services.AddDefaultServices(cl)
	if err != nil {
		log.Error().Msgf("error adding default service entries for cluster %s: %s", cl.ClusterName, err)
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error adding default service entries for cluster %s: %w", cl.ClusterName, err)
	}

	if ctrl.InstallKubefirstPro {
		log.Info().Msg("waiting for kubefirst-api Deployment to transition to Running")
		kubefirstAPI, err := k8s.ReturnDeploymentObject(
			kcfg.Clientset,
			"app.kubernetes.io/name",
			"kubefirst-api",
			"kubefirst",
			1200,
		)
		if err != nil {
			log.Error().Msgf("Error finding kubefirst api Deployment: %s", err)
			ctrl.UpdateClusterOnError(err.Error())
			return fmt.Errorf("error finding kubefirst api Deployment: %w", err)
		}

		_, err = k8s.WaitForDeploymentReady(kcfg.Clientset, kubefirstAPI, 300)
		if err != nil {
			log.Error().Msgf("Error waiting for kubefirst-api to transition to Running: %s", err)
			ctrl.UpdateClusterOnError(err.Error())
			return fmt.Errorf("error waiting for kubefirst-api to transition to Running: %w", err)
		}
	}

	// Wait for last sync wave app transition to Running
	log.Info().Msg("waiting for final sync wave Deployment to transition to Running")
	argocdDeployment, err := k8s.ReturnDeploymentObject(
		kcfg.Clientset,
		"app.kubernetes.io/name",
		"argocd-server",
		"argocd",
		3600,
	)
	if err != nil {
		log.Error().Msgf("Error finding argocd Deployment: %s", err)
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error finding argocd Deployment: %w", err)
	}
	_, err = k8s.WaitForDeploymentReady(kcfg.Clientset, argocdDeployment, 3600)
	if err != nil {
		log.Error().Msgf("Error waiting for argocd deployment to enter Ready state: %s", err)
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error waiting for argocd deployment to enter Ready state: %w", err)
	}

	ctrl.Cluster.Status = constants.ClusterStatusProvisioned
	ctrl.Cluster.InProgress = false
	err = secrets.UpdateCluster(ctrl.KubernetesClient, ctrl.Cluster)
	if err != nil {
		log.Error().Msgf("error updating cluster status: %s", err)
		return fmt.Errorf("error updating cluster status: %w", err)
	}

	log.Info().Msgf("Azure infrastructure successfully created: %s", definition.ClusterName)

	return nil
}
