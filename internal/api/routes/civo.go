/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kubefirst/kubefirst-api/internal/types"
	civointernal "github.com/kubefirst/runtime/pkg/civo"
)

// GetValidateCivoDomain godoc
// @Summary Returns status of whether or not a Civo hosted zone is validated for use with Kubefirst
// @Description Returns status of whether or not a Civo hosted zone is validated for use with Kubefirst
// @Tags civo
// @Accept json
// @Produce json
// @Param	domain	path	string	true	"Domain name, no trailing dot"
// @Success 200 {object} types.CivoDomainValidationResponse
// @Failure 400 {object} types.JSONFailureResponse
// @Router /civo/domain/validate/:domain [get]
// GetValidateCivoDomain returns status for a Civo domain validation
func GetValidateCivoDomain(c *gin.Context) {
	domainName, exists := c.Params.Get("domain")
	if !exists {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":domain parameter not provided in request",
		})
		return
	}

	// Run validate func
	validated := civointernal.TestDomainLiveness(false, domainName, "", "")
	if !validated {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: "domain validation failed",
		})
		return
	}
	c.JSON(http.StatusOK, types.CivoDomainValidationResponse{
		Validated: validated,
	})
}
