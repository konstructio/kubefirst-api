/*
Copyright © 2023 Kubefirst <kubefirst.io>
Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:
The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.
THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
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
