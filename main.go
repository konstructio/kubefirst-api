/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/kubefirst/kubefirst-api/internal/services"
	"github.com/kubefirst/kubefirst-api/pkg/segment"
	pkgtypes "github.com/kubefirst/kubefirst-api/pkg/types"
	"github.com/kubefirst/runtime/pkg"

	"github.com/joho/godotenv"
	"github.com/kubefirst/kubefirst-api/docs"
	"github.com/kubefirst/kubefirst-api/internal/db"
	api "github.com/kubefirst/kubefirst-api/internal/router"
	apitelemetry "github.com/kubefirst/kubefirst-api/internal/telemetry"
	"github.com/kubefirst/kubefirst-api/internal/utils"

	log "github.com/sirupsen/logrus"
)

// @title Kubefirst API
// @version 1.0
// @description Kubefirst API
// @contact.name Kubefirst
// @contact.email help@kubefirst.io
// @host localhost:port
// @BasePath /api/v1

const (
	port int = 8081
)

func main() {

	envError := godotenv.Load(".env")

	if envError != nil {
		log.Info("error loading .env file, using local environment variables")
	}

	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "",
	})
	log.SetReportCaller(false)

	// Check for required environment variables
	if os.Getenv("MONGODB_HOST_TYPE") == "" {
		log.Fatalf("the MONGODB_HOST_TYPE environment variable must be set to either: atlas, local")
	}
	for _, v := range []string{"MONGODB_HOST", "MONGODB_USERNAME", "MONGODB_PASSWORD"} {
		if os.Getenv(v) == "" {
			log.Fatalf("the %s environment variable must be set", v)
		}
	}

	useTelemetry := true
	if os.Getenv("USE_TELEMETRY") == "false" {
		useTelemetry = false
	} else {
		for _, v := range []string{"CLUSTER_ID", "CLUSTER_TYPE", "INSTALL_METHOD"} {
			if os.Getenv(v) == "" {
				log.Fatalf("the %s environment variable must be set", v)
			}
		}
	}

	// Verify database connectivity
	err := db.Client.EstablishMongoConnection(db.EstablishConnectArgs{
		Tries:  20,
		Silent: false,
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Infof("checking for cluster import secret for management cluster")
	// Import if needed
	importedCluster, err := db.Client.ImportClusterIfEmpty(false)
	if err != nil {
		log.Fatal(err)
	}

	if importedCluster.ClusterName != "" {
		log.Infof("adding default services for cluster %s", importedCluster.ClusterName)
		services.AddDefaultServices(&importedCluster)

		// Call default environment create code if we imported  a cluster
		// execute default environment creation concurrently
		// go func() {
		// 	log.Infof("adding default environments for cluster %s", importedCluster.ClusterName)
		// 	err := environments.CreateDefaultEnvironments(importedCluster)
		// 	if err != nil {
		// 		log.Infof("Error creating default environments %s", err.Error())
		// 	}
		// }()
		arrayOne := [3]string{"development", "staging", "production"}

		for _, env := range arrayOne {

			log.Infoln("creating cluster", env)
            // - name: CLOUD_PROVIDER
            // - name: CLUSTER_ID
            // - name: CLUSTER_TYPE
            // - name: DOMAIN_NAME
            // - name: GIT_PROVIDER
            // - name: INSTALL_METHOD
            // - name: KUBEFIRST_CLIENT
            // - name: KUBEFIRST_TEAM
            // - name: KUBEFIRST_TEAM_INFO
            // - name: KUBEFIRST_VERSION
            // - name: USE_TELEMETRY
              
			var developmentCluster = pkgtypes.WorkloadCluster{
				AdminEmail:    "alerts@kubefirst.io", //todo
				CloudProvider: os.Getenv("CLOUD_PROVIDER"),
				ClusterID:     "",
				ClusterName:   env,
				ClusterType:   "workload-vcluster",
				CloudRegion:   "fra1", //todo
				DomainName:    fmt.Sprintf("%s.%s", env, os.Getenv("DOMAIN_NAME")),
				DnsProvider:   "civo", //todo
				Environment: pkgtypes.Environment{
					ID:          [12]byte{},
					Name:        env,
					Color:       "fucia", //todo
					Description: "pretty",
				},
				GitAuth: pkgtypes.GitAuth{
					Token:      "",
					User:       "",
					Owner:      "",
					PublicKey:  "",
					PrivateKey: "",
					PublicKeys: "",
				},
				InstanceSize: "medium",
				MachineType:  "",
				NodeCount:    1,
				Status:       "",
			}
			postVcluster(developmentCluster, os.Getenv("CLUSTER_ID"))
		}

	}
	defer db.Client.Client.Disconnect(db.Client.Context)

	// Programmatically set swagger info
	docs.SwaggerInfo.Title = "Kubefirst API"
	docs.SwaggerInfo.Description = "Kubefirst API"
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.Host = fmt.Sprintf("localhost:%v", port)
	docs.SwaggerInfo.BasePath = "/api/v1"
	docs.SwaggerInfo.Schemes = []string{"http"}

	// Telemetry handler
	segClient := segment.InitClient()
	defer segClient.Client.Close()

	// Subroutine to automatically update gitops catalog
	go utils.ScheduledGitopsCatalogUpdate()

	// Subroutine to emit heartbeat
	if useTelemetry {
		go apitelemetry.Heartbeat(segClient, db.Client)
	}

	// API
	r := api.SetupRouter()

	err = r.Run(fmt.Sprintf(":%v", port))
	if err != nil {
		log.Fatalf("Error starting API: %s", err)
	}
}

func postVcluster(workloadClusterDef pkgtypes.WorkloadCluster, mgmtClusterID string) (string, error) {

	payload, err := json.Marshal(&workloadClusterDef)
	if err != nil {
		return "", err
	}

	clusterApi := fmt.Sprintf("http://kubefirst-api-ee.kubefirst.svc.cluster.local:8080/cluster/%s", mgmtClusterID)

	req, err := http.NewRequest(http.MethodPost, clusterApi, bytes.NewBuffer(payload))
	if err != nil {
		log.Infof("error setting request")
	}

	k1AccessToken := os.Getenv("")
	req.Header.Add("Content-Type", pkg.JSONContentType)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", k1AccessToken))

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	log.Infof(string(body))

	return "yay", nil
}
