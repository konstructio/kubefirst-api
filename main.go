/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package main

import (
	"fmt"
	"os"

	"github.com/kubefirst/kubefirst-api/docs"
	api "github.com/kubefirst/kubefirst-api/internal/router"

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
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "",
	})
	log.SetReportCaller(false)
	log.SetOutput(os.Stdout)

	// Programmatically set swagger info
	docs.SwaggerInfo.Title = "Kubefirst API"
	docs.SwaggerInfo.Description = "Kubefirst API"
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.Host = fmt.Sprintf("localhost:%v", port)
	docs.SwaggerInfo.BasePath = "/api/v1"
	docs.SwaggerInfo.Schemes = []string{"http"}

	// API
	r := api.SetupRouter()
	err := r.Run(fmt.Sprintf(":%v", port))
	if err != nil {
		log.Fatalf("Error starting API: %s", err)
	}
}
