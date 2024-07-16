/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package helpers

type ClusterStatusFlags struct {
	CloudProvider string
	GitProvider   string
	SetupComplete bool
	GitProtocol   string
}
