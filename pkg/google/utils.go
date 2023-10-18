/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package google

import (
	"fmt"
	"log"
	"os"
)

// GoogleConfiguration stores session data to organize all google functions into a single struct
func WriteGoogleApplicationCredentialsFile(googleApplicationCredentials, homeDir string) error {

	file, err := os.Create(fmt.Sprintf("%s/.k1/application_default_credentials.json", homeDir))
	if err != nil {
		return err
	}

	_, err = file.WriteString(googleApplicationCredentials)
	if err != nil {
		log.Fatal("error writing google application credentials file")
		return err
	}

	// Close the file writer.
	err = file.Close()
	if err != nil {
		log.Fatal("error closing file writer")
		return err
	}
	return nil
}
