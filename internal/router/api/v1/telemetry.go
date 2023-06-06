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
	"github.com/kubefirst/kubefirst-api/internal/telemetryShim"
	"github.com/kubefirst/kubefirst-api/internal/types"
	log "github.com/sirupsen/logrus"
)

// PostTelemetry godoc
// @Summary Create a Telemetry Event
// @Description Create a Telemetry Event
// @Tags telemetry
// @Accept json
// @Param	cluster_name	path	string	true	"Cluster name"
// @Param	definition	body	types.TelemetryRequest	true	"event request in JSON format"
// @Success 202 {object} types.JSONSuccessResponse
// @Router /telemetry/:cluster_name [post]
// PostTelemetry sents a new telemetry event
func PostTelemetry(c *gin.Context) {
	useTelemetry := true
	if os.Getenv("USE_TELEMETRY") == "false" {
		useTelemetry = false
	}

	if !useTelemetry {
		c.JSON(http.StatusOK, "telemetry is not enabled")
		return
	}

	clusterName, param := c.Params.Get("cluster_name")
	if !param {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":cluster_name not provided",
		})
		return
	}

	cluster, err := db.Client.GetCluster(clusterName)
	var req types.TelemetryRequest
	err = c.Bind(&req)
	// Telemetry handler
	segmentClient, err := telemetryShim.SetupTelemetry(cluster)
	if err != nil {
		log.Fatal(err)
	}
	defer segmentClient.Client.Close()

	telemetryShim.Transmit(useTelemetry, segmentClient, req.Event, "")

	c.JSON(http.StatusOK, true)
}
