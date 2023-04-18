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
	"github.com/kubefirst/kubefirst-api/providers/civo"
	"github.com/kubefirst/kubefirst-api/providers/k3d"
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
	mdbcl := &db.MongoDBClient{}
	err := mdbcl.InitDatabase()
	if err != nil {
		log.Error(err)
	}

	rec, err := mdbcl.GetCluster(clusterName)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	switch rec.CloudProvider {
	case "k3d":
	case "civo":
		err := civo.DeleteCivoCluster(&rec)
		if err != nil {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: err.Error(),
			})
			return
		}
	}

	err = mdbcl.DeleteCluster(clusterName)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusAccepted, types.JSONSuccessResponse{
		Message: "cluster deleted",
	})
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
	mdbcl := &db.MongoDBClient{}
	err := mdbcl.InitDatabase()
	if err != nil {
		log.Error(err)
	}

	cluster, err := mdbcl.GetCluster(clusterName)
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
	mdbcl := &db.MongoDBClient{}
	err := mdbcl.InitDatabase()
	if err != nil {
		log.Error(err)
	}

	allClusters, err := mdbcl.GetClusters()
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
	switch clusterDefinition.CloudProvider {
	case "k3d":
		err = k3d.CreateK3DCluster(&clusterDefinition)
		if err != nil {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: fmt.Sprintf("%s", err),
			})
			return
		}

		c.JSON(http.StatusAccepted, types.JSONSuccessResponse{
			Message: "cluster created",
		})
	case "civo":
		err = civo.CreateCivoCluster(&clusterDefinition)
		if err != nil {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: fmt.Sprintf("%s", err),
			})
			return
		}

		c.JSON(http.StatusAccepted, types.JSONSuccessResponse{
			Message: "cluster created",
		})
	}
}
