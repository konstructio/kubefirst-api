/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package middleware

type AuthorizedUser struct {
	Name   string `bson:"name" json:"name"`
	APIKey string `bson:"api_key" json:"api_key"`
}
