/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package constants

const (
	// The Namespace in which Kubefirst runs in-cluster
	KubefirstNamespace = "kubefirst"

	// The name of the Secret that holds authentication credentials
	KubefirstAuthSecretName = "kubefirst-secret"

	// The Secret created to hold initial cluster import data
	KubefirstImportSecretName = "kubefirst-initial-cluster-import"
)
