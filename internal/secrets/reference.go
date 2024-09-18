/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package secrets

import (
	"encoding/json"
	"fmt"

	"github.com/konstructio/kubefirst-api/internal/k8s"
	pkgtypes "github.com/konstructio/kubefirst-api/pkg/types"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func GetSecretReference(clientSet kubernetes.Interface, secretName string) (*pkgtypes.SecretListReference, error) {
	kubefirstSecrets, err := k8s.ReadSecretV2Old(clientSet, "kubefirst", secretName)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch secret: %w", err)
	}

	jsonString, _ := MapToStructuredJSON(kubefirstSecrets)

	jsonData, err := json.Marshal(jsonString)
	if err != nil {
		return nil, fmt.Errorf("error marshalling json: %w", err)
	}

	var secretReference pkgtypes.SecretListReference
	if err := json.Unmarshal(jsonData, &secretReference); err != nil {
		return nil, fmt.Errorf("unable to cast secret reference: %w", err)
	}

	if secretReference.Name == "" {
		return nil, apierrors.NewNotFound(v1.Resource("secrets"), secretName)
	}

	return &secretReference, nil
}

func DeleteSecretReference(clientSet kubernetes.Interface, secretName string, valueToDelete string) error {
	filteredReferenceList := pkgtypes.SecretListReference{}
	referenceList, err := GetSecretReference(clientSet, secretName)
	if err != nil {
		return fmt.Errorf("unable to get secret reference %s: %w", secretName, err)
	}

	filteredReferenceList.Name = referenceList.Name

	for _, referenceClusterName := range referenceList.List {
		if referenceClusterName != valueToDelete {
			filteredReferenceList.List = append(filteredReferenceList.List, referenceClusterName)
		}
	}

	err = UpdateSecretReference(clientSet, secretName, filteredReferenceList)
	if err != nil {
		return err
	}

	return nil
}

// UpdateSecretReference
func UpdateSecretReference(clientSet kubernetes.Interface, secretName string, secretReference pkgtypes.SecretListReference) error {
	bytes, err := json.Marshal(secretReference)
	if err != nil {
		return fmt.Errorf("error marshalling json: %w", err)
	}

	secretValuesMap, err := ParseJSONToMap(string(bytes))
	if err != nil {
		return fmt.Errorf("error parsing json to map: %w", err)
	}

	err = k8s.UpdateSecretV2(clientSet, "kubefirst", secretName, secretValuesMap)
	if err != nil {
		return fmt.Errorf("error updating secret reference: %w", err)
	}

	return nil
}

func UpsertSecretReference(clientSet kubernetes.Interface, secretName string, secretReference pkgtypes.SecretListReference) error {
	bytes, err := json.Marshal(secretReference)
	if err != nil {
		return fmt.Errorf("error marshalling json: %w", err)
	}

	secretValuesMap, err := ParseJSONToMap(string(bytes))
	if err != nil {
		return fmt.Errorf("error parsing json to map: %w", err)
	}

	secretToCreate := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: "kubefirst",
		},
		Data: secretValuesMap,
	}

	err = k8s.CreateSecretV2(clientSet, secretToCreate)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return UpdateSecretReference(clientSet, secretName, secretReference)
		}

		return fmt.Errorf("error creating secret reference: %w", err)
	}

	return nil
}

func AddSecretReferenceItem(clientSet kubernetes.Interface, secretName string, valueToAdd string) error {
	secretReference, err := GetSecretReference(clientSet, secretName)
	if err != nil {
		return fmt.Errorf("unable to get secret reference %s: %w", secretName, err)
	}

	secretReference.List = append(secretReference.List, valueToAdd)

	err = UpdateSecretReference(clientSet, secretName, *secretReference)
	if err != nil {
		return fmt.Errorf("unable to update secret reference %s: %w", secretName, err)
	}

	return nil
}
