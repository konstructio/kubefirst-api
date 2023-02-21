/*
Copyright © 2023 Kubefirst <kubefirst.io>
Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:
The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.
THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kubefirst/kubefirst-api/internal/civo"
	"github.com/kubefirst/kubefirst-api/internal/types"
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
	civoClient := &civo.Conf
	validated, err := civoClient.TestHostedZoneLiveness(domainName)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: fmt.Sprintf("%s", err),
		})
		return
	}
	c.JSON(http.StatusOK, types.CivoDomainValidationResponse{
		Validated: validated,
	})
}
