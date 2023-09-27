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
	"time"

	runtime "github.com/kubefirst/runtime/pkg"

	pkgtypes "github.com/kubefirst/kubefirst-api/pkg/types"
	log "github.com/sirupsen/logrus"
)

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

	consoleCloudUrl := fmt.Sprintf("https://kubefirst.%s", cluster.DomainName)

	err = runtime.IsAppAvailable(fmt.Sprintf("%s/api/proxyHealth", consoleCloudUrl), "kubefirst api")
	if err != nil {
		log.Error("unable to start kubefirst api")

		clctrl.HandleError(err.Error())
		return err
	}

	requestObject := pkgtypes.ProxyImportRequest{
		Body: cluster,
		Url:  "/cluster/import",
	}

	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	httpClient := http.Client{Transport: customTransport}

	payload, err := json.Marshal(requestObject)
	if err != nil {
		clctrl.HandleError(err.Error())
		return err
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/proxy", consoleCloudUrl), bytes.NewReader(payload))
	if err != nil {
		log.Errorf("error %s", err)
		clctrl.HandleError(err.Error())
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
		clctrl.HandleError(err.Error())
		return errors.New(fmt.Sprintf("unable to import cluster %s", res.Status))
	}

	_, err = io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	return nil
}
