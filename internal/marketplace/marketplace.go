/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package marketplace

import (
	"fmt"

	"github.com/kubefirst/kubefirst-api/internal/gitShim"
	"github.com/kubefirst/kubefirst-api/internal/types"
	"gopkg.in/yaml.v2"
)

// ReadActiveApplications reads the active upstream application manifest
func ReadActiveApplications() (types.MarketplaceApps, error) {
	gh := gitShim.GitHubClient{
		Client: gitShim.NewGitHub(),
	}

	activeContent, err := gh.ReadMarketplaceRepoContents()
	if err != nil {
		return types.MarketplaceApps{}, fmt.Errorf("error retrieving marketplace repository content: %s", err)
	}

	index, err := gh.ReadMarketplaceIndex(activeContent)
	if err != nil {
		return types.MarketplaceApps{}, fmt.Errorf("error retrieving marketplace index content: %s", err)
	}

	var out types.MarketplaceApps

	err = yaml.Unmarshal(index, &out)
	if err != nil {
		return types.MarketplaceApps{}, fmt.Errorf("error retrieving marketplace applications: %s", err)
	}

	return out, nil
}

// ReadApplicationDirectory reads a marketplace application's directory
func ReadApplicationDirectory(applicationName string) ([][]byte, error) {
	gh := gitShim.GitHubClient{
		Client: gitShim.NewGitHub(),
	}

	activeContent, err := gh.ReadMarketplaceRepoContents()
	if err != nil {
		return [][]byte{}, fmt.Errorf("error retrieving marketplace app directory content: %s", err)
	}

	contents, err := gh.ReadMarketplaceAppDirectory(activeContent, applicationName)
	if err != nil {
		return [][]byte{}, fmt.Errorf("error retrieving marketplace app directory content: %s", err)
	}

	return contents, nil
}
