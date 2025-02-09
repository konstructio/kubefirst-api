/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package aws

import (
	"fmt"

	awsext "github.com/konstructio/kubefirst-api/extensions/aws"
	awsinternal "github.com/konstructio/kubefirst-api/internal/aws"
	"github.com/konstructio/kubefirst-api/internal/controller"
	"github.com/konstructio/kubefirst-api/internal/k8s"
	"github.com/konstructio/kubefirst-api/internal/secrets"
	pkgtypes "github.com/konstructio/kubefirst-api/pkg/types"
	log "github.com/rs/zerolog/log"
)

func CreateAWSCluster(definition *pkgtypes.ClusterDefinition) error {
	ctrl := controller.ClusterController{}

	if err := ctrl.InitController(definition); err != nil {
		return fmt.Errorf("error initializing controller: %w", err)
	}

	// Update cluster status in database
	ctrl.Cluster.InProgress = true
	if err := secrets.UpdateCluster(ctrl.KubernetesClient, ctrl.Cluster); err != nil {
		return fmt.Errorf("error updating cluster status: %w", err)
	}

	// Validate aws region
	conf, err := awsinternal.NewAwsV3(
		ctrl.CloudRegion,
		ctrl.AWSAuth.AccessKeyID,
		ctrl.AWSAuth.SecretAccessKey,
		ctrl.AWSAuth.SessionToken,
	)
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error creating aws client: %w", err)
	}

	awsClient := &awsinternal.Configuration{Config: conf}

	if _, err := awsClient.CheckAvailabilityZones(ctrl.CloudRegion); err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error checking availability zones: %w", err)
	}

	if err := ctrl.DownloadTools(ctrl.ProviderConfig.ToolsDir); err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error downloading tools: %w", err)
	}

	if err := ctrl.DomainLivenessTest(); err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error running domain liveness test: %w", err)
	}

	if err := ctrl.StateStoreCredentials(); err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error getting state store credentials: %w", err)
	}

	if err := ctrl.GitInit(); err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error initializing git: %w", err)
	}

	if err := ctrl.InitializeBot(); err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error initializing bot: %w", err)
	}

	// Where detokeinization happens
	if err := ctrl.RepositoryPrep(); err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error preparing repository: %w", err)
	}

	if err := ctrl.RunGitTerraform(); err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error running git terraform: %w", err)
	}

	if err := ctrl.RepositoryPush(); err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error pushing repository: %w", err)
	}

	if err := ctrl.CreateCluster(); err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error creating cluster: %w", err)
	}

	if err := ctrl.DetokenizeKMSKeyID(); err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error detokenizing KMS key ID: %w", err)
	}

	// Get Cluster kubeconfig and save to path so we can reference like everything else
	// TODO replace constant references to a new config with references to an existing config created here
	// for all cloud providers
	ctrl.Kcfg, err = awsext.CreateEKSKubeconfig(&ctrl.AwsClient.Config, ctrl.ClusterName)
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("failed to create eks config: %w", err)
	}
	if err := ctrl.WaitForClusterReady(); err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error waiting for cluster to be ready: %w", err)
	}

	// Cluster bootstrap (aws specific)
	// rec, err := ctrl.GetCurrentClusterRecord()
	// if err != nil {
	// 	ctrl.HandleError(err.Error())
	// 	return err
	// }

	if err := ctrl.InstallArgoCD(); err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error installing ArgoCD: %w", err)
	}

	if err := ctrl.InitializeArgoCD(); err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error initializing ArgoCD: %w", err)
	}

	// Needs wait after cluster create
	ctrl.Cluster.InProgress = true
	if err := secrets.UpdateCluster(ctrl.KubernetesClient, ctrl.Cluster); err != nil {
		return fmt.Errorf("error updating cluster status: %w", err)
	}

	if err := ctrl.ClusterSecretsBootstrap(); err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error bootstrapping cluster secrets: %w", err)
	}

	ctrl.Cluster.ClusterSecretsCreatedCheck = true
	if err := secrets.UpdateCluster(ctrl.KubernetesClient, ctrl.Cluster); err != nil {
		ctrl.Cluster.InProgress = false

		if err := secrets.UpdateCluster(ctrl.KubernetesClient, ctrl.Cluster); err != nil {
			return fmt.Errorf("error updating cluster status after cluster secrets were created (attempt 2): %w", err)
		}

		return fmt.Errorf("error updating cluster status after cluster secrets were created: %w", err)
	}

	if err := ctrl.DeployRegistryApplication(); err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error deploying registry application: %w", err)
	}

	if err := ctrl.WaitForVault(); err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error waiting for vault: %w", err)
	}

	// * configure vault with terraform
	// * vault port-forward
	vaultStopChannel := make(chan struct{}, 1)
	defer func() {
		close(vaultStopChannel)
	}()
	k8s.OpenPortForwardPodWrapper(
		ctrl.Kcfg.Clientset,
		ctrl.Kcfg.RestConfig,
		"vault-0",
		"vault",
		8200,
		8200,
		vaultStopChannel,
	)

	if err := ctrl.InitializeVault(); err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error initializing vault: %w", err)
	}

	if err := ctrl.RunVaultTerraform(); err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error running vault terraform: %w", err)
	}

	if err := ctrl.WriteVaultSecrets(); err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return fmt.Errorf("error writing vault secrets: %w", err)
	}

	if err := ctrl.RunUsersTerraform(); err != nil {
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
