/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package internal

import (
	"net/http"
)

type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

var SupportedPlatforms = []string{
	"akamai-github",
	"aws-github",
	"aws-gitlab",
	"civo-github",
	"civo-gitlab",
	"digitalocean-github",
	"digitalocean-gitlab",
	"google-github",
	"google-gitlab",
	"k3d-github",
	"k3d-gitlab",
	"k3s-gitlab",
	"vultr-github",
	"vultr-gitlab",
}
