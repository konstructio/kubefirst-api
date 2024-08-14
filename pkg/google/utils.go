/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package google

import (
	"fmt"
	"os"

	log "github.com/rs/zerolog/log"
)

// WriteGoogleApplicationCredentialsFile writes credentials file for use throughout installation
func WriteGoogleApplicationCredentialsFile(googleApplicationCredentials, homeDir string) error {
	file, err := os.Create(fmt.Sprintf("%s/.k1/application-default-credentials.json", homeDir))
	if err != nil {
		return fmt.Errorf("failed to create Google application credentials file in %q: %w", homeDir, err)
	}

	_, err = file.WriteString(googleApplicationCredentials)
	if err != nil {
		log.Error().Msg("error writing google application credentials file")
		return fmt.Errorf("failed to write to Google application credentials file: %w", err)
	}

	// Close the file writer.
	err = file.Close()
	if err != nil {
		log.Error().Msg("error closing file writer")
		return fmt.Errorf("failed to close Google application credentials file writer: %w", err)
	}
	return nil
}
