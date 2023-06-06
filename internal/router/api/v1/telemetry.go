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
// @Summary Create a Telemtry Event
// @Description Create a Telemtry Event
// @Tags cluster
// @Accept json
// @Param	cluster_name	path	string	true	"Cluster name"
// @Param	definition	body	types.ClusterDefinition	true	"Cluster create request in JSON format"
// @Success 202 {object} types.TelemetryRequest
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

	// Create
	// If create is in progress, return error
	// Retrieve cluster info
	cluster, err := db.Client.GetCluster(clusterName)

	if err != nil {
		log.Infof("cluster %s does not exist, continuing", clusterName)
	} else {
		var req types.TelemetryRequest
		err := c.Bind(&req)
		// Telemetry handler
		segmentClient, err := telemetryShim.SetupTelemetry(cluster)
		if err != nil {
			log.Fatal(err)
		}
		defer segmentClient.Client.Close()

		telemetryShim.Transmit(useTelemetry, segmentClient, req.Event, "")

	}

	c.JSON(http.StatusOK, true)
}
