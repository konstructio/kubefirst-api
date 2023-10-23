/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package main

import (
	"fmt"
	"os"

	"github.com/kubefirst/kubefirst-api/internal/environments"
	"github.com/kubefirst/kubefirst-api/internal/services"
	"github.com/kubefirst/metrics-client/pkg/telemetry"

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
		go func() {
			log.Infof("adding default environments for cluster %s", importedCluster.ClusterName)
			err := environments.CreateDefaultEnvironments(importedCluster)
			if err != nil {
				log.Infof("Error creating default environments %s", err.Error())
			}
		}()
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
	// segClient := segment.InitClient()
	// defer segClient.Client.Close()
	// machineID, err := machineid.ID()
	// if err != nil {
	// 	fmt.Println("machine id FAILED")
	// 	log.Info("machine id FAILED")
	// }
	event := telemetry.TelemetryEvent{
		CliVersion:        "development",
		CloudProvider:     os.Getenv("CLOUD_PROVIDER"),
		ClusterID:         os.Getenv("CLUSTER_ID"),
		ClusterType:       os.Getenv("CLUSTER_TYPE"),
		DomainName:        os.Getenv("DOMAIN_NAME"),
		GitProvider:       os.Getenv("GIT_PROVIDER"),
		InstallMethod:     os.Getenv("INSTALL_METHOD"),
		KubefirstClient:   "api",
		KubefirstTeam:     os.Getenv("KUBEFIRST_TEAM"),
		KubefirstTeamInfo: os.Getenv("KUBEFIRST_TEAM_INFO"),
		MachineID:         "4023E168-98FD-53C0-98FF-DC09FFC76F88",
		ErrorMessage:      "",
		UserId:            "4023E168-98FD-53C0-98FF-DC09FFC76F88",
		MetricName:        telemetry.ClusterInstallStarted,
	}

	// Subroutine to automatically update gitops catalog
	go utils.ScheduledGitopsCatalogUpdate()

	// Subroutine to emit heartbeat
	if useTelemetry {
		go apitelemetry.Heartbeat(event, db.Client)
	}

	// API
	r := api.SetupRouter()

	err = r.Run(fmt.Sprintf(":%v", port))
	if err != nil {
		log.Fatalf("Error starting API: %s", err)
	}
}
