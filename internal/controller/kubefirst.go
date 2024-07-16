/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	awsext "github.com/kubefirst/kubefirst-api/extensions/aws"
	pkg "github.com/kubefirst/kubefirst-api/internal"
	"github.com/kubefirst/kubefirst-api/internal/k8s"
	"github.com/kubefirst/kubefirst-api/internal/secrets"
	"github.com/kubefirst/kubefirst-api/pkg/types"
	log "github.com/rs/zerolog/log"
	v1secret "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func ReadKubefirstAPITokenFromSecret(clientset *kubernetes.Clientset) string {
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
	cluster, err := secrets.GetCluster(clctrl.KubernetesClient, clctrl.ClusterName)

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

	bytes, err := json.Marshal(cluster)
	if err != nil {
		clctrl.HandleError(err.Error())
		return err
	}

	secretValuesMap, _ := secrets.ParseJSONToMap(string(bytes))

	secret := &v1secret.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "kubefirst-initial-state", Namespace: "kubefirst"},
		Data:       secretValuesMap,
	}

	err = k8s.CreateSecretV2(kcfg.Clientset, secret)

	if err != nil {
		clctrl.HandleError(err.Error())
		return fmt.Errorf("unable to save secret to management cluster. %s", err)
	}

	return nil
}

// ExportClusterRecord will export cluster record to mgmt cluster
func (clctrl *ClusterController) CreateVirtualClusters() error {
	time.Sleep(time.Minute * 2)
	var fullDomainName string

	if clctrl.SubdomainName != "" {
		fullDomainName = fmt.Sprintf("%s.%s", clctrl.SubdomainName, clctrl.DomainName)
	} else {
		fullDomainName = clctrl.DomainName
	}

	consoleCloudUrl := fmt.Sprintf("https://kubefirst.%s", fullDomainName)

	if strings.ToLower(os.Getenv("K1_LOCAL_DEBUG")) == "true" { //allow using local console running on port 3000
		consoleCloudUrl = "http://localhost:3000"
	}

	err := pkg.IsAppAvailable(fmt.Sprintf("%s/api/proxyHealth", consoleCloudUrl), "kubefirst api")
	if err != nil {
		log.Error().Msgf("unable to wait for kubefirst console: %s", err)
		clctrl.HandleError(err.Error())
		return err
	}

	requestObject := types.ProxyRequest{
		Url: fmt.Sprintf("/cluster/%s/vclusters", clctrl.ClusterName),
	}

	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	httpClient := http.Client{Transport: customTransport}

	payload, err := json.Marshal(requestObject)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/proxy", consoleCloudUrl), bytes.NewReader(payload))
	if err != nil {
		log.Error().Msgf("unable to create default clusters: %s", err)
		clctrl.HandleError(err.Error())
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")

	res, err := httpClient.Do(req)
	if err != nil {
		log.Error().Msgf("unable to create default clusters: %s", err)
		clctrl.HandleError(err.Error())
		return err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		log.Error().Msgf("unable to create default clusters: %s %s", err, body)
		clctrl.HandleError(err.Error())
		return err
	}

	log.Info().Msg("cluster creation complete")

	return nil
}
