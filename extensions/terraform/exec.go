/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package terraform

import (
	"bufio"
	"os"
	"os/exec"
	"strings"

	log "github.com/rs/zerolog/log"
)

// ExecShellWithVars Exec shell actions supporting:
//   - On-the-fly logging of result
//   - Map of Vars loaded
func ExecShellWithVars(osvars map[string]string, command string, args ...string) error {
	for k, v := range osvars {
		os.Setenv(k, v)
		suppressedValue := strings.Repeat("*", len(v))
		log.Printf(" export %s = %s", k, suppressedValue)
	}
	cmd := exec.Command(command, args...)
	cmdReaderOut, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("failed creating out pipe for: %v", command)
		return err
	}
	cmdReaderErr, err := cmd.StderrPipe()
	if err != nil {
		log.Printf("failed creating out pipe for: %v", command)
		return err
	}

	scannerOut := bufio.NewScanner(cmdReaderOut)
	stdOut := make(chan string)
	go reader(scannerOut, stdOut)
	doneOut := make(chan bool)

	scannerErr := bufio.NewScanner(cmdReaderErr)
	stdErr := make(chan string)
	go reader(scannerErr, stdErr)
	doneErr := make(chan bool)
	go func() {
		for msg := range stdOut {
			log.Printf("OUT: %s", msg)
		}
		doneOut <- true
	}()
	go func() {
		// STD Err should not be supressed, as it prevents to troubleshoot issues in case something fails.
		// On linux StdErr > StdOut by design in terms of priority.
		for msg := range stdErr {
			log.Printf("ERR: %s", msg)
		}
		doneErr <- true
	}()

	err = cmd.Run()
	if err != nil {
		log.Printf("command %q failed", command)
		close(stdOut)
		close(stdErr)
		<-doneOut
		<-doneErr
		return err
	} else {
		close(stdOut)
		close(stdErr)
	}
	<-doneOut
	<-doneErr
	return nil
}

// Not meant to be exported, for internal use only.
func reader(scanner *bufio.Scanner, out chan string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Error processing logs from command. Error: %s", r)
		}
	}()
	for scanner.Scan() {
		out <- scanner.Text()
	}
}
