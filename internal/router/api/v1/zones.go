package api

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/konstructio/kubefirst-api/internal/types"
	"github.com/konstructio/kubefirst-api/pkg/google"
)

// Currently only needs to support google
func ListZonesForRegion(c *gin.Context) {

	var zonesListRequest types.ZonesListRequest
	err := c.Bind(&zonesListRequest)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	googleConf := google.GoogleConfiguration{
		Context: context.Background(),
		Project: zonesListRequest.GoogleAuth.ProjectId,
		Region:  zonesListRequest.CloudRegion,
		KeyFile: zonesListRequest.GoogleAuth.KeyFile,
	}

	var zonesListResponse types.ZonesListResponse

	zones, err := googleConf.GetZones()
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	zonesListResponse.Zones = zones

	c.JSON(http.StatusOK, zonesListResponse)
}
