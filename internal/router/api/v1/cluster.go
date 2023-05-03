/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package api

import (
	"fmt"

	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kubefirst/kubefirst-api/internal/db"
	"github.com/kubefirst/kubefirst-api/internal/types"
	"github.com/kubefirst/kubefirst-api/providers/aws"
	"github.com/kubefirst/kubefirst-api/providers/civo"
	"github.com/kubefirst/kubefirst-api/providers/digitalocean"
	"github.com/kubefirst/kubefirst-api/providers/k3d"
	"github.com/kubefirst/kubefirst-api/providers/vultr"
	log "github.com/sirupsen/logrus"
)

// DeleteCluster godoc
// @Summary Delete a Kubefirst cluster
// @Description Delete a Kubefirst cluster
// @Tags cluster
// @Accept json
// @Produce json
// @Param	cluster_name	path	string	true	"Cluster name"
// @Success 202 {object} types.JSONSuccessResponse
// @Failure 400 {object} types.JSONFailureResponse
// @Router /cluster/:cluster_name [delete]
// DeleteCluster handles a request to delete a cluster
func DeleteCluster(c *gin.Context) {
	clusterName, param := c.Params.Get("cluster_name")
	if !param {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":cluster_name not provided",
		})
		return
	}

	// Delete cluster
	rec, err := db.Client.GetCluster(clusterName)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	switch rec.CloudProvider {
	case "aws":
		go func() {
			err := aws.DeleteAWSCluster(&rec)
			if err != nil {
				log.Errorf(err.Error())
			}
		}()

		c.JSON(http.StatusAccepted, types.JSONSuccessResponse{
			Message: "cluster delete enqueued",
		})
	case "civo":
		go func() {
			err := civo.DeleteCivoCluster(&rec)
			if err != nil {
				log.Errorf(err.Error())
			}
		}()

		c.JSON(http.StatusAccepted, types.JSONSuccessResponse{
			Message: "cluster delete enqueued",
		})
	case "digitalocean":
		go func() {
			err := digitalocean.DeleteDigitaloceanCluster(&rec)
			if err != nil {
				log.Errorf(err.Error())
			}
		}()

		c.JSON(http.StatusAccepted, types.JSONSuccessResponse{
			Message: "cluster delete enqueued",
		})
	case "k3d":
	case "vultr":
		go func() {
			err := vultr.DeleteVultrCluster(&rec)
			if err != nil {
				log.Errorf(err.Error())
			}
		}()

		c.JSON(http.StatusAccepted, types.JSONSuccessResponse{
			Message: "cluster delete enqueued",
		})
	}
}

// GetCluster godoc
// @Summary Return a configured Kubefirst cluster
// @Description Return a configured Kubefirst cluster
// @Tags cluster
// @Accept json
// @Produce json
// @Param	cluster_name	path	string	true	"Cluster name"
// @Success 200 {object} types.Cluster
// @Failure 400 {object} types.JSONFailureResponse
// @Router /cluster/:cluster_name [get]
// GetCluster returns a specific configured cluster
func GetCluster(c *gin.Context) {
	clusterName, param := c.Params.Get("cluster_name")
	if !param {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":cluster_name not provided",
		})
		return
	}

	// Retrieve cluster info
	cluster, err := db.Client.GetCluster(clusterName)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: "cluster not found",
		})
		return
	}

	c.JSON(http.StatusOK, cluster)
}

// GetClusters godoc
// @Summary Return all known configured Kubefirst clusters
// @Description Return all known configured Kubefirst clusters
// @Tags cluster
// @Accept json
// @Produce json
// @Success 200 {object} []types.Cluster
// @Failure 400 {object} types.JSONFailureResponse
// @Router /cluster [get]
// GetClusters returns all known configured clusters
func GetClusters(c *gin.Context) {
	// Retrieve all clusters info
	allClusters, err := db.Client.GetClusters()
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: fmt.Sprintf("%s", err),
		})
		return
	}

	c.JSON(http.StatusOK, allClusters)
}

