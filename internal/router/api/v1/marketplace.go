/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kubefirst/kubefirst-api/internal/marketplace"
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
	apps, err := marketplace.ParseActiveApplications()
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: fmt.Sprintf("%s", err),
		})
		return
	}

	c.JSON(http.StatusOK, apps)
}
