/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k8s

import (
	"fmt"
	"net"
)

// CheckForExistingPortForwards determines whether or not port forwards are already running
func CheckForExistingPortForwards(ports ...int) error {
	for _, port := range ports {
		listen, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%v", port))
		if err != nil {
			if listen != nil {
				_ = listen.Close()
			}
			return fmt.Errorf("port %v is in use", port)
		}
		_ = listen.Close()
	}

	return nil
}
