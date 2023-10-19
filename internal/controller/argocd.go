/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	"context"
	"fmt"
	"os"

	argocdapi "github.com/argoproj/argo-cd/v2/pkg/client/clientset/versioned"
	awsext "github.com/kubefirst/kubefirst-api/extensions/aws"
	"github.com/kubefirst/kubefirst-api/internal/telemetryShim"
	"github.com/kubefirst/runtime/pkg"
	"github.com/kubefirst/runtime/pkg/argocd"
	"github.com/kubefirst/runtime/pkg/k8s"
	"github.com/kubefirst/runtime/pkg/segment"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InstallArgoCD
func (clctrl *ClusterController) InstallArgoCD() error {
	// Logging handler
	// Logs to stdout to maintain compatibility with event streaming
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "",
	})
	log.SetReportCaller(false)
	log.SetOutput(os.Stdout)

	cl, err := clctrl.MdbCl.GetCluster(clctrl.ClusterName)
	if err != nil {
		return err
	}

	// Telemetry handler
	segmentClient, err := telemetryShim.SetupTelemetry(cl)
	if err != nil {
		return err
	}
	defer segmentClient.Client.Close()

	if !cl.ArgoCDInstallCheck {
		var kcfg *k8s.KubernetesClient

		switch clctrl.CloudProvider {
		case "aws":
			kcfg = awsext.CreateEKSKubeconfig(&clctrl.AwsClient.Config, clctrl.ClusterName)
		case "civo", "digitalocean", "vultr":
			kcfg = k8s.CreateKubeConfig(false, clctrl.ProviderConfig.Kubeconfig)
		case "google":
			var err error
			kcfg, err = clctrl.GoogleClient.GetContainerClusterAuth(clctrl.ClusterName, []byte(clctrl.GoogleAuth.KeyFile))
			if err != nil {
				return err
			}
		}

		argoCDInstallPath := fmt.Sprintf("github.com:kubefirst/manifests/argocd/cloud?ref=%s", pkg.KubefirstManifestRepoRef)
		log.Infof("installing argocd")

		err = argocd.ApplyArgoCDKustomize(kcfg.Clientset, argoCDInstallPath)
		if err != nil {
			telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricArgoCDInstallFailed, err.Error())
			return err
		}
		telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricArgoCDInstallCompleted, "")

		telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricArgoCDInstallStarted, "")
		telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricArgoCDInstallCompleted, "")

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
	// Logging handler
	// Logs to stdout to maintain compatibility with event streaming
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "",
	})
	log.SetReportCaller(false)
	log.SetOutput(os.Stdout)

	cl, err := clctrl.MdbCl.GetCluster(clctrl.ClusterName)
	if err != nil {
		return err
	}

	if !cl.ArgoCDInitializeCheck {
		var kcfg *k8s.KubernetesClient

		switch clctrl.CloudProvider {
		case "aws":
			kcfg = awsext.CreateEKSKubeconfig(&clctrl.AwsClient.Config, clctrl.ClusterName)
		case "civo", "digitalocean", "vultr":
			kcfg = k8s.CreateKubeConfig(false, clctrl.ProviderConfig.Kubeconfig)
		case "google":
			var err error
			kcfg, err = clctrl.GoogleClient.GetContainerClusterAuth(clctrl.ClusterName, []byte(clctrl.GoogleAuth.KeyFile))
			if err != nil {
				return err
			}
		}

		log.Info("Setting argocd username and password credentials")

		argocd.ArgocdSecretClient = kcfg.Clientset.CoreV1().Secrets("argocd")

		argocdPassword := k8s.GetSecretValue(argocd.ArgocdSecretClient, "argocd-initial-admin-secret", "password")
		if argocdPassword == "" {
			return fmt.Errorf("argocd password not found in secret")
		}

		log.Info("argocd username and password credentials set successfully")
		log.Info("getting an argocd auth token")

		var argoCDToken string

		switch clctrl.CloudProvider {
		case "aws", "civo", "google", "digitalocean", "vultr":

			// kcfg.Clientset.RbacV1().
			argoCDStopChannel := make(chan struct{}, 1)
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
			argoCDToken, err = argocd.GetArgoCDToken("admin", argocdPassword)
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

	// Telemetry handler
	segmentClient, err := telemetryShim.SetupTelemetry(cl)
	if err != nil {
		return err
	}
	defer segmentClient.Client.Close()

	if !cl.ArgoCDCreateRegistryCheck {
		var kcfg *k8s.KubernetesClient

		switch clctrl.CloudProvider {
		case "aws":
			kcfg = awsext.CreateEKSKubeconfig(&clctrl.AwsClient.Config, clctrl.ClusterName)
		case "civo", "digitalocean", "vultr":
			kcfg = k8s.CreateKubeConfig(false, clctrl.ProviderConfig.Kubeconfig)
		case "google":
			var err error
			kcfg, err = clctrl.GoogleClient.GetContainerClusterAuth(clctrl.ClusterName, []byte(clctrl.GoogleAuth.KeyFile))
			if err != nil {
				return err
			}
		}

		telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricCreateRegistryStarted, "")
		argocdClient, err := argocdapi.NewForConfig(kcfg.RestConfig)
		if err != nil {
			return err
		}

		log.Info("applying the registry application to argocd")

		registryURL, err := clctrl.GetRepoURL()
		if err != nil {
			return err
		}

		var registryPath string
		if clctrl.CloudProvider == "civo" && clctrl.GitProvider == "github" {
			registryPath = fmt.Sprintf("registry/clusters/%s", clctrl.ClusterName)
		} else if clctrl.CloudProvider == "civo" && clctrl.GitProvider == "gitlab" {
			registryPath = fmt.Sprintf("registry/clusters/%s", clctrl.ClusterName)
		} else if clctrl.CloudProvider == "aws" && clctrl.GitProvider == "github" {
			registryPath = fmt.Sprintf("registry/clusters/%s", clctrl.ClusterName)
		} else if clctrl.CloudProvider == "aws" && clctrl.GitProvider == "gitlab" {
			registryPath = fmt.Sprintf("registry/clusters/%s", clctrl.ClusterName)
		} else if clctrl.CloudProvider == "google" && clctrl.GitProvider == "github" {
			registryPath = fmt.Sprintf("registry/clusters/%s", clctrl.ClusterName)
		} else if clctrl.CloudProvider == "google" && clctrl.GitProvider == "gitlab" {
			registryPath = fmt.Sprintf("registry/clusters/%s", clctrl.ClusterName)
		} else if clctrl.CloudProvider == "digitalocean" && clctrl.GitProvider == "github" {
			registryPath = fmt.Sprintf("registry/clusters/%s", clctrl.ClusterName)
		} else if clctrl.CloudProvider == "digitalocean" && clctrl.GitProvider == "gitlab" {
			registryPath = fmt.Sprintf("registry/clusters/%s", clctrl.ClusterName)
		} else {
			registryPath = fmt.Sprintf("registry/%s", clctrl.ClusterName)
		}

		registryApplicationObject := argocd.GetArgoCDApplicationObject(
			registryURL,
			registryPath,
		)

		_, _ = argocdClient.ArgoprojV1alpha1().Applications("argocd").Create(context.Background(), registryApplicationObject, metav1.CreateOptions{})

		telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricCreateRegistryCompleted, "")

		err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "argocd_create_registry_check", true)
		if err != nil {
			return err
		}
	}

	return nil
}
