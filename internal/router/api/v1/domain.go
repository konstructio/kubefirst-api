/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kubefirst/kubefirst-api/internal/db"
	"github.com/kubefirst/kubefirst-api/internal/types"
)

// GetDomains godoc
// @Summary Return a configured Kubefirst cluster
// @Description Return a configured Kubefirst cluster
// @Tags domain
// @Accept json
// @Produce json
// @Param	cloud_provider	path	string	true	"The cloud provider to return registered domains/zones from"
// @Success 200 {object} types.Cluster
// @Failure 400 {object} types.JSONFailureResponse
// @Router /domain/:cloud_provider [get]
// GetDomains returns registered domains/hosted zones for a cloud provider account
func GetDomains(c *gin.Context) {
	clusterName, param := c.Params.Get("cloud_provider")
	if !param {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":cloud_provider not provided",
		})
		return
	}

	// Retrieve domain info
	cluster, err := db.Client.GetCluster(clusterName)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: "cluster not found",
		})
		return
	}

	c.JSON(http.StatusOK, cluster)
}
