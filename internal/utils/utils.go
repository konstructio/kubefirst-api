/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package utils

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/kubefirst/kubefirst-api/internal/db"

	log "github.com/rs/zerolog/log"
)

// CreateK1Directory
func CreateK1Directory(clusterName string) {
	// Create k1 dir if it doesn't exist
	homePath, err := os.UserHomeDir()
	if err != nil {
		log.Info().Msg(err.Error())
	}
	k1Dir := fmt.Sprintf("%s/.k1/%s", homePath, clusterName)
	if _, err := os.Stat(k1Dir); os.IsNotExist(err) {
		err := os.MkdirAll(k1Dir, os.ModePerm)
		if err != nil {
			log.Info().Msgf("%s directory already exists, continuing", k1Dir)
		}
	}
}

// FindStringInSlice takes []string and returns true if the supplied string is in the slice.
func FindStringInSlice(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

// ReadFileContents reads a file on the OS and returns its contents as a string
func ReadFileContents(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ReadFileContentType reads a file on the OS and returns its file type
func ReadFileContentType(filePath string) (string, error) {
	// Open File
	f, err := os.Open(filePath)
	if err != nil {
		log.Error().Msg(err.Error())
	}
	defer f.Close()

	// Only the first 512 bytes are used to sniff the content type.
	buffer := make([]byte, 512)

	_, err = f.Read(buffer)
	if err != nil {
		return "", err
	}

	// Use the net/http package's handy DectectContentType function. Always returns a valid
	// content-type by returning "application/octet-stream" if no others seemed to match.
	contentType := http.DetectContentType(buffer)

	return contentType, nil
}

// RemoveFromSlice accepts T as a comparable slice and removed the index at
// i - the returned value is the slice without the indexed entry
func RemoveFromSlice[T comparable](slice []T, i int) []T {
	slice[i] = slice[len(slice)-1]
	return slice[:len(slice)-1]
}

var BackupResolver = &net.Resolver{
	PreferGo: true,
	Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
		d := net.Dialer{
			Timeout: time.Millisecond * time.Duration(10000),
		}
		return d.DialContext(ctx, network, "8.8.8.8:53")
	},
}

// ScheduledGitopsCatalogUpdate
func ScheduledGitopsCatalogUpdate() {
	err := db.Client.UpdateGitopsCatalogApps()
	if err != nil {
		log.Warn().Msg(err.Error())
	}
	for range time.Tick(time.Minute * 30) {
		err := db.Client.UpdateGitopsCatalogApps()
		if err != nil {
			log.Warn().Msg(err.Error())
		}
	}
}

// ValidateAuthenticationFields checks a map[string]string returned from looking up an
// authentication Secret for missing fields
func ValidateAuthenticationFields(s map[string]string) error {
	for key, value := range s {
		if value == "" {
			return fmt.Errorf("field %s cannot be blank", key)
		}
	}
	return nil
}
