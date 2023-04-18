/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k3d

import (
	"fmt"
	"os"

	"github.com/go-git/go-git/v5"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/kubefirst/kubefirst-api/internal/controller"
	"github.com/kubefirst/kubefirst-api/internal/types"
	"github.com/kubefirst/runtime/pkg/gitClient"
	"github.com/kubefirst/runtime/pkg/k3d"
	"github.com/kubefirst/runtime/pkg/k8s"
	log "github.com/sirupsen/logrus"
)

func CreateK3DCluster(definition *types.ClusterDefinition) error {
	ctrl := controller.ClusterController{}
	err := ctrl.InitController(definition)
	if err != nil {
		return err
	}

	err = ctrl.DownloadTools(ctrl.GitProvider, ctrl.GitOwner, ctrl.ProviderConfig.(k3d.K3dConfig).ToolsDir)
	if err != nil {
		return err
	}

	err = ctrl.GitInit()
	if err != nil {
		return err
	}

	err = ctrl.InitializeBot()
	if err != nil {
		return err
	}

	err = ctrl.RepositoryPrep()
	if err != nil {
		return err
	}

	err = ctrl.RunGitTerraform()
	if err != nil {
		return err
	}

	err = ctrl.RepositoryPush()
	if err != nil {
		return err
	}

	err = ctrl.CreateCluster()
	if err != nil {
		return err
	}

	err = ctrl.ClusterSecretsBootstrap()
	if err != nil {
		return err
	}

	////* check for ssl restore
	//log.Info().Msg("checking for tls secrets to restore")
	//secretsFilesToRestore, err := ioutil.ReadDir(config.SSLBackupDir + "/secrets")
	//if err != nil {
	//	log.Info().Msgf("%s", err)
	//}
	//if len(secretsFilesToRestore) != 0 {
	//	// todo would like these but requires CRD's and is not currently supported
	//	// add crds ( use execShellReturnErrors? )
	//	// https://raw.githubusercontent.com/cert-manager/cert-manager/v1.11.0/deploy/crds/crd-clusterissuers.yaml
	//	// https://raw.githubusercontent.com/cert-manager/cert-manager/v1.11.0/deploy/crds/crd-certificates.yaml
	//	// add certificates, and clusterissuers
	//	log.Info().Msgf("found %d tls secrets to restore", len(secretsFilesToRestore))
	//	ssl.Restore(config.SSLBackupDir, k3d.DomainName, config.Kubeconfig)
	//} else {
	//	log.Info().Msg("no files found in secrets directory, continuing")
	//}

	err = ctrl.InstallArgoCD()
	if err != nil {
		return err
	}

	err = ctrl.InitializeArgoCD()
	if err != nil {
		return err
	}

	err = ctrl.DeployRegistryApplication()
	if err != nil {
		return err
	}

	err = ctrl.WaitForVault()
	if err != nil {
		return err
	}

	err = ctrl.InitializeVault()
	if err != nil {
		return err
	}

	//
	kcfg := k8s.CreateKubeConfig(false, ctrl.ProviderConfig.(k3d.K3dConfig).Kubeconfig)

	SetupMinioStorage(kcfg, ctrl.ProviderConfig.(k3d.K3dConfig).K1Dir, ctrl.GitProvider)

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
		return err
	}

	err = ctrl.RunUsersTerraform()
	if err != nil {
		return err
	}

	// PostRun string replacement
	err = k3d.PostRunPrepareGitopsRepository(ctrl.ClusterName,
		ctrl.ProviderConfig.(k3d.K3dConfig).GitopsDir,
		ctrl.CreateTokens("gitops").(*k3d.GitopsTokenValues),
	)
	if err != nil {
		log.Infof("Error detokenize post run: %s", err)
	}
	gitopsRepo, err := git.PlainOpen(ctrl.ProviderConfig.(k3d.K3dConfig).GitopsDir)
	if err != nil {
		log.Infof("error opening repo at: %s", ctrl.ProviderConfig.(k3d.K3dConfig).GitopsDir)
	}

	// check if file exists before rename
	_, err = os.Stat(
		fmt.Sprintf(
			"%s/terraform/%s/remote-backend.md",
			ctrl.ProviderConfig.(k3d.K3dConfig).GitopsDir,
			ctrl.ProviderConfig.(k3d.K3dConfig).GitProvider,
		),
	)
	if err == nil {
		err = os.Rename(
			fmt.Sprintf(
				"%s/terraform/%s/remote-backend.md",
				ctrl.ProviderConfig.(k3d.K3dConfig).GitopsDir,
				ctrl.GitProvider,
			), fmt.Sprintf(
				"%s/terraform/%s/remote-backend.tf",
				ctrl.ProviderConfig.(k3d.K3dConfig).GitopsDir,
				ctrl.GitProvider,
			))
		if err != nil {
			return err
		}
	}

	// Final gitops repo commit and push
	err = gitClient.Commit(gitopsRepo, "committing initial detokenized gitops-template repo content post run")
	if err != nil {
		return err
	}

	rec, err := ctrl.GetCurrentClusterRecord()
	if err != nil {
		return err
	}

	publicKeys, err := gitssh.NewPublicKeys("git", []byte(rec.PrivateKey), "")
	if err != nil {
		log.Infof("generate public keys failed: %s\n", err.Error())
	}
	err = gitopsRepo.Push(&git.PushOptions{
		RemoteName: ctrl.ProviderConfig.(k3d.K3dConfig).GitProvider,
		// This is currently broken because publickeys isn't stored
		Auth: publicKeys,
	})
	if err != nil {
		log.Infof("Error pushing repo: %s", err)
	}

	// Wait for console Deployment Pods to transition to Running
	consoleDeployment, err := k8s.ReturnDeploymentObject(
		kcfg.Clientset,
		"app.kubernetes.io/instance",
		"kubefirst-console",
		"kubefirst",
		600,
	)
	if err != nil {
		log.Errorf("Error finding console Deployment: %s", err)
		return err
	}
	_, err = k8s.WaitForDeploymentReady(kcfg.Clientset, consoleDeployment, 120)
	if err != nil {
		log.Errorf("Error waiting for console Deployment ready state: %s", err)
		return err
	}

	log.Info("cluster creation complete")

	// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricMgmtClusterInstallCompleted, "")

	// defer func(c segment.SegmentClient) {
	// 	err := c.Client.Close()
	// 	if err != nil {
	// 		log.Info().Msgf("error closing segment client %s", err.Error())
	// 	}
	// }(*segmentClient)

	return nil
}
