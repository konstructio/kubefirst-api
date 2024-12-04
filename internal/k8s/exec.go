/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k8s

import (
	"context"
	"errors"
	"fmt"
	"net"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

// CreateSecretV2 creates a Kubernetes Secret
func CreateSecretV2(clientset kubernetes.Interface, secret *v1.Secret) error {
	_, err := clientset.CoreV1().Secrets(secret.Namespace).Create(
		context.Background(),
		secret,
		metav1.CreateOptions{},
	)
	if err != nil {
		return fmt.Errorf("error creating Secret %q in Namespace %q: %w", secret.Name, secret.Namespace, err)
	}
	log.Info().Msgf("created Secret %s in Namespace %s", secret.Name, secret.Namespace)
	return nil
}

// ReadConfigMapV2 reads the content of a Kubernetes ConfigMap
func ReadConfigMapV2(kubeConfigPath, namespace, configMapName string) (map[string]string, error) {
	clientset, err := GetClientSet(kubeConfigPath)
	if err != nil {
		return nil, fmt.Errorf("error getting client set from kubeConfigPath %q: %w", kubeConfigPath, err)
	}
	configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.Background(), configMapName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting ConfigMap %q in Namespace %q: %w", configMapName, namespace, err)
	}

	parsedSecretData := make(map[string]string)
	for key, value := range configMap.Data {
		parsedSecretData[key] = value
	}

	return parsedSecretData, nil
}

// ReadSecretV2 reads the content of a Kubernetes Secret
func ReadSecretV2(clientset kubernetes.Interface, namespace, secretName string) (map[string]string, error) {
	secret, err := clientset.CoreV1().Secrets(namespace).Get(context.Background(), secretName, metav1.GetOptions{})
	if err != nil {
		log.Warn().Msgf("no secret found: %s", err)
		return nil, fmt.Errorf("error getting Secret %q in Namespace %q: %w", secretName, namespace, err)
	}

	parsedSecretData := make(map[string]string)
	for key, value := range secret.Data {
		parsedSecretData[key] = string(value)
	}

	return parsedSecretData, nil
}

// ReadService reads a Kubernetes Service object
func ReadService(kubeConfigPath, namespace, serviceName string) (*v1.Service, error) {
	clientset, err := GetClientSet(kubeConfigPath)
	if err != nil {
		return &v1.Service{}, fmt.Errorf("error getting client set from kubeConfigPath %q: %w", kubeConfigPath, err)
	}

	service, err := clientset.CoreV1().Services(namespace).Get(context.Background(), serviceName, metav1.GetOptions{})
	if err != nil {
		log.Error().Msgf("error getting Service %q in Namespace %q: %s", serviceName, namespace, err)
		return &v1.Service{}, fmt.Errorf("error getting Service %q in Namespace %q: %w", serviceName, namespace, err)
	}

	return service, nil
}

func ReturnDeploymentObject(client kubernetes.Interface, matchLabel string, matchLabelValue string, namespace string, timeoutSeconds int) (*appsv1.Deployment, error) {
	timeout := time.Duration(timeoutSeconds) * time.Second
	var deployment *appsv1.Deployment

	err := wait.PollUntilContextTimeout(context.Background(), 5*time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		deployments, err := client.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", matchLabel, matchLabelValue),
		})
		if err != nil {
			// if we couldn't connect, ask to try again
			if isNetworkingError(err) {
				return false, nil
			}

			// if we got an error, return it
			return false, fmt.Errorf("error getting Deployment: %w", err)
		}

		// if we couldn't find any deployments, ask to try again
		if len(deployments.Items) == 0 {
			return false, nil
		}

		// fetch the first item from the list matching the labels
		deployment = &deployments.Items[0]

		// Check if it has at least one replica, if not, ask to try again
		if deployment.Status.Replicas == 0 {
			return false, nil
		}

		// if we found a deployment, return it
		return true, nil
	})
	if err != nil {
		return nil, fmt.Errorf("error waiting for Deployment: %w", err)
	}

	return deployment, nil
}

// ReturnPodObject returns a matching v1.Pod object based on the filters
func ReturnPodObject(kubeConfigPath, matchLabel, matchLabelValue, namespace string, timeoutSeconds int) (*v1.Pod, error) {
	clientset, err := GetClientSet(kubeConfigPath)
	if err != nil {
		return nil, fmt.Errorf("error getting client set from kubeConfigPath %q: %w", kubeConfigPath, err)
	}

	labelSelector := fmt.Sprintf("%s=%s", matchLabel, matchLabelValue)
	log.Info().Msgf("waiting for pod with label %s=%s to be created in namespace %q", matchLabel, matchLabelValue, namespace)

	var pod *v1.Pod

	err = wait.PollUntilContextTimeout(context.Background(), 5*time.Second, time.Duration(timeoutSeconds)*time.Second, true, func(ctx context.Context) (bool, error) {
		podList, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: labelSelector,
		})
		if err != nil {
			// If we couldn't connect, retry
			if isNetworkingError(err) {
				log.Warn().Msgf("connection error, retrying: %s", err.Error())
				return false, nil
			}

			// For other errors, log and return the error to stop polling
			log.Error().Msgf("error listing Pods: %v", err)
			return false, fmt.Errorf("error listing pods: %w", err)
		}

		if len(podList.Items) == 0 {
			// No Pods found, continue polling
			return false, nil
		}

		pod = &podList.Items[0]
		if pod.Status.Phase == v1.PodPending || pod.Status.Phase == v1.PodRunning {
			// Pod is in the desired state
			return true, nil
		}

		// Pod is not yet in the desired state, continue polling
		return false, nil
	})
	if err != nil {
		log.Error().Msg("the pod was not created within the timeout period")
		return nil, fmt.Errorf("the Pod %q in Namespace %q was not created within the timeout period: %w", matchLabelValue, namespace, err)
	}

	return pod, nil
}

