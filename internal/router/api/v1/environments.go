package api

import (
	"fmt"

	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kubefirst/kubefirst-api/internal/db"
	environments "github.com/kubefirst/kubefirst-api/internal/environments"
	"github.com/kubefirst/kubefirst-api/internal/types"
	pkgtypes "github.com/kubefirst/kubefirst-api/pkg/types"
)

func GetEnvironments(c *gin.Context) {
	environments, err := db.Client.GetEnvironments()

	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, environments)
}



func CreateEnvironment(c *gin.Context) {

	// Bind to variable as application/json, handle error
	var environmentDefinition pkgtypes.Environment
	err := c.Bind(&environmentDefinition)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	newEnv, err := environments.NewEnvironment(environmentDefinition)

	if err != nil {
		c.JSON(http.StatusConflict, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusCreated, newEnv)
}

func DeleteEnvironment(c *gin.Context) {
	envId, param := c.Params.Get("environment_id")

	if !param {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":environment_id not provided",
		})
		return
	}

	err := db.Client.DeleteEnvironment(envId)

	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.JSONSuccessResponse{
		Message: fmt.Sprintf("successfully deleted environment with id: %v", envId),
	})

}

func UpdateEnvironment(c *gin.Context) {
	envId, param := c.Params.Get("environment_id")

	if !param {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":environment_id not provided",
		})
		return
	}
	
	var environmentUpdate types.EnvironmentUpdateRequest
	err := c.Bind(&environmentUpdate)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}
	
	if environmentUpdate.Color == "" && environmentUpdate.Description == "" {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: "please provide a description and or color to update",
		})
		return
	}


	updateErr := db.Client.UpdateEnvironment(envId, environmentUpdate)

	if updateErr != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: updateErr.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.JSONSuccessResponse{
		Message: fmt.Sprintf("successfully updated environment with id: %v", envId),
	})

}
