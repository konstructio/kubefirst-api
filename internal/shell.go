/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package internal

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/rs/zerolog/log"
)

// ExecShellReturnStrings Exec shell actions returning a string for use by the caller.
func ExecShellReturnStrings(command string, args ...string) (string, string, error) {
	var outb, errb bytes.Buffer
	k := exec.Command(command, args...)
	//  log.Info()().Msg()("Command:", k.String()) //Do not remove this line used for some debugging, will be wrapped by debug log some day.
	k.Stdout = &outb
	k.Stderr = &errb
	err := k.Run()
	if err != nil {
		log.Error().Err(err).Msgf("error executing command")
	}

	if len(errb.String()) > 0 {
		log.Error().Msgf("error executing command: %s", errb.String())
	}

	log.Info().Msgf("OUT: %s", outb.String())
	log.Info().Msgf("Command: %s", command)

	return outb.String(), errb.String(), err
}

// ExecShellReturnStringsV2 exec shell, discard stdout
func ExecShellReturnStringsV2(command string, args ...string) (string, error) {
	var errb bytes.Buffer
	k := exec.Command(command, args...)
	//  log.Info()().Msg()("Command:", k.String()) //Do not remove this line used for some debugging, will be wrapped by debug log some day.
	k.Stdout = io.Discard
	k.Stderr = &errb
	err := k.Run()
	if err != nil {
		log.Error().Err(err).Msgf("error executing command")
	}

	if len(errb.String()) > 0 {
		log.Error().Msgf("error executing command: %s", errb.String())
	}

	return errb.String(), err
}

type outputLogger struct {
	prefix  string
	isError bool
}

func (l outputLogger) Write(p []byte) (int, error) {
	if l.isError {
		log.Error().Msgf("%s %s", l.prefix, string(p))
	} else {
		log.Info().Msgf("%s %s", l.prefix, string(p))
	}
	return len(p), nil
}

// ExecShellWithVars Exec shell actions supporting:
//   - On-the-fly logging of result
//   - Map of Vars loaded
func ExecShellWithVars(osvars map[string]string, command string, args ...string) error {
	allvars := os.Environ()
	for k, v := range osvars {
		allvars = append(allvars, k+"="+v)
		log.Info().Msgf("adding %s=%q to environment", k, strings.Repeat("*", len(v)))
	}

	cmd := exec.Command(command, args...)
	cmd.Stdout = &outputLogger{prefix: command, isError: false}
	cmd.Stderr = &outputLogger{prefix: command, isError: true}
	cmd.Env = allvars

	if err := cmd.Run(); err != nil {
		if exitError := new(exec.ExitError); errors.As(err, &exitError) {
			return fmt.Errorf("command %s failed with exit code %d", command, exitError.ExitCode())
		}

		return fmt.Errorf("command %s failed: %w", command, err)
	}

	return nil
}
