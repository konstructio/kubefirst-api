package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kubefirst/kubefirst-api/internal/types"
	"github.com/kubefirst/runtime/pkg/civo"
)


func ListInstanceSizesForRegion(c *gin.Context) {
	dnsProvider, param := c.Params.Get("dns_provider")

	if !param {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":dns_provider not provided",
		})
		return
	}

	var instanceSizesRequest types.InstanceSizesRequest
	err := c.Bind(&instanceSizesRequest)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	var instanceSizesResponse types.InstanceSizesResponse

	switch dnsProvider {
		case "civo":
			if instanceSizesRequest.CivoAuth.Token == "" {
				c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
					Message: "missing civo auth token, try again",
				})
				return
			}

			civoConfig := civo.CivoConfiguration{
				Client:  civo.NewCivo(instanceSizesRequest.CivoAuth.Token, instanceSizesRequest.CloudRegion),
				Context: context.Background(),
			}

			instanceSizes, err := civoConfig.ListInstanceSizes()
			if err != nil {
				c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
					Message: err.Error(),
				})
				return
			}

			instanceSizesResponse.InstanceSizes = instanceSizes

		default:
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: fmt.Sprintf("unsupported dns provider: %s", dnsProvider),
			})
			return
	}

	c.JSON(http.StatusOK, instanceSizesResponse)

}

