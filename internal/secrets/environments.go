package secrets

import (
	"encoding/json"
	"fmt"

	"github.com/kubefirst/kubefirst-api/internal/k8s"
	"github.com/kubefirst/kubefirst-api/internal/types"
	pkgtypes "github.com/kubefirst/kubefirst-api/pkg/types"
	log "github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const KUBEFIRST_ENVIRONMENTS_SECRET_NAME = "kubefirst-environments"
const KUBEFIRST_ENVIRONMENT_PREFIX = "kubefirst-environment"

// GetEnvironments
func GetEnvironments(clientSet *kubernetes.Clientset) ([]pkgtypes.Environment, error) {
	environmentList := []pkgtypes.Environment{}
	environmentReferenceList, _ := GetSecretReference(clientSet, KUBEFIRST_ENVIRONMENTS_SECRET_NAME)
	for _, environmentName := range environmentReferenceList.List {
		environment, _ := GetEnvironment(clientSet, environmentName)
		if environment.Name != "" {
			environmentList = append(environmentList, environment)
		}
	}

	return environmentList, nil
}

// GetEnvironment
func GetEnvironment(clientSet *kubernetes.Clientset, name string) (pkgtypes.Environment, error) {
	environment := pkgtypes.Environment{}

	kubefirstSecrets, _ := k8s.ReadSecretV2Old(clientSet, "kubefirst", fmt.Sprintf("%s-%s", KUBEFIRST_ENVIRONMENT_PREFIX, name))
	jsonString, _ := MapToStructuredJSON(kubefirstSecrets)

	jsonData, err := json.Marshal(jsonString)
	if err != nil {
		return environment, fmt.Errorf("error marshalling json %s: %s", name, err)
	}

	err = json.Unmarshal([]byte(jsonData), &environment)
	if err != nil {
		return environment, fmt.Errorf("unable to cast environment %s: %s", name, err)
	}

	return environment, nil
}

// InsertEnvironment
func InsertEnvironment(clientSet *kubernetes.Clientset, env pkgtypes.Environment) (pkgtypes.Environment, error) {
	environment := pkgtypes.Environment{
		ID:                primitive.NewObjectID(),
		Name:              env.Name,
		Color:             env.Color,
		Description:       env.Description,
		CreationTimestamp: env.CreationTimestamp,
	}

	err := AddSecretReferenceItem(clientSet, KUBEFIRST_ENVIRONMENTS_SECRET_NAME, env.Name)
	if err != nil {
		return environment, err
	}

	bytes, _ := json.Marshal(environment)
	secretValuesMap, _ := ParseJSONToMap(string(bytes))

	secretToCreate := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", KUBEFIRST_ENVIRONMENT_PREFIX, env.Name),
			Namespace: "kubefirst",
		},
		Data: secretValuesMap,
	}

	err = k8s.CreateSecretV2(clientSet, secretToCreate)

	if err != nil {
		return environment, fmt.Errorf("error creating kubernetes environment secret: %s", err)
	}

	return environment, nil
}

func DeleteEnvironment(clientSet *kubernetes.Clientset, envId string) error {
	objectId, _ := primitive.ObjectIDFromHex(envId)
	environmentSecretReference, _ := GetSecretReference(clientSet, KUBEFIRST_ENVIRONMENTS_SECRET_NAME)
	environmentToDelete := pkgtypes.Environment{}

	for _, environmentName := range environmentSecretReference.List {
		environment, _ := GetEnvironment(clientSet, environmentName)

		if environment.ID == objectId {
			environmentToDelete = environment
		}
	}

	err := DeleteSecretReference(clientSet, KUBEFIRST_ENVIRONMENTS_SECRET_NAME, environmentToDelete.Name)
	if err != nil {
		return fmt.Errorf("error deleting environment %s reference", environmentToDelete.Name)
	}

	err = k8s.DeleteSecretV2(clientSet, "kubefirst", fmt.Sprintf("%s-%s", KUBEFIRST_ENVIRONMENT_PREFIX, environmentToDelete.Name))
	if err != nil {
		return fmt.Errorf("error deleting environment %s: %s", environmentToDelete.Name, err)
	}

	log.Info().Msgf("environment deleted: %v", environmentToDelete.Name)

	return nil
}

func UpdateEnvironment(clientSet *kubernetes.Clientset, id string, env types.EnvironmentUpdateRequest) error {
	objectId, _ := primitive.ObjectIDFromHex(id)
	environmentSecretReference, _ := GetSecretReference(clientSet, KUBEFIRST_ENVIRONMENTS_SECRET_NAME)
	environmentToUpdate := pkgtypes.Environment{}

	for _, environmentName := range environmentSecretReference.List {
		environment, _ := GetEnvironment(clientSet, environmentName)

		if environment.ID == objectId {
			environmentToUpdate = environment
		}
	}

	environmentToUpdate.Color = env.Color
	environmentToUpdate.Description = env.Description

	bytes, _ := json.Marshal(environmentToUpdate)
	secretValuesMap, _ := ParseJSONToMap(string(bytes))

	err := k8s.UpdateSecretV2(clientSet, "kubefirst", fmt.Sprintf("%s-%s", KUBEFIRST_ENVIRONMENT_PREFIX, environmentToUpdate.Name), secretValuesMap)

	if err != nil {
		return fmt.Errorf("error creating kubernetes secret: %s", err)
	}

	return nil
}
