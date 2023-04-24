/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package types

type LogMessage struct {
	Type    string `json:"-"`
	Message string `json:"message"`
}
