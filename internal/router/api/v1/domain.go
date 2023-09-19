/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package api

import (
	"context"
	"fmt"
	"net/http"

	cloudflare_api "github.com/cloudflare/cloudflare-go"
	"github.com/gin-gonic/gin"
	"github.com/kubefirst/kubefirst-api/internal/types"
	awsinternal "github.com/kubefirst/runtime/pkg/aws"
	"github.com/kubefirst/runtime/pkg/civo"
	cloudflare "github.com/kubefirst/runtime/pkg/cloudflare"
	"github.com/kubefirst/runtime/pkg/digitalocean"
	"github.com/kubefirst/runtime/pkg/vultr"
)

// PostDomains godoc
// @Summary Return a list of registered domains/hosted zones for a cloud provider account
// @Description Return a list of registered domains/hosted zones for a cloud provider account
// @Tags domain
// @Accept json
// @Produce json
// @Param	request	body	types.DomainListRequest	true	"Domain list request in JSON format"
// @Success 200 {object} types.DomainListResponse
// @Failure 400 {object} types.JSONFailureResponse
// @Router /domain/:cloud_provider [post]
// @Param Authorization header string true "API key" default(Bearer <API key>)
// PostDomains returns registered domains/hosted zones for a cloud provider account
func PostDomains(c *gin.Context) {
	dnsProvider, param := c.Params.Get("dns_provider")
	if !param {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":dns_provider not provided",
		})
		return
	}

	// Bind to variable as application/json, handle error
	var domainListRequest types.DomainListRequest
	err := c.Bind(&domainListRequest)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	var domainListResponse types.DomainListResponse

	switch dnsProvider {
	case "aws":
		if domainListRequest.AWSAuth.AccessKeyID == "" ||
			domainListRequest.AWSAuth.SecretAccessKey == "" ||
			domainListRequest.AWSAuth.SessionToken == "" {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: "missing authentication credentials in request, please check and try again",
			})
			return
		}
		awsConf := &awsinternal.AWSConfiguration{
			Config: awsinternal.NewAwsV3(
				domainListRequest.CloudRegion,
				domainListRequest.AWSAuth.AccessKeyID,
				domainListRequest.AWSAuth.SecretAccessKey,
				domainListRequest.AWSAuth.SessionToken,
			),
		}

		domains, err := awsConf.GetHostedZones()
		if err != nil {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: err.Error(),
			})
			return
		}
		domainListResponse.Domains = domains
	case "cloudflare":
		//check for token, make sure it aint blank
		if domainListRequest.CloudflareAuth.APIToken == "" {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: "missing authentication credentials in request, please check and try again",
			})
			return
		}

		client, err := cloudflare_api.NewWithAPIToken(domainListRequest.CloudflareAuth.APIToken)
		if err != nil {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: fmt.Sprintf("Could not create cloudflare client, %v", err),
			})
			return
		}

		cloudflareConf := cloudflare.CloudflareConfiguration{
			Client:  client,
			Context: context.Background(),
		}

		domains, err := cloudflareConf.GetDNSDomains()
		if err != nil {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: err.Error(),
			})
			return
		}

		domainListResponse.Domains = domains

	case "civo":
		if domainListRequest.CivoAuth.Token == "" {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: "missing authentication credentials in request, please check and try again",
			})
			return
		}
		civoConf := civo.CivoConfiguration{
			Client:  civo.NewCivo(domainListRequest.CivoAuth.Token, domainListRequest.CloudRegion),
			Context: context.Background(),
		}

		domains, err := civoConf.GetDNSDomains(domainListRequest.CloudRegion)
		if err != nil {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: err.Error(),
			})
			return
		}
		domainListResponse.Domains = domains
	case "digitalocean":
		if domainListRequest.DigitaloceanAuth.Token == "" {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: "missing authentication credentials in request, please check and try again",
			})
			return
		}
		digitaloceanConf := digitalocean.DigitaloceanConfiguration{
			Client:  digitalocean.NewDigitalocean(domainListRequest.DigitaloceanAuth.Token),
			Context: context.Background(),
		}

		domains, err := digitaloceanConf.GetDNSDomains()
		if err != nil {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: err.Error(),
			})
			return
		}
		domainListResponse.Domains = domains
	case "vultr":
		if domainListRequest.VultrAuth.Token == "" {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: "missing authentication credentials in request, please check and try again",
			})
			return
		}
		vultrConf := vultr.VultrConfiguration{
			Client:  vultr.NewVultr(domainListRequest.VultrAuth.Token),
			Context: context.Background(),
		}

		domains, err := vultrConf.GetDNSDomains()
		if err != nil {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: err.Error(),
			})
			return
		}
		domainListResponse.Domains = domains
	default:
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: fmt.Sprintf("unsupported provider: %s", dnsProvider),
		})
		return
	}

	c.JSON(http.StatusOK, domainListResponse)
}
