/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package types

// AWSAuth holds necessary auth credentials for interacting with aws
type AWSAuth struct {
	AccessKeyID     string `bson:"access_key_id" json:"access_key_id"`
	SecretAccessKey string `bson:"secret_access_key" json:"secret_access_key"`
	SessionToken    string `bson:"session_token" json:"session_token"`
}

// CivoAuth holds necessary auth credentials for interacting with civo
type CivoAuth struct {
	Token string `bson:"token" json:"token"`
}

// DigitaloceanAuth holds necessary auth credentials for interacting with digitalocean
type DigitaloceanAuth struct {
	Token        string `bson:"token" json:"token"`
	SpacesKey    string `bson:"spaces_key" json:"spaces_key"`
	SpacesSecret string `bson:"spaces_secret" json:"spaces_secret"`
}

// VultrAuth holds necessary auth credentials for interacting with vultr
type VultrAuth struct {
	Token string `bson:"token" json:"token"`
}

// StateStoreCredentials
type StateStoreCredentials struct {
	AccessKeyID     string `bson:"access_key_id,omitempty" json:"access_key_id,omitempty"`
	SecretAccessKey string `bson:"secret_access_key,omitempty" json:"secret_access_key,omitempty"`
	Name            string `bson:"name,omitempty" json:"name,omitempty"`
	ID              string `bson:"id,omitempty" json:"id,omitempty"`
}
