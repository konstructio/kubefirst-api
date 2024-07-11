/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package civo

import (
	"context"

	"github.com/civo/civogo"
)

type CivoConfiguration struct {
	Client  *civogo.Client
	Context context.Context
}
