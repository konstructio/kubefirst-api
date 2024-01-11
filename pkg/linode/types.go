/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package linode

import (
	"context"

	"github.com/linode/linodego"
)

type LinodeConfiguration struct {
	Client  linodego.Client
	Context context.Context
}