// ReturnStatefulSetObject returns a matching appsv1.StatefulSet object based on the filters
func ReturnStatefulSetObject(clientset kubernetes.Interface, matchLabel, matchLabelValue, namespace string, timeoutSeconds int) (*appsv1.StatefulSet, error) {
	labelSelector := fmt.Sprintf("%s=%s", matchLabel, matchLabelValue)
	log.Info().Msgf("waiting for StatefulSet with label %s=%s to be created in namespace %q", matchLabel, matchLabelValue, namespace)

	var statefulSet *appsv1.StatefulSet

	err := wait.PollUntilContextTimeout(context.Background(), 5*time.Second, time.Duration(timeoutSeconds)*time.Second, true, func(ctx context.Context) (bool, error) {
		statefulSets, err := clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: labelSelector,
		})
		if err != nil {
			// If we couldn't connect, retry
			if isNetworkingError(err) {
				log.Warn().Msgf("connection error, retrying: %s", err.Error())
				return false, nil
			}

			// For other errors, log and return the error to stop polling
			log.Error().Msgf("error listing StatefulSets: %v", err)
			return false, fmt.Errorf("error listing statefulsets: %w", err)
		}

		if len(statefulSets.Items) == 0 {
			// No StatefulSets found, continue polling
			return false, nil
		}

		sts := &statefulSets.Items[0]
		if sts.Status.Replicas > 0 {
			statefulSet = sts
			return true, nil
		}

		// StatefulSet does not have replicas yet, continue polling
		return false, nil
	})
	if err != nil {
		log.Error().Msg("the StatefulSet was not created within the timeout period")
		return nil, fmt.Errorf("the StatefulSet %q in Namespace %q was not created within the timeout period: %w", matchLabelValue, namespace, err)
	}

	return statefulSet, nil
}

// WaitForDeploymentReady waits for a target Deployment to become ready
func WaitForDeploymentReady(clientset kubernetes.Interface, deployment *appsv1.Deployment, timeoutSeconds int) (bool, error) {
	deploymentName := deployment.Name
	namespace := deployment.Namespace

	// Get the desired number of replicas from the deployment spec
	if deployment.Spec.Replicas == nil {
		log.Error().Msgf("deployment %q in namespace %q has nil spec.replicas field", deploymentName, namespace)
		return false, fmt.Errorf("deployment %q in Namespace %q has nil Spec.Replicas", deploymentName, namespace)
	}
	desiredReplicas := *deployment.Spec.Replicas

	log.Info().Msgf("waiting for deployment %q in namespace %q to be ready - this could take up to %v seconds", deploymentName, namespace, timeoutSeconds)

	err := wait.PollUntilContextTimeout(context.Background(), 5*time.Second, time.Duration(timeoutSeconds)*time.Second, true, func(ctx context.Context) (bool, error) {
		// Get the latest Deployment object
		currentDeployment, err := clientset.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
		if err != nil {
			// If we couldn't connect, retry
			if isNetworkingError(err) {
				log.Warn().Msgf("connection error, retrying: %s", err.Error())
				return false, nil
			}

			// For other errors, log and return the error to stop polling
			log.Error().Msgf("error when getting deployment %q in namespace %q: %v", deploymentName, namespace, err)
			return false, fmt.Errorf("error listing statefulsets: %w", err)
		}

		if currentDeployment.Status.ReadyReplicas == desiredReplicas {
			log.Info().Msgf("all pods in deployment %q are ready", deploymentName)
			return true, nil
		}

		// Deployment is not yet ready, continue polling
		return false, nil
	})
	if err != nil {
		log.Error().Msgf("the deployment %q in namespace %q was not ready within the timeout period", deploymentName, namespace)
		return false, fmt.Errorf("the Deployment %q in Namespace %q was not ready within the timeout period: %w", deploymentName, namespace, err)
	}

	return true, nil
}

