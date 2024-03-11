/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	awsext "github.com/kubefirst/kubefirst-api/extensions/aws"
	"github.com/kubefirst/runtime/pkg/k8s"
	log "github.com/rs/zerolog/log"
	v1secret "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		log.Error().Msgf("Error exporting cluster record: %s", err)
		clctrl.HandleError(err.Error())
		return err
	}

	cluster.Status = "provisioned"
	cluster.InProgress = false

	time.Sleep(time.Second * 10)

	var kcfg *k8s.KubernetesClient

	switch clctrl.CloudProvider {
	case "aws":
		kcfg = awsext.CreateEKSKubeconfig(&clctrl.AwsClient.Config, clctrl.ClusterName)
	case "akamai", "civo", "digitalocean", "k3s", "vultr":
		kcfg = k8s.CreateKubeConfig(false, clctrl.ProviderConfig.Kubeconfig)
	case "google":
		var err error
		kcfg, err = clctrl.GoogleClient.GetContainerClusterAuth(clctrl.ClusterName, []byte(clctrl.GoogleAuth.KeyFile))
		if err != nil {
			return err
		}
	}

	payload, err := json.Marshal(cluster)
	if err != nil {
		clctrl.HandleError(err.Error())
		return err
	}

	secret := &v1secret.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "mongodb-state", Namespace: "kubefirst"},
		Data: map[string][]byte{
			"cluster-0":    []byte(payload),
			"cluster-name": []byte(clctrl.ClusterName),
		},
	}

	err = k8s.CreateSecretV2(kcfg.Clientset, secret)

	if err != nil {
		clctrl.HandleError(err.Error())
		return errors.New(fmt.Sprintf("unable to save secret to management cluster. %s", err))
	}

	return nil
}
