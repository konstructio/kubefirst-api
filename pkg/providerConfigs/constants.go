/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package providerConfigs

import (
	"runtime"

	runtimepkg "github.com/kubefirst/kubefirst-api/internal"
)

const (
	GithubHost             = "github.com"
	GitlabHost             = "gitlab.com"
	KubectlClientVersion   = "v1.25.7"
	LocalhostOS            = runtime.GOOS
	LocalhostArch          = runtime.GOARCH
	TerraformClientVersion = "1.3.8"

	ArgocdPortForwardURL = runtimepkg.ArgocdPortForwardURL
	VaultPortForwardURL  = runtimepkg.VaultPortForwardURL

	TokenRegexPattern = "<([A-Z_0-9-]+)>"
	leftDelimiter     = "<%"
	rightDelimiter    = "%>"
)
