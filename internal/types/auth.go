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

// VultrAuth holds necessary auth credentials for interacting with vultr
type CloudflareAuth struct {
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
	SessionToken    string `bson:"session_token,omitempty" json:"session_token,omitempty"`
	Name            string `bson:"name,omitempty" json:"name,omitempty"`
	ID              string `bson:"id,omitempty" json:"id,omitempty"`
}

// Auth for Git Provider
type GitAuth struct {
	Token      string `bson:"git_token,omitempty" json:"git_token,omitempty"`
	User       string `bson:"git_username,omitempty" json:"git_username,omitempty"`
	Owner      string `bson:"git_owner,omitempty" json:"git_owner,omitempty"`
	PublicKey  string `bson:"public_key,omitempty" json:"public_key,omitempty"`
	PrivateKey string `bson:"private_key,omitempty" json:"private_key,omitempty"`
	PublicKeys string `bson:"private_keys,omitempty" json:"private_keys,omitempty"`
}
