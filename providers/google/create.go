/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package google

import (
	"fmt"
	"os"

	"github.com/kubefirst/kubefirst-api/internal/constants"
	"github.com/kubefirst/kubefirst-api/internal/controller"
	"github.com/kubefirst/kubefirst-api/internal/db"
	"github.com/kubefirst/kubefirst-api/internal/services"
	"github.com/kubefirst/kubefirst-api/internal/telemetryShim"
	"github.com/kubefirst/kubefirst-api/pkg/google"
	"github.com/kubefirst/kubefirst-api/pkg/segment"
	pkgtypes "github.com/kubefirst/kubefirst-api/pkg/types"
	"github.com/kubefirst/runtime/pkg/k8s"
	log "github.com/sirupsen/logrus"
)

func CreateGoogleCluster(definition *pkgtypes.ClusterDefinition) error {
	ctrl := controller.ClusterController{}
	err := ctrl.InitController(definition)
	if err != nil {
		return err
	}

	// Update cluster status in database
	err = ctrl.MdbCl.UpdateCluster(ctrl.ClusterName, "in_progress", true)
	if err != nil {
		return err
	}

	// TODO Validate Google region
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("error getting home path: %s", err)
	}

	err = google.WriteGoogleApplicationCredentialsFile(definition.GoogleAuth.KeyFile, homeDir)
	if err != nil {
		log.Fatalf("error writing google application credentials file: %s", err)
	}

	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", fmt.Sprintf("%s/.k1/application-default-credentials.json", homeDir))

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

	//Checks for existing repos
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

	//Where detokeinization happens
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

	err = ctrl.DetokenizeKMSKeyID()
	if err != nil {
		ctrl.HandleError(err.Error())
		return err
	}

	kcfg, err := ctrl.GoogleClient.GetContainerClusterAuth(ctrl.ClusterName, []byte(ctrl.GoogleAuth.KeyFile))
	if err != nil {
		return err
	}
	//Save config
	ctrl.Kcfg = kcfg

	err = ctrl.WaitForClusterReady()
	if err != nil {
		ctrl.HandleError(err.Error())
		return err
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

	// Needs wait after cluster create
	err = ctrl.MdbCl.UpdateCluster(ctrl.ClusterName, "in_progress", false)
	if err != nil {
		return err
	}

	err = ctrl.ClusterSecretsBootstrap()
	if err != nil {
		ctrl.HandleError(err.Error())
		return err
	}

	err = ctrl.MdbCl.UpdateCluster(ctrl.ClusterName, "cluster_secrets_created_check", true)
	if err != nil {
		err = ctrl.MdbCl.UpdateCluster(ctrl.ClusterName, "in_progress", false)
		if err != nil {
			return err
		}

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

	err = ctrl.InitializeVault()
	if err != nil {
		ctrl.HandleError(err.Error())
		return err
	}

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
		log.Errorf("Error finding kubefirst api Deployment: %s", err)
		ctrl.HandleError(err.Error())
		return err
	}
	_, err = k8s.WaitForDeploymentReady(kcfg.Clientset, consoleDeployment, 120)
	if err != nil {
		log.Errorf("Error waiting for kubefirst api Deployment ready state: %s", err)

		ctrl.HandleError(err.Error())
		return err
	}

	cluster1KubefirstApiStopChannel := make(chan struct{}, 1)
	defer func() {
		close(cluster1KubefirstApiStopChannel)
	}()

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
		rec, err := ctrl.GetCurrentClusterRecord()
		if err != nil {
			return err
		}

		// Telemetry handler
		segmentClient, err := telemetryShim.SetupTelemetry(rec)
		if err != nil {
			return err
		}
		defer segmentClient.Client.Close()

		telemetryShim.Transmit(segmentClient, segment.MetricClusterInstallCompleted, "")

		// Create default service entries
		cl, _ := db.Client.GetCluster(ctrl.ClusterName)
		err = services.AddDefaultServices(&cl)
		if err != nil {
			log.Errorf("error adding default service entries for cluster %s: %s", cl.ClusterName, err)
		}
	}

	return nil
}
