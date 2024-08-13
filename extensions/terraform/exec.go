/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package terraform

import (
	log "github.com/rs/zerolog/log"
)

type outputLogger struct {
	prefix  string
	isError bool
}

func (l outputLogger) Write(p []byte) (n int, err error) {
	if l.isError {
		log.Error().Msgf("%s %s", l.prefix, string(p))
	} else {
		log.Info().Msgf("%s %s", l.prefix, string(p))
	}
	return len(p), nil
}
