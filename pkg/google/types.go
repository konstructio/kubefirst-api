/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package google

import "context"

// GoogleConfiguration stores session data to organize all google functions into a single struct
type GoogleConfiguration struct {
	Context context.Context
	Project string
	Region  string
	KeyFile string
}
