/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package types

type EnvironmentUpdateRequest struct {
	Color       			string 						 `bson:"color,omitempty" json:"color,omitempty"`
	Description       string 						 `bson:"description,omitempty" json:"description,omitempty"`
}
