/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	awsext "github.com/kubefirst/kubefirst-api/extensions/aws"
	"github.com/kubefirst/runtime/pkg"
	runtime "github.com/kubefirst/runtime/pkg"
	"github.com/kubefirst/runtime/pkg/k8s"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
)

func readKubefirstAPITokenFromSecret(clientset *kubernetes.Clientset) string {
	existingKubernetesSecret, err := k8s.ReadSecretV2(clientset, "kubefirst", "kubefirst-initial-secrets")
	if err != nil || existingKubernetesSecret == nil {
		log.Printf("Error reading existing Secret data: %s", err)
		return ""
	}
	return existingKubernetesSecret["K1_ACCESS_TOKEN"]
}

// ExportClusterRecord will export cluster record to mgmt cluster
// To be intiated by cluster 0
func (clctrl *ClusterController) ExportClusterRecord() error {
	cluster, err := clctrl.MdbCl.GetCluster(clctrl.ClusterName)
	if err != nil {
		log.Errorf("Error exporting cluster record: %s", err)
		clctrl.HandleError(err.Error())
		return err
	}

	cluster.Status = "provisioned"
	cluster.InProgress = false

	time.Sleep(time.Second * 10)

	
	apiURL := "http://localhost:8081" //referencing local port forwarded to api pod in kubernetes cluster

	var kubefirstSecret string
	if strings.Contains(apiURL, "localhost") {
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
		kubefirstSecret = readKubefirstAPITokenFromSecret(kcfg.Clientset)
	} else {
		kubefirstSecret = "feedkray"
	}
	err = runtime.IsAppAvailable(fmt.Sprintf("%s/api/v1/health", apiURL), "kubefirst api")
	if err != nil {
		log.Error("unable to start kubefirst api")

		clctrl.HandleError(err.Error())
		return err
	}

	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	httpClient := http.Client{Transport: customTransport}

	payload, err := json.Marshal(cluster)
	if err != nil {
		clctrl.HandleError(err.Error())
		return err
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/v1/cluster/import", apiURL), bytes.NewReader(payload))
	if err != nil {
		log.Errorf("error %s", err)
		clctrl.HandleError(err.Error())
		return err
	}
	req.Header.Add("Content-Type", pkg.JSONContentType)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", kubefirstSecret))

	res, err := httpClient.Do(req)
	if err != nil {
		log.Errorf("error %s", err)
		return err
	}

	if res.StatusCode != http.StatusOK {
		log.Errorf("unable to import cluster %s", res.Status)
		clctrl.HandleError(err.Error())
		return errors.New(fmt.Sprintf("unable to import cluster %s", res.Status))
	}

	_, err = io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	return nil
}
