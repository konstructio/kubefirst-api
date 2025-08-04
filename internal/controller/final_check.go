package controller

import (
	"fmt"

	awsext "github.com/konstructio/kubefirst-api/extensions/aws"
	"github.com/konstructio/kubefirst-api/internal/constants"
	"github.com/konstructio/kubefirst-api/internal/k8s"
	"github.com/konstructio/kubefirst-api/internal/secrets"
	"github.com/konstructio/kubefirst-api/internal/services"
	log "github.com/rs/zerolog/log"
)

func (clctrl *ClusterController) FinalCheck() error {
	if clctrl.CloudProvider == "aws" {
		// Regenerate EKS kubeconfig to get a fresh token, default only lasts 15 minutes
		log.Info().Msg("regenerating EKS kubeconfig to get a fresh token")
		kcfg, err := awsext.CreateEKSKubeconfig(&clctrl.AwsClient.Config, clctrl.ClusterName)
		if err != nil {
			return fmt.Errorf("failed to regenerate EKS kubeconfig: %w", err)
		}
		clctrl.Kcfg = kcfg
	}

	cluster, err := clctrl.GetCurrentClusterRecord()
	if err != nil {
		return fmt.Errorf("failed to get current cluster record: %w", err)
	}

	if !cluster.FinalCheck {
		// Wait for last sync wave app transition to Running
		log.Info().Msg("waiting for final sync wave Deployment to transition to Running")
		crossplaneDeployment, err := k8s.ReturnDeploymentObject(
			clctrl.Kcfg.Clientset,
			"app.kubernetes.io/instance",
			"crossplane",
			"crossplane-system",
			3600,
		)
		if err != nil {
			clctrl.UpdateClusterOnError(err.Error())
			return fmt.Errorf("error finding crossplane Deployment: %w", err)
		}

		log.Info().Msg("waiting on dns, tls certificates from letsencrypt and remaining sync waves.\n this may take up to 60 minutes but regularly completes in under 20 minutes")
		_, err = k8s.WaitForDeploymentReady(clctrl.Kcfg.Clientset, crossplaneDeployment, 3600)
		if err != nil {
			clctrl.UpdateClusterOnError(err.Error())
			return fmt.Errorf("error waiting for crossplane deployment to enter Ready state: %w", err)
		}

		// * export and import cluster
		if err := clctrl.ExportClusterRecord(); err != nil {
			log.Error().Msgf("Error exporting cluster record: %s", err)
			return fmt.Errorf("error exporting cluster record: %w", err)
		}
		clctrl.Cluster.Status = constants.ClusterStatusProvisioned
		clctrl.Cluster.InProgress = false

		if err := secrets.UpdateCluster(clctrl.KubernetesClient, clctrl.Cluster); err != nil {
			return fmt.Errorf("error updating cluster status: %w", err)
		}

		// Create default service entries
		cl, err := secrets.GetCluster(clctrl.KubernetesClient, clctrl.ClusterName)
		if err != nil {
			log.Error().Msgf("error getting cluster %s: %s", clctrl.ClusterName, err)
			clctrl.UpdateClusterOnError(err.Error())
			return fmt.Errorf("error getting cluster %s: %w", clctrl.ClusterName, err)
		}

		if err := services.AddDefaultServices(cl); err != nil {
			log.Error().Msgf("error adding default service entries for cluster %s: %s", cl.ClusterName, err)
			clctrl.UpdateClusterOnError(err.Error())
			return fmt.Errorf("error adding default service entries for cluster %s: %w", cl.ClusterName, err)
		}

		if clctrl.InstallKubefirstPro {
			log.Info().Msg("waiting for kubefirst-pro-api Deployment to transition to Running")
			kubefirstProAPI, err := k8s.ReturnDeploymentObject(
				clctrl.Kcfg.Clientset,
				"app.kubernetes.io/name",
				"kubefirst-pro-api",
				"kubefirst",
				1200,
			)
			if err != nil {
				clctrl.UpdateClusterOnError(err.Error())
				return fmt.Errorf("error finding kubefirst-pro-api Deployment: %w", err)
			}

			_, err = k8s.WaitForDeploymentReady(clctrl.Kcfg.Clientset, kubefirstProAPI, 300)
			if err != nil {
				clctrl.UpdateClusterOnError(err.Error())
				return fmt.Errorf("error waiting for kubefirst-pro-api deployment to enter Ready state: %w", err)
			}
		}

		// Wait for last sync wave app transition to Running
		log.Info().Msg("waiting for final sync wave Deployment to transition to Running")
		argocdDeployment, err := k8s.ReturnDeploymentObject(
			clctrl.Kcfg.Clientset,
			"app.kubernetes.io/name",
			"argocd-server",
			"argocd",
			3600,
		)
		if err != nil {
			clctrl.UpdateClusterOnError(err.Error())
			return fmt.Errorf("error finding argocd Deployment: %w", err)
		}
		_, err = k8s.WaitForDeploymentReady(clctrl.Kcfg.Clientset, argocdDeployment, 3600)
		if err != nil {
			clctrl.UpdateClusterOnError(err.Error())
			return fmt.Errorf("error waiting for argocd deployment to enter Ready state: %w", err)
		}

		clctrl.Cluster.FinalCheck = true
		err = secrets.UpdateCluster(clctrl.KubernetesClient, clctrl.Cluster)
		if err != nil {
			return fmt.Errorf("failed to update cluster with new status details: %w", err)
		}

		log.Info().Msg("cluster creation complete")
	}

	return nil
}
