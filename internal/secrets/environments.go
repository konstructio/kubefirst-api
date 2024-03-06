package secrets

import (
	"encoding/json"
	"fmt"

	"github.com/kubefirst/kubefirst-api/internal/types"
	"github.com/kubefirst/kubefirst-api/pkg/k8s"
	pkgtypes "github.com/kubefirst/kubefirst-api/pkg/types"
	log "github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const KUBEFIRST_ENVIRONMENTS_SECRET_NAME = "kubefirst-environments"

// GetEnvironments
func GetEnvironments(clientSet *kubernetes.Clientset) ([]pkgtypes.Environment, error) {
	environmentList := []pkgtypes.Environment{}

	kubefirstSecrets, err := k8s.ReadSecretV2(clientSet, "kubefirst", KUBEFIRST_ENVIRONMENTS_SECRET_NAME)

	jsonString, err := MapToStructuredJSON(kubefirstSecrets)

	jsonData, err := json.Marshal(jsonString)
	if err != nil {
		return environmentList, fmt.Errorf("error marshalling json: %s", err)
	}

	err = json.Unmarshal([]byte(jsonData), &environmentList)
	if err != nil {
		return environmentList, fmt.Errorf("unable to cast environment: %s", err)
	}

	return environmentList, nil
}

// GetEnvironment
func GetEnvironment(clientSet *kubernetes.Clientset, name string) (pkgtypes.Environment, error) {
	environmentList := []pkgtypes.Environment{}
	environment := pkgtypes.Environment{}

	kubefirstSecrets, err := k8s.ReadSecretV2(clientSet, "kubefirst", KUBEFIRST_ENVIRONMENTS_SECRET_NAME)

	jsonString, err := MapToStructuredJSON(kubefirstSecrets)

	jsonData, err := json.Marshal(jsonString)
	if err != nil {
		return environment, fmt.Errorf("error marshalling json %s: %s", name, err)
	}

	err = json.Unmarshal([]byte(jsonData), &environmentList)
	if err != nil {
		return environment, fmt.Errorf("unable to cast cluster %s: %s", name, err)
	}

	for _, environmentItem := range environmentList {
		if environmentItem.Name == name {
			environment = environmentItem
		}
	}

	return environment, nil
}

// InsertEnvironment
func InsertEnvironment(clientSet *kubernetes.Clientset, env pkgtypes.Environment) (pkgtypes.Environment, error) {
	enviroments, _ := GetEnvironments(clientSet)
	environment := pkgtypes.Environment{
		ID:                primitive.NewObjectID(),
		Name:              env.Name,
		Color:             env.Color,
		Description:       env.Description,
		CreationTimestamp: env.CreationTimestamp,
	}

	if len(enviroments) == 0 {
		enviroments = append(enviroments, environment)
		bytes, _ := json.Marshal(enviroments)
		secretValuesMap, _ := ParseJSONToMap(string(bytes))

		secretToCreate := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      KUBEFIRST_ENVIRONMENTS_SECRET_NAME,
				Namespace: "kubefirst",
			},
			Data: secretValuesMap,
		}

		err := k8s.CreateSecretV2(clientSet, secretToCreate)

		if err != nil {
			return environment, fmt.Errorf("error creating kubernetes environment secret: %s", err)
		}

		return environment, nil
	}

	clusterList := append(enviroments, environment)
	bytes, _ := json.Marshal(clusterList)
	secretValuesMap, _ := ParseJSONToMap(string(bytes))

	err := k8s.UpdateSecretV2(clientSet, "kubefirst", KUBEFIRST_ENVIRONMENTS_SECRET_NAME, secretValuesMap)

	if err != nil {
		return environment, fmt.Errorf("error creating kubernetes secret: %s", err)
	}

	return environment, nil
}

func DeleteEnvironment(clientSet *kubernetes.Clientset, envId string) error {
	objectId, _ := primitive.ObjectIDFromHex(envId)
	environmentList, err := GetEnvironments(clientSet)
	filteredEnvironmentList := []pkgtypes.Environment{}

	for _, environment := range environmentList {
		if environment.ID != objectId {
			filteredEnvironmentList = append(filteredEnvironmentList, environment)
		}
	}

	bytes, err := json.Marshal(filteredEnvironmentList)
	secretValuesMap, _ := ParseJSONToMap(string(bytes))

	err = k8s.UpdateSecretV2(clientSet, "kubefirst", KUBEFIRST_ENVIRONMENTS_SECRET_NAME, secretValuesMap)
	if err != nil {
		return fmt.Errorf("error deleting environments %s: %s", envId, err)
	}

	log.Info().Msgf("environment deleted: %v", envId)

	return nil
}

func UpdateEnvironment(clientSet *kubernetes.Clientset, id string, env types.EnvironmentUpdateRequest) error {
	objectId, _ := primitive.ObjectIDFromHex(id)
	environments, _ := GetEnvironments(clientSet)

	for _, environmentItem := range environments {
		if environmentItem.ID == objectId {
			environmentItem.Color = env.Color
			environmentItem.Description = env.Description
		}
	}

	bytes, _ := json.Marshal(environments)
	secretValuesMap, _ := ParseJSONToMap(string(bytes))

	err := k8s.UpdateSecretV2(clientSet, "kubefirst", KUBEFIRST_ENVIRONMENTS_SECRET_NAME, secretValuesMap)

	if err != nil {
		return fmt.Errorf("error creating kubernetes secret: %s", err)
	}

	return nil
}
