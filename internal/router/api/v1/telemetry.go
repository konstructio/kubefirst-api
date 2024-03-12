/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kubefirst/kubefirst-api/internal/constants"
	"github.com/kubefirst/kubefirst-api/internal/env"
	"github.com/kubefirst/kubefirst-api/internal/secrets"
	"github.com/kubefirst/kubefirst-api/internal/types"
	"github.com/kubefirst/kubefirst-api/internal/utils"
	"github.com/kubefirst/metrics-client/pkg/telemetry"
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
	clusterName, param := c.Params.Get("cluster_name")
	if !param {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":cluster_name not provided",
		})
		return
	}
	kcfg := utils.GetKubernetesClient(clusterName)

	// Retrieve cluster info
	cl, err := secrets.GetCluster(kcfg.Clientset, clusterName)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: "cluster not found",
		})
		return
	}

	env, _ := env.GetEnv(constants.SilenceGetEnv)

	telEvent := telemetry.TelemetryEvent{
		CliVersion:        env.KubefirstVersion,
		CloudProvider:     cl.CloudProvider,
		ClusterID:         cl.ClusterID,
		ClusterType:       cl.ClusterType,
		DomainName:        cl.DomainName,
		GitProvider:       cl.GitProvider,
		InstallMethod:     "",
		KubefirstClient:   "api",
		KubefirstTeam:     env.KubefirstTeam,
		KubefirstTeamInfo: env.KubefirstTeamInfo,
		MachineID:         cl.DomainName,
		ErrorMessage:      "",
		UserId:            cl.DomainName,
		MetricName:        "",
	}

	var req types.TelemetryRequest
	err = c.Bind(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	telemetry.SendEvent(telEvent, req.Event, "")

	c.JSON(http.StatusOK, true)
}
