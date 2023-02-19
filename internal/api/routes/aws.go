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
	"github.com/kubefirst/kubefirst-api/internal/aws"
	"github.com/kubefirst/kubefirst-api/internal/types"
	//log "github.com/sirupsen/logrus"
)

// GetAWSProfiles godoc
// @Summary Returns a list of configured AWS profiles
// @Description Returns a list of configured AWS profiles
// @Tags aws
// @Accept json
// @Produce json
// @Success 200 {object} types.AWSProfilesResponse
// @Failure 400 {object} types.JSONFailureResponse
// @Router /aws/profiles [get]
// GetAWSProfiles returns AWS profiles found on the local host
func GetAWSProfiles(c *gin.Context) {
	// Fetch profiles
	awsConfig := &aws.Conf
	profiles, err := awsConfig.ListLocalProfiles("")
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: fmt.Sprintf("%s", err),
		})
		return
	}
	c.JSON(http.StatusOK, types.AWSProfilesResponse{
		Profiles: profiles,
	})
}

// GetValidateAWSDomain godoc
// @Summary Returns status of whether or not an AWS hosted zone is validated for use with Kubefirst
// @Description Returns status of whether or not an AWS hosted zone is validated for use with Kubefirst
// @Tags aws
// @Accept json
// @Produce json
// @Param	domain	path	string	true	"Domain name, no trailing dot"
// @Success 200 {object} types.AWSDomainValidateResponse
// @Failure 400 {object} types.JSONFailureResponse
// @Router /aws/domain/validate/:domain [get]
// GetValidateAWSDomain returns status for an AWS domain validation
func GetValidateAWSDomain(c *gin.Context) {
	domainName, exists := c.Params.Get("domain")
	if !exists {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":domain parameter not provided in request",
		})
		return
	}

	// Run validate func
	awsClient := &aws.Conf
	validated, err := awsClient.TestHostedZoneLiveness(domainName)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: fmt.Sprintf("%s", err),
		})
		return
	}
	c.JSON(http.StatusOK, types.AWSDomainValidateResponse{
		Validated: validated,
	})
}
