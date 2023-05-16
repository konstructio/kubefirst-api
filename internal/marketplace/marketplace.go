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

// ParseActiveApplications reads the active upstream application manifest
func ParseActiveApplications() (types.MarketplaceApps, error) {
	gh := gitShim.GitHubClient{
		Client: gitShim.NewGitHub(),
	}

	activeContent, err := gh.ReadMarketplaceRepoContents()
	if err != nil {
		return types.MarketplaceApps{}, fmt.Errorf("error retrieving marketplace repository content: %s", err)
	}

	index, err := gh.ParseMarketplaceIndex(activeContent)
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
