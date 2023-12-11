package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kubefirst/kubefirst-api/pkg/constants"
)

func GetCloudProviderDefaults(c *gin.Context) {
	cloudDefaults := constants.GetCloudDefaults()

	c.JSON(http.StatusOK, cloudDefaults)
}
