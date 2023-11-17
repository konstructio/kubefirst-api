/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package environments

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/kubefirst/kubefirst-api/internal/db"
	"github.com/kubefirst/kubefirst-api/internal/env"
	"github.com/kubefirst/kubefirst-api/pkg/types"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func NewEnvironment(envDef types.Environment) (types.Environment, error) {
	// Create new environment
	envDef.CreationTimestamp = fmt.Sprintf("%v", primitive.NewDateTimeFromTime(time.Now().UTC()))

	newEnv, err := db.Client.InsertEnvironment(envDef)

	return newEnv, err
}

func CreateDefaultEnvironments(mgmtCluster types.Cluster) error {

	// Logging handler
	// Logs to stdout to maintain compatibility with event streaming
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "",
	})
	log.SetReportCaller(false)
	log.SetOutput(os.Stdout)
	
	defaultClusterNames := []string{"development", "staging", "production"}

	defaultVclusterTemplate := types.WorkloadCluster{
		AdminEmail:    mgmtCluster.AlertsEmail,
		CloudProvider: mgmtCluster.CloudProvider,
		ClusterID:     mgmtCluster.ClusterID,
		ClusterName:   "not so empty string which should be replaced",
		ClusterType:   "workload-vcluster",
		CloudRegion:   mgmtCluster.CloudRegion,
		DomainName:    "not so empty string which should be replaced",
		DnsProvider:   mgmtCluster.DnsProvider,
		Environment: types.Environment{
			Name:        "not so empty string which should be replaced",
			Description: "not so empty string which should be replaced",
		},
		GitAuth:      mgmtCluster.GitAuth,
		InstanceSize: "", // left up to terraform
		NodeType:     "", //left up to terraform
		NodeCount:    3,  //defaulted here
	}
	
	defaultClusters := []types.WorkloadCluster{}

	for _, clusterName := range defaultClusterNames {
		vcluster := defaultVclusterTemplate
		vcluster.ClusterName = clusterName
		vcluster.Environment.Name = clusterName
		vcluster.DomainName = fmt.Sprintf("%s.%s", clusterName, mgmtCluster.DomainName)
		vcluster.Environment.Description = fmt.Sprintf("Default %s environment", clusterName)
		switch clusterName {
		case "development":
			vcluster.Environment.Color = "green"
		case "staging":
			vcluster.Environment.Color = "gold"
		case "production":
			vcluster.Environment.Color = "pink"
		}

		var err error
		vcluster.Environment, err = NewEnvironment(vcluster.Environment)
		if err != nil {
			log.Errorf("error creating default environment in db for env %s", err)
		}
		defaultClusters = append(defaultClusters, vcluster)
	}

	defaultEnvironmentSet := types.WorkloadClusterSet{
		Clusters: defaultClusters,
	}

	// call api-ee to create clusters
	return callApiEE(defaultEnvironmentSet)
}

func callApiEE(goPayload types.WorkloadClusterSet) error {
	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	httpClient := http.Client{Transport: customTransport}

	payload, err := json.Marshal(goPayload)
	if err != nil {
		return err
	}

	env, _ := env.GetEnv()

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/v1/environments/%s", env.EnterpriseApiUrl, env.ClusterId), bytes.NewReader(payload))

	if err != nil {
		log.Errorf("error creating http request %s", err)
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")

	res, err := httpClient.Do(req)
	timer := 0
	for err != nil {
		if timer > 12 {
			log.Errorf("error in http call to api ee: api url (%s) did not come up within 2 minutes %s", req.URL, err.Error())
		} else {
			res, err = httpClient.Do(req)
		}
		timer++
		time.Sleep(10 * time.Second)
	}

	if res.StatusCode != http.StatusAccepted {
		log.Errorf("unable to create default workload clusters and default environments %s: \n request: %s", res.Status, res.Request.URL)
		return err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	log.Info("Default environments initiatied", string(body))

	return nil
}
