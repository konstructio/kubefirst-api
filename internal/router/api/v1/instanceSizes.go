package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kubefirst/kubefirst-api/internal/types"
	"github.com/kubefirst/kubefirst-api/pkg/google"
	awsinternal "github.com/kubefirst/runtime/pkg/aws"
	"github.com/kubefirst/runtime/pkg/civo"
	"github.com/kubefirst/runtime/pkg/digitalocean"
	"github.com/kubefirst/runtime/pkg/vultr"
)

func ListInstanceSizesForRegion(c *gin.Context) {
	dnsProvider, param := c.Params.Get("cloud_provider")

	if !param {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":cloud_provider not provided",
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
		c.JSON(http.StatusOK, instanceSizesResponse)
		return

	case "aws":
		if instanceSizesRequest.AWSAuth.AccessKeyID == "" ||
			instanceSizesRequest.AWSAuth.SecretAccessKey == "" ||
			instanceSizesRequest.AWSAuth.SessionToken == "" {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: "missing authentication credentials in request, please check and try again",
			})
			return
		}

		awsConf := &awsinternal.AWSConfiguration{
			Config: awsinternal.NewAwsV3(
				instanceSizesRequest.CloudRegion,
				instanceSizesRequest.AWSAuth.AccessKeyID,
				instanceSizesRequest.AWSAuth.SecretAccessKey,
				instanceSizesRequest.AWSAuth.SessionToken,
			),
		}

		if err != nil {
			fmt.Println("Error describing instance offerings:", err)
			return
		}

		instanceSizes, err := awsConf.ListInstanceSizesForRegion()
		if err != nil {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: err.Error(),
			})
			return
		}

		instanceSizesResponse.InstanceSizes = instanceSizes
		c.JSON(http.StatusOK, instanceSizesResponse)
		return

	case "digitalocean":
		if instanceSizesRequest.DigitaloceanAuth.Token == "" {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: "missing authentication credentials in request, please check and try again",
			})
			return
		}

		digitaloceanConf := digitalocean.DigitaloceanConfiguration{
			Client:  digitalocean.NewDigitalocean(instanceSizesRequest.DigitaloceanAuth.Token),
			Context: context.Background(),
		}

		instances, err := digitaloceanConf.ListInstances()
		if err != nil {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: err.Error(),
			})
			return
		}

		instanceSizesResponse.InstanceSizes = instances
		c.JSON(http.StatusOK, instanceSizesResponse)
		return

	case "vultr":
		if instanceSizesRequest.VultrAuth.Token == "" {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: "missing authentication credentials in request, please check and try again",
			})
			return
		}

		vultrConf := vultr.VultrConfiguration{
			Client:  vultr.NewVultr(instanceSizesRequest.VultrAuth.Token),
			Context: context.Background(),
		}

		instances, err := vultrConf.ListInstances()
		if err != nil {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: err.Error(),
			})
			return
		}

		instanceSizesResponse.InstanceSizes = instances
		c.JSON(http.StatusOK, instanceSizesResponse)
		return

	case "google":

		if instanceSizesRequest.CloudZone == "" {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: "missing cloud_zone arg, please check and try again",
			})
			return
		}

		if instanceSizesRequest.GoogleAuth.ProjectId == "" ||
			instanceSizesRequest.GoogleAuth.KeyFile == "" {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: "missing authentication credentials in request, please check and try again",
			})
			return
		}

		googleConf := google.GoogleConfiguration{
			Context: context.Background(),
			Project: instanceSizesRequest.GoogleAuth.ProjectId,
			Region:  instanceSizesRequest.CloudRegion,
			KeyFile: instanceSizesRequest.GoogleAuth.KeyFile,
		}

		instances, err := googleConf.ListInstances(instanceSizesRequest.CloudZone)

		if err != nil {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: err.Error(),
			})
			return
		}

		instanceSizesResponse.InstanceSizes = instances

	default:
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: fmt.Sprintf("unsupported dns provider: %s", dnsProvider),
		})
		return
	}

	c.JSON(http.StatusOK, instanceSizesResponse)
}
