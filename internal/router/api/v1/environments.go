package api

import (
	"fmt"
	"time"

	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kubefirst/kubefirst-api/internal/db"
	"github.com/kubefirst/kubefirst-api/internal/types"
	pkgtypes "github.com/kubefirst/kubefirst-api/pkg/types"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

	environmentDefinition.CreationTimestamp = fmt.Sprintf("%v", primitive.NewDateTimeFromTime(time.Now().UTC()))

	newEnv, err := db.Client.InsertEnvironment(environmentDefinition)

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