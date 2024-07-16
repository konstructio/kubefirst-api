/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package digitalocean

import (
	"context"
	"fmt"

	"github.com/digitalocean/godo"
)

// GetRegions lists all available regions
func (c *DigitaloceanConfiguration) GetRegions() ([]string, error) {
	var regionList []string

	regions, _, err := c.Client.Regions.List(c.Context, &godo.ListOptions{})
	if err != nil {
		return []string{}, err
	}

	for _, region := range regions {
		regionList = append(regionList, region.Slug)
	}

	return regionList, nil
}

func (c *DigitaloceanConfiguration) ListInstances() ([]string, error) {

	DO_MAX_PER_PAGE := 200
	instances, _, err := c.Client.Sizes.List(context.Background(), &godo.ListOptions{PerPage: DO_MAX_PER_PAGE})

	if err != nil {
		return nil, err
	}

	var instanceNames []string
	for _, instance := range instances {
		instanceNames = append(instanceNames, instance.Slug)
	}

	return instanceNames, nil
}

func (c *DigitaloceanConfiguration) GetKubeconfig(clusterName string) ([]byte, error) {
	clusters, _, err := c.Client.Kubernetes.List(context.Background(), &godo.ListOptions{})

	if err != nil {
		return nil, err
	}

	var clusterId string
	for _, cluster := range clusters {
		if cluster.Name == clusterName {
			clusterId = cluster.ID
			continue
		}
	}

	if clusterId == "" {
		return nil, fmt.Errorf("could not find cluster ID for cluster name %s", clusterName)
	}

	config, _, err := c.Client.Kubernetes.GetKubeConfig(context.Background(), clusterId)

	if err != nil {
		return nil, err
	}

	return config.KubeconfigYAML, nil
}
