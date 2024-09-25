/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/konstructio/kubefirst-api/internal/secrets"
	"github.com/konstructio/kubefirst-api/internal/types"
	"github.com/konstructio/kubefirst-api/internal/utils"
)

// GetGitopsCatalogApps godoc
//
//	@Summary		Returns a list of available Kubefirst gitops catalog applications
//	@Description	Returns a list of available Kubefirst gitops catalog applications
//	@Tags			gitops-catalog
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	types.GitopsCatalogApps
//	@Failure		400	{object}	types.JSONFailureResponse
//	@Router			/gitops-catalog/:cluster_name/:cloud_provider/apps [get]
//	@Param			Authorization	header	string	true	"API key"	default(Bearer <API key>)
//
// GetGitopsCatalogApps returns a list of available Kubefirst gitops catalog applications
func GetGitopsCatalogApps(c *gin.Context) {
	cloudProvider, param := c.Params.Get("cloud_provider")
	if !param {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":cloud_provider not provided",
		})
		return
	}

	clusterName, param := c.Params.Get("cluster_name")
	if !param {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":cluster_name not provided",
		})
		return
	}

	kcfg := utils.GetKubernetesClient(clusterName)
	cluster, err := secrets.GetCluster(kcfg.Clientset, clusterName)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: "cluster not found",
		})
		return
	}

	apps, err := secrets.GetGitopsCatalogAppsByCloudProvider(kcfg.Clientset, cloudProvider, cluster.GitProvider)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, apps)
}

// UpdateGitopsCatalogApps godoc
//
//	@Summary		Updates the list of available Kubefirst gitops catalog applications
//	@Description	Updates the list of available Kubefirst gitops catalog applications
//	@Tags			gitops-catalog
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	types.JSONSuccessResponse
//	@Failure		400	{object}	types.JSONFailureResponse
//	@Router			/gitops-catalog/apps/update [get]
//	@Param			Authorization	header	string	true	"API key"	default(Bearer <API key>)
//
// UpdateGitopsCatalogApps updates the list of available Kubefirst gitops catalog applications
func UpdateGitopsCatalogApps(c *gin.Context) {
	kcfg := utils.GetKubernetesClient("TODO: Secrets")
	err := secrets.UpdateGitopsCatalogApps(kcfg.Clientset)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.JSONSuccessResponse{
		Message: "gitops catalog application directory updated",
	})
}
