/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k8s

import (
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
func ReadSecretV2(clientset *kubernetes.Clientset, namespace string, secretName string) (map[string]interface{}, error) {
	secret, err := clientset.CoreV1().Secrets(namespace).Get(context.Background(), secretName, metav1.GetOptions{})
	if err != nil {
		log.Warn().Msgf("no secret found: %s\n", err)
		return map[string]interface{}{}, err
	}

	parsedSecretData := make(map[string]interface{})
	for key, value := range secret.Data {
		parsedSecretData[key] = string(value)
	}

	return parsedSecretData, nil
}

// DeleteSecretV2 reads the content of a Kubernetes Secret
func DeleteSecretV2(clientset *kubernetes.Clientset, namespace string, secretName string) error {
	err := clientset.CoreV1().Secrets(namespace).Delete(context.Background(), secretName, metav1.DeleteOptions{})
	if err != nil {
		log.Error().Msgf("error deleting secret: %s\n", err)
		return err
	}
	return nil
}

// UpdateSecretV2 updates the key value pairs of a Kubernetes Secret
func UpdateSecretV2(clientset *kubernetes.Clientset, namespace string, secretName string, secretValues map[string][]byte) error {
	currentSecret, err := clientset.CoreV1().Secrets(namespace).Get(context.Background(), secretName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	currentSecret.Data = secretValues

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

func mapToJSON(data map[string]interface{}) (string, error) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return "", err // Handle error.
	}
	return string(jsonBytes), nil
}
