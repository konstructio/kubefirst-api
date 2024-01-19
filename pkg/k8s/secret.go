/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k8s

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/rs/zerolog/log"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// CreateSecretV2 creates a Kubernetes Secret
func CreateSecretV2(clientset *kubernetes.Clientset, secret *v1.Secret) error {
	_, err := clientset.CoreV1().Secrets(secret.Namespace).Create(
		context.Background(),
		secret,
		metav1.CreateOptions{},
	)
	if err != nil {
		return err
	}
	log.Info().Msgf("created Secret %s in Namespace %s\n", secret.Name, secret.Namespace)
	return nil
}

// ReadSecretV2 reads the content of a Kubernetes Secret
func ReadSecretV2(clientset *kubernetes.Clientset, namespace string, secretName string) (map[string]string, error) {
	secret, err := clientset.CoreV1().Secrets(namespace).Get(context.Background(), secretName, metav1.GetOptions{})
	if err != nil {
		log.Error().Msgf("error getting secret: %s\n", err)
		return map[string]string{}, err
	}

	parsedSecretData := make(map[string]string)
	for key, value := range secret.Data {
		parsedSecretData[key] = string(value)
	}

	return parsedSecretData, nil
}

// UpdateSecretV2 updates the key value pairs of a Kubernetes Secret
func UpdateSecretV2(clientset *kubernetes.Clientset, namespace string, secretName string, secretValues UpdateSecretArgs) error {
	// decode into json
	secretsToUpdate, err := json.Marshal(secretValues)
	if err != nil {
		return err
	}

	// create map to iterate over values to change
	secretsToUpdateMap := make(map[string]string)
	err = json.Unmarshal(secretsToUpdate, &secretsToUpdateMap)
	if err != nil {
		return err
	}

	currentSecret, err := clientset.CoreV1().Secrets(namespace).Get(context.Background(), secretName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	for key, newValue := range secretsToUpdateMap {
		curVal, exists := currentSecret.Data[key]
		byteNewVal := []byte(newValue)

		if exists && !bytes.Equal(curVal, byteNewVal) {
			currentSecret.Data[key] = byteNewVal
		}
	}

	_, err = clientset.CoreV1().Secrets(currentSecret.Namespace).Update(
		context.Background(),
		currentSecret,
		metav1.UpdateOptions{},
	)

	if err != nil {
		return err
	}

	log.Info().Msgf("updated Secret %s in Namespace %s\n", currentSecret.Name, currentSecret.Namespace)
	return nil
}
