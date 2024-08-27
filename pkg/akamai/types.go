/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package akamai

import (
	"context"

	"github.com/konstructio/kubefirst-api/pkg/types"
	"github.com/linode/linodego"
)

type AkamaiConfiguration struct {
	Client  linodego.Client
	Context context.Context
}

type AkamaiBucketAndKeysConfiguration struct {
	StateStoreDetails     types.StateStoreDetails
	StateStoreCredentials types.StateStoreCredentials
}
