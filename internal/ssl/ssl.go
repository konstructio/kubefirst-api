/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package ssl

import (
	"context"
	"fmt"
	"os"
	"strings"

	// cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	// cm "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned"
	"github.com/rs/zerolog/log"

	pkg "github.com/konstructio/kubefirst-api/internal"
	"github.com/konstructio/kubefirst-api/internal/k8s"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/applyconfigurations/core/v1"
	"sigs.k8s.io/yaml"
)

func Restore(backupDir, kubeconfigPath string) error {
	sslSecretFiles, err := os.ReadDir(backupDir + "/secrets")
	if err != nil {
		return fmt.Errorf("error reading directory: %w", err)
	}

	clientset, err := k8s.GetClientSet(kubeconfigPath)
	if err != nil {
		return fmt.Errorf("error getting clientset: %w", err)
	}

	for _, secret := range sslSecretFiles {
		// file is named with convention $namespace-$secretName.yaml
		//  todo link to backup source code
		namespace := strings.Split(secret.Name(), "-")[0]
		log.Info().Msg("creating secret: " + secret.Name())

		f, err := os.ReadFile(backupDir + "/secrets/" + secret.Name())
		if err != nil {
			return fmt.Errorf("error reading file %q: %w", secret.Name(), err)
		}

		secretData := &v1.SecretApplyConfiguration{}

		err = yaml.Unmarshal(f, secretData)
		if err != nil {
			return fmt.Errorf("error unmarshalling yaml: %w", err)
		}

		sec, err := clientset.CoreV1().Secrets(namespace).Apply(context.Background(), secretData, metav1.ApplyOptions{FieldManager: "application/apply-patch"})
		if err != nil {
			return fmt.Errorf("error applying secret: %w", err)
		}
		log.Info().Msgf("created secret: %s", sec.Name)
	}
	return nil
}

func Backup(backupDir, kubeconfigPath string) error {
	clientset, err := k8s.GetClientSet(kubeconfigPath)
	if err != nil {
		return fmt.Errorf("error getting clientset: %w", err)
	}

	// * corev1 secret resources
	secrets, err := clientset.CoreV1().Secrets("").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing secrets: %w", err)
	}

	for _, secret := range secrets.Items {
		if strings.Contains(secret.Name, "-tls") {
			log.Info().Msg("backing up secret (ns/resource): " + secret.Namespace + "/" + secret.Name)

			// modify fields of secret for restore
			secret.APIVersion = "v1"
			secret.Kind = "Secret"
			secret.SetManagedFields(nil)
			secret.SetOwnerReferences(nil)
			secret.SetAnnotations(nil)
			secret.SetCreationTimestamp(metav1.Time{})
			secret.SetResourceVersion("")
			secret.SetUID("")

			fileName := fmt.Sprintf("%s/%s-%s.yaml", backupDir+"/secrets", secret.Namespace, secret.Name)
			log.Info().Msgf("writing file: %q", fileName)
			yamlContent, err := yaml.Marshal(secret)
			if err != nil {
				return fmt.Errorf("unable to marshal yaml: %w", err)
			}
			if err := pkg.CreateFile(fileName, yamlContent); err != nil {
				return fmt.Errorf("error creating file: %w", err)
			}
		} else {
			log.Info().Msgf("skipping secret: %s", secret.Name)
		}
	}
	return nil
}
