/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package types

type EvalSSHKeyRequest struct {
	GitProvider     string
	GitlabGroupFlag string
	GitToken        string
}
