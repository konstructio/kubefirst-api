/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	"context"
	"fmt"
	"strings"

	argocdapi "github.com/argoproj/argo-cd/v2/pkg/client/clientset/versioned"
	"github.com/kubefirst/runtime/pkg"
	"github.com/kubefirst/runtime/pkg/argocd"
	"github.com/kubefirst/runtime/pkg/helpers"
	"github.com/kubefirst/runtime/pkg/k3d"
	"github.com/kubefirst/runtime/pkg/k8s"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InstallArgoCD
func (clctrl *ClusterController) InstallArgoCD() error {
	cl, err := clctrl.MdbCl.GetCluster(clctrl.ClusterName)
	if err != nil {
		return err
	}

	if !cl.ArgoCDInstallCheck {
		kcfg := k8s.CreateKubeConfig(false, clctrl.ProviderConfig.Kubeconfig)

		argoCDInstallPath := fmt.Sprintf("github.com:kubefirst/manifests/argocd/k3d?ref=%s", pkg.KubefirstManifestRepoRef)

		// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricArgoCDInstallStarted, "")

		log.Infof("installing argocd")

		// Build and apply manifests
		yamlData, err := kcfg.KustomizeBuild(argoCDInstallPath)
		if err != nil {
			return err
		}
		output, err := kcfg.SplitYAMLFile(yamlData)
		if err != nil {
			return err
		}
		err = kcfg.ApplyObjects("", output)
		if err != nil {
			// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricArgoCDInstallFailed, err.Error())
			return err
		}

		// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricArgoCDInstallCompleted, "")

		// Wait for ArgoCD to be ready
		_, err = k8s.VerifyArgoCDReadiness(kcfg.Clientset, true, 300)
		if err != nil {
			log.Errorf("error waiting for ArgoCD to become ready: %s", err)
			return err
		}

		err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "argocd_install_check", true)
		if err != nil {
			return err
		}
	}

	return nil
}

// InitializeArgoCD
func (clctrl *ClusterController) InitializeArgoCD() error {
	cl, err := clctrl.MdbCl.GetCluster(clctrl.ClusterName)
	if err != nil {
		return err
	}

	if !cl.ArgoCDInitializeCheck {
		kcfg := k8s.CreateKubeConfig(false, clctrl.ProviderConfig.Kubeconfig)

		log.Info("Setting argocd username and password credentials")

		argocd.ArgocdSecretClient = kcfg.Clientset.CoreV1().Secrets("argocd")

		argocdPassword := k8s.GetSecretValue(argocd.ArgocdSecretClient, "argocd-initial-admin-secret", "password")
		if argocdPassword == "" {
			return fmt.Errorf("argocd password not found in secret")
		}

		log.Info("argocd username and password credentials set successfully")
		log.Info("Getting an argocd auth token")

		// Test https to argocd
		var argoCDToken string
		// only the host, not the protocol
		err := helpers.TestEndpointTLS(strings.Replace(k3d.ArgocdURL, "https://", "", 1))
		if err != nil {
			argoCDStopChannel := make(chan struct{}, 1)
			log.Infof("argocd not available via https, using http")
			defer func() {
				close(argoCDStopChannel)
			}()
			k8s.OpenPortForwardPodWrapper(
				kcfg.Clientset,
				kcfg.RestConfig,
				"argocd-server",
				"argocd",
				8080,
				8080,
				argoCDStopChannel,
			)
			argoCDHTTPURL := strings.Replace(
				k3d.ArgocdURL,
				"https://",
				"http://",
				1,
			) + ":8080"
			argoCDToken, err = argocd.GetArgocdTokenV2(clctrl.HttpClient, argoCDHTTPURL, "admin", argocdPassword)
			if err != nil {
				return err
			}
		} else {
			argoCDToken, err = argocd.GetArgocdTokenV2(clctrl.HttpClient, k3d.ArgocdURL, "admin", argocdPassword)
			if err != nil {
				return err
			}
		}

		log.Info("argocd admin auth token set")

		err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "argocd_password", argocdPassword)
		if err != nil {
			return err
		}
		err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "argocd_auth_token", argoCDToken)
		if err != nil {
			return err
		}
		err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "argocd_initialize_check", true)
		if err != nil {
			return err
		}
	}

	return nil
}

// DeployRegistryApplication
func (clctrl *ClusterController) DeployRegistryApplication() error {
	cl, err := clctrl.MdbCl.GetCluster(clctrl.ClusterName)
	if err != nil {
		return err
	}

	if !cl.ArgoCDCreateRegistryCheck {
		kcfg := k8s.CreateKubeConfig(false, clctrl.ProviderConfig.Kubeconfig)

		// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricCreateRegistryStarted, "")
		argocdClient, err := argocdapi.NewForConfig(kcfg.RestConfig)
		if err != nil {
			return err
		}

		log.Info("applying the registry application to argocd")
		registryApplicationObject := argocd.GetArgoCDApplicationObject(clctrl.ProviderConfig.DestinationGitopsRepoGitURL, fmt.Sprintf("registry/%s", clctrl.ClusterName))
		_, _ = argocdClient.ArgoprojV1alpha1().Applications("argocd").Create(context.Background(), registryApplicationObject, metav1.CreateOptions{})

		// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricCreateRegistryCompleted, "")

		err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "argocd_create_registry_check", true)
		if err != nil {
			return err
		}
	}

	return nil
}
