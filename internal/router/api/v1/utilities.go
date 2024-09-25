/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/konstructio/kubefirst-api/internal/types"
)

// getHealth godoc
//
//	@Summary		Return health status if the application is running.
//	@Description	Return health status if the application is running.
//	@Tags			health
//	@Produce		json
//	@Success		200	{object}	types.JSONHealthResponse
//	@Router			/health [get]
//	@Param			Authorization	header	string	true	"API key"	default(Bearer <API key>)
func GetHealth(c *gin.Context) {
	c.JSON(http.StatusOK, types.JSONHealthResponse{
		Status: "healthz",
	})
}
