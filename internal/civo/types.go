/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package civo

import (
	"github.com/civo/civogo"
)

// CivoConfiguration stores session data to organize all Civo functions into a single struct
type CivoConfiguration struct {
	Config civogo.Client
	Region string
}

// CivoTXTRecord stores Civo DNS TXT record data
type CivoTXTRecord struct {
	Name  string
	Value string
	TTL   int
}
