/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package main

import (
	"fmt"

	"github.com/kubefirst/kubefirst-api/docs"
	"github.com/kubefirst/kubefirst-api/internal/db"
	"github.com/kubefirst/kubefirst-api/internal/env"
	"github.com/kubefirst/kubefirst-api/internal/environments"
	api "github.com/kubefirst/kubefirst-api/internal/router"
	"github.com/kubefirst/kubefirst-api/internal/services"
	apitelemetry "github.com/kubefirst/kubefirst-api/internal/telemetry"
	"github.com/kubefirst/kubefirst-api/internal/utils"
	"github.com/kubefirst/metrics-client/pkg/telemetry"

	log "github.com/sirupsen/logrus"
)

// @title Kubefirst API
// @version 1.0
// @description Kubefirst API
// @contact.name Kubefirst
// @contact.email help@kubefirst.io
// @host localhost:port
// @BasePath /api/v1

func main() {

	env, getEnvError := env.GetEnv()

	if getEnvError != nil {
		log.Fatal(getEnvError.Error())
	}

	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "",
	})
	log.SetReportCaller(false)

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
	docs.SwaggerInfo.Host = fmt.Sprintf("localhost:%v", env.ServerPort)
	docs.SwaggerInfo.BasePath = "/api/v1"
	docs.SwaggerInfo.Schemes = []string{"http"}

	// Telemetry handler
	telemetryEvent := telemetry.TelemetryEvent{
		CliVersion:        env.KubefirstVersion,
		CloudProvider:     env.CloudProvider,
		ClusterID:         env.ClusterId,
		ClusterType:       env.ClusterType,
		DomainName:        env.DomainName,
		ErrorMessage:      "",
		GitProvider:       env.GitProvider,
		InstallMethod:     env.InstallMethod,
		KubefirstClient:   "api",
		KubefirstTeam:     env.KubefirstTeam,
		KubefirstTeamInfo: env.KubefirstTeamInfo,
		MachineID:         env.ClusterId,
		MetricName:        telemetry.ClusterInstallStarted,
		UserId:            env.ClusterId,
	}

	// Subroutine to automatically update gitops catalog
	go utils.ScheduledGitopsCatalogUpdate()

	go apitelemetry.Heartbeat(telemetryEvent, db.Client)

	// API
	r := api.SetupRouter()

	err = r.Run(fmt.Sprintf(":%v", env.ServerPort))
	if err != nil {
		log.Fatalf("Error starting API: %s", err)
	}
}
