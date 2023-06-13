/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kubefirst/kubefirst-api/internal/db"
	"github.com/kubefirst/kubefirst-api/internal/types"
)

// GetMarketplaceApps godoc
// @Summary Returns a list of available Kubefirst marketplace applications
// @Description Returns a list of available Kubefirst marketplace applications
// @Tags marketplace
// @Accept json
// @Produce json
// @Success 200 {object} types.MarketplaceApps
// @Failure 400 {object} types.JSONFailureResponse
// @Router /marketplace/apps [get]
// GetMarketplaceApps returns a list of available Kubefirst marketplace applications
func GetMarketplaceApps(c *gin.Context) {
	apps, err := db.Client.GetMarketplaceApps()
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, apps)
}

// UpdateMarketplaceApps godoc
// @Summary Updates the list of available Kubefirst marketplace applications
// @Description Updates the list of available Kubefirst marketplace applications
// @Tags marketplace
// @Accept json
// @Produce json
// @Success 200 {object} types.JSONSuccessResponse
// @Failure 400 {object} types.JSONFailureResponse
// @Router /marketplace/apps/update [get]
// UpdateMarketplaceApps updates the list of available Kubefirst marketplace applications
func UpdateMarketplaceApps(c *gin.Context) {
	err := db.Client.UpdateMarketplaceApps()
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.JSONSuccessResponse{
		Message: "marketplace application directory updated",
	})
}