// PostCreateCluster godoc
// @Summary Create a Kubefirst cluster
// @Description Create a Kubefirst cluster
// @Tags cluster
// @Accept json
// @Produce json
// @Param	cluster_name	path	string	true	"Cluster name"
// @Param	definition	body	types.ClusterDefinition	true	"Cluster create request in JSON format"
// @Success 202 {object} types.JSONSuccessResponse
// @Failure 400 {object} types.JSONFailureResponse
// @Router /cluster/:cluster_name [post]
// PostCreateCluster handles a request to create a cluster
func PostCreateCluster(c *gin.Context) {
	clusterName, param := c.Params.Get("cluster_name")
	if !param {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":cluster_name not provided",
		})
		return
	}

	// Bind to variable as application/json, handle error
	var clusterDefinition types.ClusterDefinition
	err := c.Bind(&clusterDefinition)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}
	clusterDefinition.ClusterName = clusterName

	// Create
	// If create is in progress, return error
	// Retrieve cluster info
	cluster, err := db.Client.GetCluster(clusterName)
	if err != nil {
		log.Infof("cluster %s does not exist, continuing", clusterName)
	} else {
		if cluster.InProgress {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: fmt.Sprintf("%s has an active process running and another create cannot be enqeued", clusterName),
			})
			return
		}
	}

	switch clusterDefinition.CloudProvider {
	case "aws":
		go func() {
			err = aws.CreateAWSCluster(&clusterDefinition)
			if err != nil {
				log.Errorf(err.Error())
			}
		}()

		c.JSON(http.StatusAccepted, types.JSONSuccessResponse{
			Message: "cluster create enqueued",
		})
	case "civo":
		go func() {
			err = civo.CreateCivoCluster(&clusterDefinition)
			if err != nil {
				log.Errorf(err.Error())
			}
		}()

		c.JSON(http.StatusAccepted, types.JSONSuccessResponse{
			Message: "cluster create enqueued",
		})
	case "digitalocean":
		go func() {
			err = digitalocean.CreateDigitaloceanCluster(&clusterDefinition)
			if err != nil {
				log.Errorf(err.Error())
			}
		}()

		c.JSON(http.StatusAccepted, types.JSONSuccessResponse{
			Message: "cluster create enqueued",
		})
	case "k3d":
		go func() {
			err = k3d.CreateK3DCluster(&clusterDefinition)
			if err != nil {
				log.Errorf(err.Error())
			}
		}()

		c.JSON(http.StatusAccepted, types.JSONSuccessResponse{
			Message: "cluster create enqueued",
		})
	case "vultr":
		go func() {
			err = vultr.CreateVultrCluster(&clusterDefinition)
			if err != nil {
				log.Errorf(err.Error())
			}
		}()

		c.JSON(http.StatusAccepted, types.JSONSuccessResponse{
			Message: "cluster create enqueued",
		})
	}
}

// PostExportCluster godoc
// @Summary Export a Kubefirst cluster database entry
// @Description Export a Kubefirst cluster database entry
// @Tags cluster
// @Accept json
// @Produce json
// @Param	cluster_name	path	string	true	"Cluster name"
// @Success 202 {object} types.JSONSuccessResponse
// @Failure 400 {object} types.JSONFailureResponse
// @Router /cluster/:cluster_name/export [post]
// PostExportCluster handles a request to export a cluster
func PostExportCluster(c *gin.Context) {
	clusterName, param := c.Params.Get("cluster_name")
	if !param {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":cluster_name not provided",
		})
		return
	}

	// Export
	err := db.Client.Export(clusterName)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: fmt.Sprintf("error exporting cluster %s: %s", clusterName, err),
		})
		return
	}

	c.JSON(http.StatusOK, types.JSONSuccessResponse{
		Message: "cluster exported",
	})
}

// PostImportCluster godoc
// @Summary Import a Kubefirst cluster database entry
// @Description Import a Kubefirst cluster database entry
// @Tags cluster
// @Accept json
// @Produce json
// @Param	request_body	body	types.ImportClusterRequest	true	"Cluster import request in JSON format"
// @Success 202 {object} types.JSONSuccessResponse
// @Failure 400 {object} types.JSONFailureResponse
// @Router /cluster/import [post]
// PostImportCluster handles a request to import a cluster
func PostImportCluster(c *gin.Context) {
	// Bind to variable as application/json, handle error
	var req types.ImportClusterRequest
	err := c.Bind(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	// Export
	err = db.Client.Restore(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: fmt.Sprintf("error importing cluster %s: %s", req.ClusterName, err),
		})
		return
	}

	c.JSON(http.StatusOK, types.JSONSuccessResponse{
		Message: "cluster imported",
	})
}

// PostResetClusterProgress godoc
// @Summary Remove a cluster progress marker from a cluster entry
// @Description Remove a cluster progress marker from a cluster entry
// @Tags cluster
// @Accept json
// @Produce json
// @Param	cluster_name	path	string	true	"Cluster name"
// @Success 202 {object} types.JSONSuccessResponse
// @Failure 400 {object} types.JSONFailureResponse
// @Router /cluster/:cluster_name/reset_progress [post]
// PostResetClusterProgress removes a cluster progress marker from a cluster entry
func PostResetClusterProgress(c *gin.Context) {
	clusterName, param := c.Params.Get("cluster_name")
	if !param {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":cluster_name not provided",
		})
		return
	}

	// Reset
	err := db.Client.UpdateCluster(clusterName, "in_progress", false)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: fmt.Sprintf("error updating cluster %s: %s", clusterName, err),
		})
		return
	}

	c.JSON(http.StatusOK, types.JSONSuccessResponse{
		Message: "cluster updated",
	})
}
