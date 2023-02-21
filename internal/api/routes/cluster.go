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
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	cl "github.com/kubefirst/kubefirst-api/internal/cluster"
	"github.com/kubefirst/kubefirst-api/internal/types"
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
			Message: fmt.Sprintf("%s", ":cluster_name not provided"),
		})
		return
	}

	// Run create func
	clusters := cl.ClusterEntries{}
	err := clusters.DeleteOne(clusterName)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: fmt.Sprintf("%s", err),
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
// @Success 200 {object} cluster.ClusterEntry
// @Failure 400 {object} types.JSONFailureResponse
// @Router /cluster/:cluster_name [get]
// GetCluster returns a specific configured cluster
func GetCluster(c *gin.Context) {
	clusterName, param := c.Params.Get("cluster_name")
	if !param {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: fmt.Sprintf("%s", ":cluster_name not provided"),
		})
		return
	}

	// Retrieve cluster info
	clusters := cl.ClusterEntries{}
	cluster, err := clusters.ReadOne(clusterName)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: fmt.Sprintf("%s", err),
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
// @Success 200 {object} []cluster.ClusterEntry
// @Failure 400 {object} types.JSONFailureResponse
// @Router /cluster [get]
// GetClusters returns all known configured cluster
func GetClusters(c *gin.Context) {
	// Retrieve all clusters info
	clusters := cl.ClusterEntries{}
	allClusters, err := clusters.ReadAll()
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
			Message: fmt.Sprintf("%s", ":cluster_name not provided"),
		})
		return
	}

	// Bind to variable as application/json, handle error
	var clusterDefinition types.ClusterDefinition
	err := c.Bind(&clusterDefinition)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: fmt.Sprintf("%s", err),
		})
		return
	}

	// Run create func
	err = cl.CreateCluster(clusterName, clusterDefinition)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: fmt.Sprintf("%s", err),
		})
		return
	}
	c.JSON(http.StatusAccepted, types.JSONSuccessResponse{
		Message: "cluster create enqueued",
	})

}
