/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package civo

import (
	"github.com/civo/civogo"
)

func NewCivo(civoToken string, region string) *civogo.Client {
	civoClient, _ := civogo.NewClient(civoToken, region)

	return civoClient
}
