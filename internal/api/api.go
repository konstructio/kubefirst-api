/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package api

import (
	"github.com/gin-gonic/gin"
	routes "github.com/kubefirst/kubefirst-api/internal/api/routes"
	log "github.com/sirupsen/logrus"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// SetupRouter instantiates the gin handler instance
func SetupRouter() *gin.Engine {
	// Release mode in production
	// Omit when developing for debug
	gin.SetMode(gin.ReleaseMode)
	log.Info("Starting kubefirst API...")
	r := gin.New()

	// Establish routes we don't want to log requests to
	r.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: []string{
			"/api/v1/health",
		},
	}))

	// Recovery middleware
	r.Use(gin.Recovery())

	// Define api/v1 group
	v1 := r.Group("api/v1")
	{
		// Cluster
		v1.GET("/cluster", routes.GetClusters)
		v1.GET("/cluster/:cluster_name", routes.GetCluster)
		v1.DELETE("/cluster/:cluster_name", routes.DeleteCluster)
		v1.POST("/cluster/:cluster_name", routes.PostCreateCluster)

		// AWS
		v1.GET("/aws/profiles", routes.GetAWSProfiles)
		v1.GET("/aws/validate/domain/:domain", routes.GetValidateAWSDomain)

		// Civo
		v1.GET("/civo/validate/domain/:domain", routes.GetValidateCivoDomain)

		// Utilities
		v1.GET("/health", routes.GetHealth)
	}

	// swagger-ui
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return r
}
