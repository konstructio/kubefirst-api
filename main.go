/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package main

import (
	"fmt"
	"os"

	"github.com/kubefirst/kubefirst-api/internal/services"

	"github.com/joho/godotenv"
	"github.com/kubefirst/kubefirst-api/docs"
	"github.com/kubefirst/kubefirst-api/internal/db"
	api "github.com/kubefirst/kubefirst-api/internal/router"
	"github.com/kubefirst/kubefirst-api/internal/telemetryShim"
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
	for _, v := range []string{"MONGODB_HOST", "MONGODB_USERNAME", "MONGODB_PASSWORD", "CLOUD_PROVIDER"} {
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
	importedCluster, err := db.Client.ImportClusterIfEmpty(false, os.Getenv("CLOUD_PROVIDER"))
	if err != nil {
		log.Fatal(err)
	}

	if importedCluster.ClusterName != "" {
		services.AddDefaultServices(&importedCluster)
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
	segmentClient, err := telemetryShim.SetupInitialTelemetry(os.Getenv("CLUSTER_ID"), os.Getenv("CLUSTER_TYPE"), os.Getenv("INSTALL_METHOD"))
	if err != nil {
		log.Warn(err)
	}
	defer segmentClient.Client.Close()

	// Subroutine to automatically update gitops catalog
	go utils.ScheduledGitopsCatalogUpdate()

	// Subroutine to emit heartbeat
	if useTelemetry {
		go telemetryShim.Heartbeat(segmentClient)
	}

	// API
	r := api.SetupRouter()

	err = r.Run(fmt.Sprintf(":%v", port))
	if err != nil {
		log.Fatalf("Error starting API: %s", err)
	}
}