// WaitForPodReady waits for a target Pod to become ready
func WaitForPodReady(clientset kubernetes.Interface, pod *v1.Pod, timeoutSeconds int) (bool, error) {
	podName := pod.Name
	namespace := pod.Namespace

	log.Info().Msgf("waiting for pod %q in namespace %q to be ready - this could take up to %v seconds", podName, namespace, timeoutSeconds)

	err := wait.PollUntilContextTimeout(context.Background(), 5*time.Second, time.Duration(timeoutSeconds)*time.Second, true, func(ctx context.Context) (bool, error) {
		// Get the latest Pod object
		currentPod, err := clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			// If we couldn't connect, retry
			if isNetworkingError(err) {
				log.Warn().Msgf("connection error, retrying: %s", err.Error())
				return false, nil
			}

			// For other errors, log and return the error to stop polling
			log.Error().Msgf("error getting pod %q in namespace %q: %v", podName, namespace, err)
			return false, fmt.Errorf("error listing pods: %w", err)
		}

		if currentPod.Status.Phase == v1.PodRunning {
			log.Info().Msgf("pod %q has status %q", podName, currentPod.Status.Phase)
			return true, nil
		}

		// Pod is not yet ready, continue polling
		return false, nil
	})
	if err != nil {
		log.Error().Msgf("the operation timed out while waiting for pod %q in namespace %q to become ready", podName, namespace)
		return false, fmt.Errorf("the operation timed out while waiting for Pod %q in Namespace %q: %w", podName, namespace, err)
	}

	return true, nil
}

// WaitForStatefulSetReady waits for a target StatefulSet to become ready
func WaitForStatefulSetReady(clientset kubernetes.Interface, statefulset *appsv1.StatefulSet, timeoutSeconds int, ignoreReady bool) (bool, error) {
	statefulSetName := statefulset.Name
	namespace := statefulset.Namespace

	// Get the desired number of replicas from the StatefulSet spec
	if statefulset.Spec.Replicas == nil {
		log.Error().Msgf("statefulSet %q in namespace %s has nil spec.replicas", statefulSetName, namespace)
		return false, fmt.Errorf("StatefulSet %q in Namespace %q has nil Spec.Replicas", statefulSetName, namespace)
	}
	desiredReplicas := *statefulset.Spec.Replicas

	log.Info().Msgf("waiting for statefulset %q in namespace %q to be ready - this could take up to %v seconds", statefulSetName, namespace, timeoutSeconds)

	err := wait.PollUntilContextTimeout(context.Background(), 5*time.Second, time.Duration(timeoutSeconds)*time.Second, true, func(ctx context.Context) (bool, error) {
		// Get the latest StatefulSet object
		currentStatefulSet, err := clientset.AppsV1().StatefulSets(namespace).Get(ctx, statefulSetName, metav1.GetOptions{})
		if err != nil {
			// If we couldn't connect, retry
			if isNetworkingError(err) {
				log.Warn().Msgf("connection error, retrying: %s", err.Error())
				return false, nil
			}

			// For other errors, log and return the error to stop polling
			log.Error().Msgf("error when getting statefulset %q in namespace %s: %v", statefulSetName, namespace, err)
			return false, fmt.Errorf("error listing statefulsets: %w", err)
		}

		if ignoreReady {
			// Check if CurrentReplicas match desired replicas
			if currentStatefulSet.Status.CurrentReplicas == desiredReplicas {
				currentRevision := currentStatefulSet.Status.CurrentRevision

				// Get Pods owned by the StatefulSet
				pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
					LabelSelector: fmt.Sprintf("controller-revision-hash=%s", currentRevision),
				})
				if err != nil {
					// If we couldn't connect, retry
					if isNetworkingError(err) {
						log.Warn().Msg("connection refused while listing pods, retrying...")
						return false, nil
					}

					log.Error().Msgf("could not find pods owned by statefulset %q in namespace %q: %v", statefulSetName, namespace, err)
					return false, fmt.Errorf("error listing statefulsets: %w", err)
				}

				// Check if all Pods are in Running phase
				allRunning := true
				for _, pod := range pods.Items {
					if pod.Status.Phase != v1.PodRunning {
						allRunning = false
						break
					}
				}

				if allRunning {
					log.Info().Msgf("all pods in statefulset %q are running", statefulSetName)
					return true, nil
				}
			}
		} else {
			// Check if ReadyReplicas match desired replicas
			if currentStatefulSet.Status.ReadyReplicas == desiredReplicas {
				log.Info().Msgf("all pods in statefulset %q are ready", statefulSetName)
				return true, nil
			}
		}

		// Continue polling
		return false, nil
	})
	if err != nil {
		log.Error().Msgf("the statefulset %q in namespace %q was not ready within the timeout period", statefulSetName, namespace)
		return false, fmt.Errorf("the StatefulSet %q in Namespace %q was not ready within the timeout period: %w", statefulSetName, namespace, err)
	}

	return true, nil
}

// isNetworkingError checks if the error is a networking error
// that could be due to the cluster not being ready yet. It's the
// responsibility of the caller to decide if these errors are fatal
// or if they should be retried.
func isNetworkingError(err error) bool {
	// Check if the error is a networking error, which could be
	// when the cluster is starting up or when the network pieces
	// aren't yet ready
	if errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ETIMEDOUT) {
		return true
	}

	// Check if the error is a timeout error
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	return false
}
