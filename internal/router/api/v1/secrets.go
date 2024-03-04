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
	"github.com/kubefirst/kubefirst-api/extensions/aws"
	"github.com/kubefirst/kubefirst-api/internal/constants"
	"github.com/kubefirst/kubefirst-api/internal/env"
	"github.com/kubefirst/kubefirst-api/internal/types"
	"github.com/kubefirst/kubefirst-api/pkg/aws"
	"github.com/kubefirst/kubefirst-api/pkg/k8s"
	awsinternal "github.com/kubefirst/runtime/pkg/aws"
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

	// todo, need to intelligently figure out how to get the kubeconfig
	// for the cluster, based on the cloud provider
	//! init container that uses the environment variable to get the kubeconfig. small image all cloud provider clis, env auth with tokens.
	var awsConf *awsinternal.AWSConfiguration
	var kcfg *k8s.KubernetesClient

	//! restConfig in cluster


	if os.Getenv("CLOUD_PROVIDER") == "aws" && os.Getenv("IS_CLUSTER_ZERO") == "false" {
		awsConf = &awsinternal.AWSConfiguration{
			Config: aws.NewEKSServiceAccountClientV1(),
		}

		kcfg = aws.CreateEKSKubeconfig(awsConf, clusterName)
	} else {
		//! begin

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

		kcfg = k8s.CreateKubeConfig(inCluster, fmt.Sprintf("%s/kubeconfig", clusterDir))

	}

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
