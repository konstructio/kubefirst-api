/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package vault

const (
	// Default address when leveraging port-forward
	VaultDefaultAddress = "http://127.0.0.1:8200"
	// Name for the Secret that gets created that contains root auth data
	VaultSecretName string = "vault-unseal-secret"
	// Namespace that Vault runs in
	VaultNamespace string = "vault"
	// number of recovery shares for Vault unseal
	RecoveryShares int = 5
	// number of recovery keys for Vault
	RecoveryThreshold int = 3
	// number of secret shares for Vault unseal
	SecretShares = 5
	// number of secret threshold Vault unseal
	SecretThreshold = 3
)
