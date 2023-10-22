/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package api

import (
	"net/http"
	"os"

	"github.com/denisbrodbeck/machineid"
	"github.com/gin-gonic/gin"
	"github.com/kubefirst/kubefirst-api/internal/db"
	"github.com/kubefirst/kubefirst-api/internal/types"
	"github.com/kubefirst/kubefirst-api/pkg/metrics"
	"github.com/kubefirst/kubefirst-api/pkg/segment"
	"github.com/kubefirst/metrics-client/pkg/telemetry"
	"github.com/segmentio/analytics-go"
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
// @Param Authorization header string true "API key" default(Bearer <API key>)
// PostTelemetry sents a new telemetry event
func PostTelemetry(c *gin.Context) {

	// TODO
	// TODO THIS ROUTE NEEDS TO BE FIXED
	// TODO
	// TODO
	// Telemetry handler
	machineID, _ := machineid.ID()
	segClient := telemetry.SegmentClient{
		TelemetryEvent: telemetry.TelemetryEvent{
			CliVersion:        os.Getenv("KUBEFIRST_VERSION"),
			CloudProvider:     "cl.CloudProvider",
			ClusterID:         "cl.ClusterID",
			ClusterType:       "cl.ClusterType",
			DomainName:        "cl.DomainName",
			GitProvider:       "cl.GitProvider",
			InstallMethod:     "",
			KubefirstClient:   "api",
			KubefirstTeam:     os.Getenv("KUBEFIRST_TEAM"),
			KubefirstTeamInfo: os.Getenv("KUBEFIRST_TEAM_INFO"),
			MachineID:         machineID,
			ErrorMessage:      "",
			UserId:            machineID,
			MetricName:        metrics.ClusterDeleteStarted,
		},
		Client: analytics.New(segment.SegmentIOWriteKey),
	}
	defer segClient.Client.Close()
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

	// Retrieve cluster info
	_, err := db.Client.GetCluster(clusterName)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: "cluster not found",
		})
		return
	}

	var req types.TelemetryRequest
	err = c.Bind(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	telemetry.SendCountMetric(&segClient, req.Event, err.Error())

	c.JSON(http.StatusOK, true)
}
