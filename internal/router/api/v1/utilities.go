/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package api

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/kubefirst/kubefirst-api/internal/db"
	"github.com/kubefirst/kubefirst-api/internal/types"
	log "github.com/sirupsen/logrus"
)

// getHealth godoc
// @Summary Return health status if the application is running.
// @Description Return health status if the application is running.
// @Tags health
// @Produce json
// @Success 200 {object} types.JSONHealthResponse
// @Router /health [get]
func GetHealth(c *gin.Context) {
	// Logging handler
	// Logs to stdout to maintain compatibility with event streaming
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "",
	})
	log.SetReportCaller(false)
	log.SetOutput(os.Stdout)

	// Verify database connectivity
	err := db.Client.TestDatabaseConnection()
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONHealthResponse{
			Status: "database connection failed",
		})
	}
	defer db.Client.Client.Disconnect(db.Client.Context)

	c.JSON(http.StatusOK, types.JSONHealthResponse{
		Status: "healthz",
	})
}
