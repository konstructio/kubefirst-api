/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package api

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/kubefirst/kubefirst-api/internal/db"
	"github.com/kubefirst/kubefirst-api/internal/middleware"
	router "github.com/kubefirst/kubefirst-api/internal/router/api/v1"
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

	// CORS
	r.Use(cors.New(cors.Config{
		AllowAllOrigins: true,
		AllowMethods:    []string{"DELETE", "GET", "HEAD", "PATCH", "POST", "PUT", "OPTIONS"},
		AllowHeaders:    []string{"origin", "content-Type"},
	}))

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
		v1.GET("/cluster", middleware.ValidateAPIKey(db.Client.UsersCollection), router.GetClusters)
		v1.POST("/cluster/import", middleware.ValidateAPIKey(db.Client.UsersCollection), router.PostImportCluster)

		v1.GET("/cluster/:cluster_name", middleware.ValidateAPIKey(db.Client.UsersCollection), router.GetCluster)
		v1.DELETE("/cluster/:cluster_name", middleware.ValidateAPIKey(db.Client.UsersCollection), router.DeleteCluster)
		v1.POST("/cluster/:cluster_name", middleware.ValidateAPIKey(db.Client.UsersCollection), router.PostCreateCluster)
		v1.POST("/cluster/:cluster_name/export", middleware.ValidateAPIKey(db.Client.UsersCollection), router.PostExportCluster)
		v1.POST("/cluster/:cluster_name/reset_progress", middleware.ValidateAPIKey(db.Client.UsersCollection), router.PostResetClusterProgress)

		// Gitops Catalog
		v1.GET("/gitops-catalog/apps", middleware.ValidateAPIKey(db.Client.UsersCollection), router.GetGitopsCatalogApps)
		v1.GET("/gitops-catalog/apps/update", middleware.ValidateAPIKey(db.Client.UsersCollection), router.UpdateGitopsCatalogApps)

		// Services
		v1.GET("/services/:cluster_name", middleware.ValidateAPIKey(db.Client.UsersCollection), router.GetServices)
		v1.POST("/services/:cluster_name/:service_name", middleware.ValidateAPIKey(db.Client.UsersCollection), router.PostAddServiceToCluster)
		v1.DELETE("/services/:cluster_name/:service_name", middleware.ValidateAPIKey(db.Client.UsersCollection), router.DeleteServiceFromCluster)

		// Domains
		v1.POST("/domain/:cloud_provider", middleware.ValidateAPIKey(db.Client.UsersCollection), router.PostDomains)
		v1.GET("/domain/validate/aws/:domain", middleware.ValidateAPIKey(db.Client.UsersCollection), router.GetValidateAWSDomain)
		v1.GET("/domain/validate/civo/:domain", middleware.ValidateAPIKey(db.Client.UsersCollection), router.GetValidateCivoDomain)

		// Regions
		v1.POST("/region/:cloud_provider", middleware.ValidateAPIKey(db.Client.UsersCollection), router.PostRegions)

		// Utilities
		v1.GET("/health", router.GetHealth)

		// Event streaming
		v1.GET("/stream", middleware.ValidateAPIKey(db.Client.UsersCollection), router.GetLogs)

		// Telemetry
		v1.POST("/telemetry/:cluster_name", middleware.ValidateAPIKey(db.Client.UsersCollection), router.PostTelemetry)
	}

	// swagger-ui
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return r
}
