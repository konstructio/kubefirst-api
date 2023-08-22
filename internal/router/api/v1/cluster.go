/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package api

import (
	"fmt"
	"os"

	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kubefirst/kubefirst-api/internal/constants"
	"github.com/kubefirst/kubefirst-api/internal/db"
	"github.com/kubefirst/kubefirst-api/internal/gitShim"
	"github.com/kubefirst/kubefirst-api/internal/services"
	"github.com/kubefirst/kubefirst-api/internal/types"
	"github.com/kubefirst/kubefirst-api/internal/utils"
	"github.com/kubefirst/kubefirst-api/providers/aws"
	"github.com/kubefirst/kubefirst-api/providers/civo"
	"github.com/kubefirst/kubefirst-api/providers/digitalocean"
	"github.com/kubefirst/kubefirst-api/providers/vultr"
	"github.com/kubefirst/runtime/pkg/k8s"
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
// @Param Authorization header string true "API key" default(Bearer <API key>)
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

	if rec.LastCondition != "" {
		err = db.Client.UpdateCluster(rec.ClusterName, "last_condition", "")
		if err != nil {
			log.Warnf("error updating cluster last_condition field: %s", err)
		}
	}
	if rec.Status == constants.ClusterStatusError {
		err = db.Client.UpdateCluster(rec.ClusterName, "status", constants.ClusterStatusDeleting)
		if err != nil {
			log.Warnf("error updating cluster status field: %s", err)
		}
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
// @Param Authorization header string true "API key" default(Bearer <API key>)
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
// @Param Authorization header string true "API key" default(Bearer <API key>)
// GetClusters returns all known configured clusters
func GetClusters(c *gin.Context) {
	// Retrieve all clusters info
	allClusters, err := db.Client.GetClusters()
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
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
// @Param Authorization header string true "API key" default(Bearer <API key>)
// PostCreateCluster handles a request to create a cluster
func PostCreateCluster(c *gin.Context) {

	// jsonData, err := io.ReadAll(c.Request.Body)
	// fmt.Spintf(string(jsonData))
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
		if cluster.LastCondition != "" {
			err = db.Client.UpdateCluster(cluster.ClusterName, "last_condition", "")
			if err != nil {
				log.Warnf("error updating cluster last_condition field: %s", err)
			}
		}
		if cluster.Status == constants.ClusterStatusError {
			err = db.Client.UpdateCluster(cluster.ClusterName, "status", constants.ClusterStatusProvisioning)
			if err != nil {
				log.Warnf("error updating cluster status field: %s", err)
			}
		}
	}

	// Determine authentication type
	inCluster := false
	useSecretForAuth := false
	var k1AuthSecret = map[string]string{}
	if os.Getenv("IN_CLUSTER") == "true" {
		inCluster = true
	}

	if inCluster {
		kcfg := k8s.CreateKubeConfig(inCluster, "")
		k1AuthSecret, err := k8s.ReadSecretV2(kcfg.Clientset, constants.KubefirstNamespace, constants.KubefirstAuthSecretName)
		if err != nil {
			log.Warnf("authentication secret does not exist, continuing: %s", err)
		} else {
			log.Info("authentication secret exists, checking contents")
			if k1AuthSecret == nil {
				c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
					Message: "authentication secret found but contains no data, please check and try again",
				})
				return
			}
			useSecretForAuth = true
		}
	}

	switch clusterDefinition.CloudProvider {
	case "aws":
		if useSecretForAuth {
			err := utils.ValidateAuthenticationFields(k1AuthSecret)
			if err != nil {
				c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
					Message: fmt.Sprintf("error checking aws auth: %s", err),
				})
				return
			}
			clusterDefinition.AWSAuth = types.AWSAuth{
				AccessKeyID:     k1AuthSecret["aws-access-key-id"],
				SecretAccessKey: k1AuthSecret["aws-secret-access-key"],
				SessionToken:    k1AuthSecret["aws-session-token"],
			}
		} else {
			if clusterDefinition.AWSAuth.AccessKeyID == "" ||
				clusterDefinition.AWSAuth.SecretAccessKey == "" ||
				clusterDefinition.AWSAuth.SessionToken == "" {
				c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
					Message: "missing authentication credentials in request, please check and try again",
				})
				return
			}
		}
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
		if useSecretForAuth {
			err := utils.ValidateAuthenticationFields(k1AuthSecret)
			if err != nil {
				c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
					Message: fmt.Sprintf("error checking civo auth: %s", err),
				})
				return
			}
			clusterDefinition.CivoAuth = types.CivoAuth{
				Token: k1AuthSecret["civo-token"],
			}
		} else {
			if clusterDefinition.CivoAuth.Token == "" {
				c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
					Message: "missing authentication credentials in request, please check and try again",
				})
				return
			}
		}
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
		if useSecretForAuth {
			err := utils.ValidateAuthenticationFields(k1AuthSecret)
			if err != nil {
				c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
					Message: fmt.Sprintf("error checking digitalocean auth: %s", err),
				})
				return
			}
			clusterDefinition.DigitaloceanAuth = types.DigitaloceanAuth{
				Token:        k1AuthSecret["do-token"],
				SpacesKey:    k1AuthSecret["do-spaces-key"],
				SpacesSecret: k1AuthSecret["do-spaces-token"],
			}
		} else {
			if clusterDefinition.DigitaloceanAuth.Token == "" ||
				clusterDefinition.DigitaloceanAuth.SpacesKey == "" ||
				clusterDefinition.DigitaloceanAuth.SpacesSecret == "" {
				c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
					Message: "missing authentication credentials in request, please check and try again",
				})
				return
			}
		}
		go func() {
			err = digitalocean.CreateDigitaloceanCluster(&clusterDefinition)
			if err != nil {
				log.Errorf(err.Error())
			}
		}()

		c.JSON(http.StatusAccepted, types.JSONSuccessResponse{
			Message: "cluster create enqueued",
		})
	case "vultr":
		if useSecretForAuth {
			err := utils.ValidateAuthenticationFields(k1AuthSecret)
			if err != nil {
				c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
					Message: fmt.Sprintf("error checking vultr auth: %s", err),
				})
				return
			}
			clusterDefinition.VultrAuth = types.VultrAuth{
				Token: k1AuthSecret["vultr-api-key"],
			}
		} else {
			if clusterDefinition.VultrAuth.Token == "" {
				c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
					Message: "missing authentication credentials in request, please check and try again",
				})
				return
			}
		}
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
// @Param Authorization header string true "API key" default(Bearer <API key>)
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
// @Param Authorization header string true "API key" default(Bearer <API key>)
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

	log.Info("Restoring database")
	// Restores database record
	err = db.Client.Restore(&req)

	// Create default service entries
	log.Info("Adding default services")
	cl, _ := db.Client.GetCluster(req.ClusterName)
	err = services.AddDefaultServices(&cl)
	if err != nil {
		log.Errorf("error adding default service entries for cluster %s: %s", cl.ClusterName, err)
	}

	err = gitShim.PrepareMgmtCluster(cl)
	if err != nil {
		log.Fatalf("error cloning repository: %s", err)
	}

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
// @Param Authorization header string true "API key" default(Bearer <API key>)
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
