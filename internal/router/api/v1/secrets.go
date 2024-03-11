/*
Copyright (C) 2021-2024, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package api

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/kubefirst/kubefirst-api/internal/constants"
	"github.com/kubefirst/kubefirst-api/internal/env"
	"github.com/kubefirst/kubefirst-api/internal/types"
	"github.com/kubefirst/kubefirst-api/pkg/k8s"
)

func GetClusterSecret(c *gin.Context) {
	clusterName, param := c.Params.Get("cluster_name")
	if !param {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":cluster_name not provided",
		})
		return
	}

	secret, param := c.Params.Get("secret")
	if !param {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":secret not provided",
		})
		return
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: "error finding home directory",
		})
		return
	}

	clusterDir := fmt.Sprintf("%s/.k1/%s", homeDir, clusterName)

	env, _ := env.GetEnv(constants.SilenceGetEnv)

	var inCluster bool = false
	if env.InCluster == "true" {
		inCluster = true
	}

	kcfg := k8s.CreateKubeConfig(inCluster, fmt.Sprintf("%s/kubeconfig", clusterDir))

	secrets, err := k8s.ReadSecretV2(kcfg.Clientset, "kubefirst", secret)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, secrets)
}

func UpdateClusterSecret(c *gin.Context) {
	clusterName, param := c.Params.Get("cluster_name")
	if !param {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":cluster_name not provided",
		})
		return
	}

	secret, param := c.Params.Get("secret")
	if !param {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":secret not provided",
		})
		return
	}

	var clusterSecretUpdates k8s.UpdateSecretArgs
	err := c.Bind(&clusterSecretUpdates)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: "error finding home directory",
		})
		return
	}

	clusterDir := fmt.Sprintf("%s/.k1/%s", homeDir, clusterName)

	env, _ := env.GetEnv(constants.SilenceGetEnv)

	var inCluster bool = false
	if env.InCluster == "true" {
		inCluster = true
	}

	kcfg := k8s.CreateKubeConfig(inCluster, fmt.Sprintf("%s/kubeconfig", clusterDir))

	err = k8s.UpdateSecretV2(kcfg.Clientset, "kubefirst", secret, clusterSecretUpdates)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.JSONSuccessResponse{
		Message: "cluster secret updated",
	})
}
