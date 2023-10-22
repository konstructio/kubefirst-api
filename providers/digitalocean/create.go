/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package digitalocean

import (
	"os"

	"github.com/kubefirst/kubefirst-api/internal/constants"
	"github.com/kubefirst/kubefirst-api/internal/controller"
	"github.com/kubefirst/kubefirst-api/internal/db"
	"github.com/kubefirst/kubefirst-api/internal/services"
	pkgtypes "github.com/kubefirst/kubefirst-api/pkg/types"
	"github.com/kubefirst/metrics-client/pkg/telemetry"
	"github.com/kubefirst/runtime/pkg/k8s"
	"github.com/kubefirst/runtime/pkg/ssl"
	log "github.com/sirupsen/logrus"
)

// CreateDigitaloceanCluster
func CreateDigitaloceanCluster(definition *pkgtypes.ClusterDefinition) error {
	ctrl := controller.ClusterController{}
	err := ctrl.InitController(definition)
	if err != nil {
		return err
	}

	err = ctrl.MdbCl.UpdateCluster(ctrl.ClusterName, "in_progress", true)
	if err != nil {
		return err
	}

	err = ctrl.DownloadTools(ctrl.ProviderConfig.ToolsDir)
	if err != nil {
		ctrl.HandleError(err.Error())
		return err
	}

	err = ctrl.DomainLivenessTest()
	if err != nil {
		ctrl.HandleError(err.Error())
		return err
	}

	err = ctrl.StateStoreCredentials()
	if err != nil {
		ctrl.HandleError(err.Error())
		return err
	}

	err = ctrl.GitInit()
	if err != nil {
		ctrl.HandleError(err.Error())
		return err
	}

	err = ctrl.InitializeBot()
	if err != nil {
		ctrl.HandleError(err.Error())
		return err
	}

	err = ctrl.RepositoryPrep()
	if err != nil {
		ctrl.HandleError(err.Error())
		return err
	}

	err = ctrl.RunGitTerraform()
	if err != nil {
		ctrl.HandleError(err.Error())
		return err
	}

	err = ctrl.RepositoryPush()
	if err != nil {
		ctrl.HandleError(err.Error())
		return err
	}

	err = ctrl.CreateCluster()
	if err != nil {
		ctrl.HandleError(err.Error())
		return err
	}

	err = ctrl.WaitForClusterReady()
	if err != nil {
		ctrl.HandleError(err.Error())
		return err
	}

	err = ctrl.ClusterSecretsBootstrap()
	if err != nil {
		ctrl.HandleError(err.Error())
		return err
	}

	//* check for ssl restore
	log.Info("checking for tls secrets to restore")
	secretsFilesToRestore, err := os.ReadDir(ctrl.ProviderConfig.SSLBackupDir + "/secrets")
	if err != nil {
		log.Infof("%s", err)
	}
	if len(secretsFilesToRestore) != 0 {
		// todo would like these but requires CRD's and is not currently supported
		// add crds ( use execShellReturnErrors? )
		// https://raw.githubusercontent.com/cert-manager/cert-manager/v1.11.0/deploy/crds/crd-clusterissuers.yaml
		// https://raw.githubusercontent.com/cert-manager/cert-manager/v1.11.0/deploy/crds/crd-certificates.yaml
		// add certificates, and clusterissuers
		log.Infof("found %d tls secrets to restore", len(secretsFilesToRestore))
		ssl.Restore(ctrl.ProviderConfig.SSLBackupDir, ctrl.DomainName, ctrl.ProviderConfig.Kubeconfig)
	} else {
		log.Info("no files found in secrets directory, continuing")
	}

	err = ctrl.InstallArgoCD()
	if err != nil {
		ctrl.HandleError(err.Error())
		return err
	}

	err = ctrl.InitializeArgoCD()
	if err != nil {
		ctrl.HandleError(err.Error())
		return err
	}

	err = ctrl.DeployRegistryApplication()
	if err != nil {
		ctrl.HandleError(err.Error())
		return err
	}

	err = ctrl.WaitForVault()
	if err != nil {
		ctrl.HandleError(err.Error())
		return err
	}

	err = ctrl.InitializeVault()
	if err != nil {
		ctrl.HandleError(err.Error())
		return err
	}

	// Create kubeconfig client
	kcfg := k8s.CreateKubeConfig(false, ctrl.ProviderConfig.Kubeconfig)

	// SetupMinioStorage(kcfg, ctrl.ProviderConfig.K1Dir, ctrl.GitProvider)

	//* configure vault with terraform
	//* vault port-forward
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
		ctrl.HandleError(err.Error())
		return err
	}

	err = ctrl.WriteVaultSecrets()
	if err != nil {
		ctrl.HandleError(err.Error())
		return err
	}

	err = ctrl.RunUsersTerraform()
	if err != nil {
		ctrl.HandleError(err.Error())
		return err
	}

	// Wait for console Deployment Pods to transition to Running
	log.Info("deploying kubefirst console and verifying cluster installation is complete")
	consoleDeployment, err := k8s.ReturnDeploymentObject(
		kcfg.Clientset,
		"app.kubernetes.io/name",
		"console",
		"kubefirst",
		1200,
	)
	if err != nil {
		log.Errorf("Error finding console Deployment: %s", err)

		ctrl.HandleError(err.Error())
		return err
	}
	_, err = k8s.WaitForDeploymentReady(kcfg.Clientset, consoleDeployment, 120)
	if err != nil {
		log.Errorf("Error waiting for console Deployment ready state: %s", err)

		ctrl.HandleError(err.Error())
		return err
	}

	log.Info("cluster creation complete")

	//* export and import cluster
	err = ctrl.ExportClusterRecord()
	if err != nil {
		log.Errorf("Error exporting cluster record: %s", err)
		return err
	} else {
		err = ctrl.MdbCl.UpdateCluster(ctrl.ClusterName, "status", constants.ClusterStatusProvisioned)
		if err != nil {
			return err
		}

		err = ctrl.MdbCl.UpdateCluster(ctrl.ClusterName, "in_progress", false)
		if err != nil {
			return err
		}

		log.Info("cluster creation complete")

		// Telemetry handler

		telemetry.SendEvent(ctrl.SegmentClient, telemetry.ClusterInstallCompleted, "")

		// Create default service entries
		cl, _ := db.Client.GetCluster(ctrl.ClusterName)
		err = services.AddDefaultServices(&cl)
		if err != nil {
			log.Errorf("error adding default service entries for cluster %s: %s", cl.ClusterName, err)
		}
	}

	return nil
}
