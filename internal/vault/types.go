/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package vault

import vaultapi "github.com/hashicorp/vault/api"

type VaultConfiguration struct {
	Config vaultapi.Config
}
