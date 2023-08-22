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
	"time"

	"github.com/kubefirst/kubefirst-api/internal/db"
	"github.com/kubefirst/runtime/pkg"
	runtimetypes "github.com/kubefirst/runtime/pkg/types"
	log "github.com/sirupsen/logrus"
)

// ExportClusterRecord will export cluster record to mgmt cluster
func (clctrl *ClusterController) ExportClusterRecord() error {
	cluster, err := clctrl.MdbCl.GetCluster(clctrl.ClusterName)

	err = db.Client.Export(clctrl.ClusterName)
	if err != nil {
		log.Errorf("Error exporting cluster record: %s", err)
		return err
	}

	time.Sleep(time.Second * 10)

	consoleCloudUrl := fmt.Sprintf("https://kubefirst.%s", cluster.DomainName)

	err = pkg.IsAppAvailable(fmt.Sprintf("%s/api/proxyHealth", consoleCloudUrl), "kubefirst api")
	if err != nil {
		log.Error("unable to start kubefirst api")
	}

	importObject := runtimetypes.ImportClusterRequest{
		ClusterName:   cluster.ClusterName,
		CloudRegion:   cluster.CloudRegion,
		CloudProvider: cluster.CloudProvider,
	}
	importObject.StateStoreCredentials.AccessKeyID = cluster.StateStoreCredentials.AccessKeyID
	importObject.StateStoreCredentials.ID = cluster.StateStoreCredentials.ID
	importObject.StateStoreCredentials.Name = cluster.StateStoreCredentials.Name
	importObject.StateStoreCredentials.SecretAccessKey = cluster.StateStoreCredentials.SecretAccessKey
	importObject.StateStoreCredentials.SessionToken = cluster.StateStoreCredentials.SessionToken

	importObject.StateStoreDetails.Hostname = cluster.StateStoreDetails.Hostname
	importObject.StateStoreDetails.ID = cluster.StateStoreDetails.ID
	importObject.StateStoreDetails.Name = cluster.StateStoreDetails.Name

	requestObject := runtimetypes.ProxyImportRequest{
		Body: importObject,
		Url:  "/cluster/import",
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
		log.Errorf("error %s", err)
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")

	res, err := httpClient.Do(req)
	if err != nil {
		log.Errorf("error %s", err)
		return err
	}

	if res.StatusCode != http.StatusOK {
		log.Errorf("unable to import cluster %s", res.Status)
		return nil
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	log.Infof("Import: %s", string(body))

	return nil
}
