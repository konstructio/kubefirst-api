/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package types

// AWSAuth holds necessary auth credentials for interacting with aws
type SecretListReference struct {
	Name string   `bson:"name" json:"name"`
	List []string `bson:"list" json:"list"`
}
