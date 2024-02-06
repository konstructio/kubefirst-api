/*
Copyright (C) 2021-2024, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/kubefirst/kubefirst-api/internal/constants"
	"github.com/kubefirst/kubefirst-api/internal/env"
	"github.com/kubefirst/kubefirst-api/internal/types"
	"github.com/kubefirst/kubefirst-api/pkg/k8s"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	jsonString, err := mapToStructuredJSON(secrets)

	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, jsonString)
}

func CreateClusterSecret(c *gin.Context) {
	clusterName, param := c.Params.Get("cluster_name")
	if !param {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":cluster_name not provided",
		})
		return
	}

	secretName, param := c.Params.Get("secret")
	if !param {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":secret not provided",
		})
		return
	}

	var secretValues map[string]interface{}
	err := c.Bind(&secretValues)
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

	env, _ := env.GetEnv(constants.SilenceGetEnv)

	var inCluster bool = false
	if env.InCluster == "true" {
		inCluster = true
	}

	clusterDir := fmt.Sprintf("%s/.k1/%s", homeDir, clusterName)

	kcfg := k8s.CreateKubeConfig(inCluster, fmt.Sprintf("%s/kubeconfig", clusterDir))

	bytes, err := json.Marshal(secretValues)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: "error stringifying object",
		})
		return
	}

	secretValuesMap, err := parseJSONToMap(string(bytes))

	secretToCreate := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: "kubefirst",
		},
		Data: secretValuesMap,
	}

	err = k8s.CreateSecretV2(kcfg.Clientset, secretToCreate)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.JSONSuccessResponse{
		Message: "cluster secret created",
	})
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

	var secretValues map[string]interface{}
	err := c.Bind(&secretValues)
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

	env, _ := env.GetEnv(constants.SilenceGetEnv)

	var inCluster bool = false
	if env.InCluster == "true" {
		inCluster = true
	}

	clusterDir := fmt.Sprintf("%s/.k1/%s", homeDir, clusterName)

	kcfg := k8s.CreateKubeConfig(inCluster, fmt.Sprintf("%s/kubeconfig", clusterDir))

	bytes, err := json.Marshal(secretValues)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: "error stringifying object",
		})
		return
	}

	secretValuesMap, err := parseJSONToMap(string(bytes))
	err = k8s.UpdateSecretV2(kcfg.Clientset, "kubefirst", secret, secretValuesMap)
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

func parseJSONToMap(jsonStr string) (map[string][]byte, error) {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &result)
	if err != nil {
		return nil, err
	}

	secretData := make(map[string][]byte)
	for key, value := range result {
		switch v := value.(type) {
		case map[string]interface{}, []interface{}: // For nested structures, marshal back to JSON
			bytes, err := json.Marshal(v)
			if err != nil {
				return nil, err
			}
			secretData[key] = bytes
		default:
			bytes, err := json.Marshal(v)
			if err != nil {
				return nil, err
			}
			secretData[key] = bytes
		}
	}

	return secretData, nil
}

// mapToStructuredJSON takes a map[string]interface{} as input, where some values may be JSON strings,
// and returns an interface{} that represents the structured JSON object.
func mapToStructuredJSON(input map[string]interface{}) (interface{}, error) {
	// Create a result map to hold our structured data.
	result := make(map[string]interface{})

	for key, value := range input {
		switch v := value.(type) {
		case string:
			// Attempt to unmarshal the string into a map if it's JSON.
			// This covers cases where the string is actually a JSON object or array.
			var jsonVal interface{}
			err := json.Unmarshal([]byte(v), &jsonVal)
			if err == nil {
				// If successful, use the unmarshaled value.
				result[key] = jsonVal
			} else {
				// If the string is not JSON, remove surrounding quotes if present and use the string directly.
				if len(v) > 1 && v[0] == '"' && v[len(v)-1] == '"' {
					result[key] = v[1 : len(v)-1]
				} else {
					result[key] = v
				}
			}
		default:
			// For all other types, use the value directly.
			result[key] = value
		}
	}

	return result, nil
}
