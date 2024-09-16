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

	"github.com/konstructio/kubefirst-api/internal/constants"
	"github.com/konstructio/kubefirst-api/internal/env"
	"github.com/konstructio/kubefirst-api/internal/k8s"
	"github.com/konstructio/kubefirst-api/internal/secrets"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	log "github.com/rs/zerolog/log"
)

// CreateK1Directory
func CreateK1Directory(clusterName string) {
	// Create k1 dir if it doesn't exist
	homePath, err := os.UserHomeDir()
	if err != nil {
		log.Info().Msg(fmt.Errorf("error getting user home directory: %w", err).Error())
		return
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
		return "", fmt.Errorf("error reading file contents from %s: %w", filePath, err)
	}
	return string(data), nil
}

// ReadFileContentType reads a file on the OS and returns its file type
func ReadFileContentType(filePath string) (string, error) {
	// Open File
	f, err := os.Open(filePath)
	if err != nil {
		log.Error().Msg(fmt.Errorf("error opening file %q: %w", filePath, err).Error())
		return "", fmt.Errorf("error opening file %q: %w", filePath, err)
	}
	defer f.Close()

	// Only the first 512 bytes are used to sniff the content type.
	buffer := make([]byte, 512)

	_, err = f.Read(buffer)
	if err != nil {
		return "", fmt.Errorf("error reading file %s: %w", filePath, err)
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
	Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
		d := net.Dialer{
			Timeout: time.Millisecond * time.Duration(10000),
		}
		return d.DialContext(ctx, network, "8.8.8.8:53")
	},
}

// ScheduledGitopsCatalogUpdate
func ScheduledGitopsCatalogUpdate() {
	kcfg := GetKubernetesClient("")

	err := secrets.UpdateGitopsCatalogApps(kcfg.Clientset)
	if err != nil {
		log.Warn().Msg(err.Error())
	}
	for range time.Tick(time.Minute * 30) {
		err := secrets.UpdateGitopsCatalogApps(kcfg.Clientset)
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

// GetKubernetesClient for cluster zero and existing cluster
func GetKubernetesClient(clusterName string) *k8s.KubernetesClient {
	// Get Environment variables
	env, _ := env.GetEnv(constants.SilenceGetEnv)

	// Create Kubernetes Client Context
	inCluster := env.InCluster
	homeDir, _ := os.UserHomeDir()
	clusterDir := fmt.Sprintf("%s/.k1/%s", homeDir, clusterName)
	kubeconfigPath := fmt.Sprintf("%s/kubeconfig", clusterDir)

	if env.K1LocalDebug {
		kubeconfigPath = env.K1LocalKubeconfigPath
	}

	kcfg, err := k8s.CreateKubeConfig(inCluster, kubeconfigPath)
	if err != nil {
		log.Error().Msg(fmt.Errorf("error creating kubeconfig: %w", err).Error())
	}

	return kcfg
}

func CreateKubefirstNamespaceIfNotExists(clientSet *kubernetes.Clientset) error {
	_, err := clientSet.CoreV1().Namespaces().Get(context.TODO(), "kubefirst", metav1.GetOptions{})
	if err != nil {
		namespace := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kubefirst"}}
		_, err = clientSet.CoreV1().Namespaces().Create(context.TODO(), namespace, metav1.CreateOptions{})
		if err != nil {
			log.Error().Err(fmt.Errorf("error creating namespace %s: %w", "kubefirst", err)).Msg("")
			return fmt.Errorf("error creating namespace %s: %w", "kubefirst", err)
		}
		log.Info().Msgf("namespace created: %s", "kubefirst")
	} else {
		log.Warn().Msgf("namespace %s already exists - skipping", "kubefirst")
	}

	return nil
}
