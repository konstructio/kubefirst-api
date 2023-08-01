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
	"io"
	"net/http"
	"time"

	"github.com/kubefirst/kubefirst-api/internal/db"
	"github.com/kubefirst/kubefirst-api/internal/types"
	"github.com/kubefirst/runtime/pkg/k8s"
	log "github.com/sirupsen/logrus"
)

// ExportClusterRecord will export cluster record to mgmt cluster
func (clctrl *ClusterController) ExportClusterRecord() error {
	kcfg := k8s.CreateKubeConfig(false, clctrl.ProviderConfig.Kubeconfig)
	cluster, err := clctrl.MdbCl.GetCluster(clctrl.ClusterName)

	err = db.Client.Export(clctrl.ClusterName)
	if err != nil {
		log.Errorf("Error exporting cluster record: %s", err)
		return err
	}

	log.Println("Cluster exported:", clctrl.ClusterName)

	//* kubefirst api port-forward
	kubefirstApiStopChannel := make(chan struct{}, 1)
	defer func() {
		close(kubefirstApiStopChannel)
	}()
	k8s.OpenPortForwardPodWrapper(
		kcfg.Clientset,
		kcfg.RestConfig,
		"kubefirst-console-kubefirst-api",
		"kubefirst",
		8081,
		8085,
		kubefirstApiStopChannel,
	)

	time.Sleep(time.Second * 20)

	importUrl := "http://localhost:8085/api/v1/cluster/import"

	importObject := types.ImportClusterRequest{
		ClusterName:           clctrl.ClusterName,
		CloudRegion:           clctrl.CloudRegion,
		CloudProvider:         clctrl.CloudProvider,
		StateStoreCredentials: cluster.StateStoreCredentials,
		StateStoreDetails:     cluster.StateStoreDetails,
	}

	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	httpClient := http.Client{Transport: customTransport}

	payload, err := json.Marshal(importObject)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, importUrl, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")

	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		log.Println("unable to import cluster", res.StatusCode)
		return nil
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	log.Println("Import:", string(body))

	return nil
}
