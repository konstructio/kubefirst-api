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
	"io"
	"os"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/term"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

// CreateSecretV2 creates a Kubernetes Secret
func CreateSecretV2(clientset *kubernetes.Clientset, secret *v1.Secret) error {
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
func ReadSecretV2(clientset *kubernetes.Clientset, namespace, secretName string) (map[string]string, error) {
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

// PodExecSession executes a command against a Pod
func PodExecSession(kubeConfigPath string, p *PodSessionOptions, silent bool) error {
	// v1.PodExecOptions is passed to the rest client to form the req URL
	podExecOptions := v1.PodExecOptions{
		Stdin:   p.Stdin,
		Stdout:  p.Stdout,
		Stderr:  p.Stderr,
		TTY:     p.TtyEnabled,
		Command: p.Command,
	}

	err := podExec(kubeConfigPath, p, podExecOptions, silent)
	if err != nil {
		return fmt.Errorf("error executing command in Pod %q: %w", p.PodName, err)
	}
	return nil
}

// podExec performs kube-exec on a Pod with a given command
func podExec(kubeConfigPath string, ps *PodSessionOptions, pe v1.PodExecOptions, silent bool) error {
	clientset, err := GetClientSet(kubeConfigPath)
	if err != nil {
		return fmt.Errorf("error getting client set from kubeConfigPath %q: %w", kubeConfigPath, err)
	}

	config, err := GetClientConfig(kubeConfigPath)
	if err != nil {
		return fmt.Errorf("error getting client config from kubeConfigPath %q: %w", kubeConfigPath, err)
	}

	// Format the request to be sent to the API
	req := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(ps.PodName).
		Namespace(ps.Namespace).
		SubResource("exec")
	req.VersionedParams(&pe, scheme.ParameterCodec)

	// POST op against Kubernetes API to initiate remote command
	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		log.Error().Msgf("error executing command on Pod %s in Namespace %s: %s", ps.PodName, ps.Namespace, err)
		return fmt.Errorf("error executing command on Pod %q in Namespace %q: %w", ps.PodName, ps.Namespace, err)
	}

	// Put the terminal into raw mode to prevent it echoing characters twice
	oldState, err := term.MakeRaw(0)
	if err != nil {
		log.Error().Msgf("error when attempting to start terminal: %s", err)
		return fmt.Errorf("error when attempting to start terminal: %w", err)
	}
	defer term.Restore(0, oldState)

	var showOutput io.Writer
	if silent {
		showOutput = io.Discard
	} else {
		showOutput = os.Stdout
	}
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  os.Stdin,
		Stdout: showOutput,
		Stderr: os.Stderr,
		Tty:    ps.TtyEnabled,
	})
	if err != nil {
		log.Error().Msgf("error streaming pod command in Pod %s: %s", ps.PodName, err)
		return fmt.Errorf("error streaming pod command in Pod %q: %w", ps.PodName, err)
	}
	return nil
}

func ReturnDeploymentObject(client kubernetes.Interface, matchLabel string, matchLabelValue string, namespace string, timeoutSeconds int) (*appsv1.Deployment, error) {
	timeout := time.Duration(timeoutSeconds) * time.Second
	var deployment *appsv1.Deployment

	err := wait.PollImmediate(15*time.Second, timeout, func() (bool, error) {
		deployments, err := client.AppsV1().Deployments(namespace).List(context.Background(), metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", matchLabel, matchLabelValue),
		})
		if err != nil {
			// if we couldn't connect, ask to try again
			if errors.Is(err, syscall.ECONNREFUSED) {
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
	log.Info().Msgf("waiting for Pod with label %s=%s to be created in namespace %q", matchLabel, matchLabelValue, namespace)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Error().Msg("Timeout waiting for Pod to be created")
			return nil, fmt.Errorf("the Pod %q in Namespace %q was not created within the timeout period", matchLabelValue, namespace)
		case <-ticker.C:
			podList, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
				LabelSelector: labelSelector,
			})
			if err != nil {
				log.Error().Msgf("Error listing Pods: %v", err)
				return nil, fmt.Errorf("error when listing Pods: %w", err)
			}

			if len(podList.Items) == 0 {
				continue
			}

			pod := &podList.Items[0]
			if pod.Status.Phase == v1.PodPending || pod.Status.Phase == v1.PodRunning {
				return pod, nil
			}
		}
	}
}

// ReturnStatefulSetObject returns a matching appsv1.StatefulSet object based on the filters
// ReturnStatefulSetObject returns a matching appsv1.StatefulSet object based on the filters
func ReturnStatefulSetObject(clientset *kubernetes.Clientset, matchLabel, matchLabelValue, namespace string, timeoutSeconds int) (*appsv1.StatefulSet, error) {
	labelSelector := fmt.Sprintf("%s=%s", matchLabel, matchLabelValue)
	log.Info().Msgf("Waiting for StatefulSet with label %s to be created in namespace %s", labelSelector, namespace)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Error().Msg("Timeout waiting for StatefulSet to be created")
			return nil, fmt.Errorf("the StatefulSet %q in Namespace %q was not created within the timeout period", matchLabelValue, namespace)
		case <-ticker.C:
			statefulSets, err := clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{
				LabelSelector: labelSelector,
			})
			if err != nil {
				log.Error().Msgf("Error listing StatefulSets: %v", err)
				return nil, fmt.Errorf("error when listing StatefulSets: %w", err)
			}

			if len(statefulSets.Items) == 0 {
				continue
			}

			sts := &statefulSets.Items[0]
			if sts.Status.Replicas > 0 {
				return sts, nil
			}
		}
	}
}

// WaitForDeploymentReady waits for a target Deployment to become ready
func WaitForDeploymentReady(clientset *kubernetes.Clientset, deployment *appsv1.Deployment, timeoutSeconds int) (bool, error) {
	deploymentName := deployment.Name
	namespace := deployment.Namespace

	// Get the desired number of replicas from the deployment spec
	if deployment.Spec.Replicas == nil {
		log.Error().Msgf("Deployment %s in Namespace %s has nil Spec.Replicas", deploymentName, namespace)
		return false, fmt.Errorf("deployment %q in Namespace %q has nil Spec.Replicas", deploymentName, namespace)
	}
	desiredReplicas := *deployment.Spec.Replicas

	log.Info().Msgf("Waiting for Deployment %s in Namespace %s to be ready - this could take up to %v seconds", deploymentName, namespace, timeoutSeconds)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Error().Msgf("The Deployment %s in Namespace %s was not ready within the timeout period", deploymentName, namespace)
			return false, fmt.Errorf("the Deployment %q in Namespace %q was not ready within the timeout period", deploymentName, namespace)
		case <-ticker.C:
			// Get the latest Deployment object
			currentDeployment, err := clientset.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
			if err != nil {
				log.Error().Msgf("Error when getting Deployment %s in Namespace %s: %v", deploymentName, namespace, err)
				return false, fmt.Errorf("error when getting Deployment %q in Namespace %q: %w", deploymentName, namespace, err)
			}

			if currentDeployment.Status.ReadyReplicas == desiredReplicas {
				log.Info().Msgf("All Pods in Deployment %s are ready", deploymentName)
				return true, nil
			}
		}
	}
}

// WaitForPodReady waits for a target Pod to become ready
func WaitForPodReady(clientset *kubernetes.Clientset, pod *v1.Pod, timeoutSeconds int) (bool, error) {
	podName := pod.Name
	namespace := pod.Namespace

	log.Info().Msgf("Waiting for Pod %s in Namespace %s to be ready - this could take up to %v seconds", podName, namespace, timeoutSeconds)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Error().Msgf("The operation timed out while waiting for Pod %s in Namespace %s to become ready", podName, namespace)
			return false, fmt.Errorf("the operation timed out while waiting for Pod %q in Namespace %q", podName, namespace)
		case <-ticker.C:
			// Get the latest Pod object
			currentPod, err := clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
			if err != nil {
				log.Error().Msgf("Error when getting Pod %s in Namespace %s: %v", podName, namespace, err)
				return false, fmt.Errorf("error when getting Pod %q in Namespace %q: %w", podName, namespace, err)
			}

			if currentPod.Status.Phase == v1.PodRunning {
				log.Info().Msgf("Pod %s is %s.", podName, currentPod.Status.Phase)
				return true, nil
			}
		}
	}
}

// WaitForStatefulSetReady waits for a target StatefulSet to become ready
func WaitForStatefulSetReady(clientset *kubernetes.Clientset, statefulset *appsv1.StatefulSet, timeoutSeconds int, ignoreReady bool) (bool, error) {
	statefulSetName := statefulset.Name
	namespace := statefulset.Namespace

	// Get the desired number of replicas from the StatefulSet spec
	if statefulset.Spec.Replicas == nil {
		log.Error().Msgf("StatefulSet %s in Namespace %s has nil Spec.Replicas", statefulSetName, namespace)
		return false, fmt.Errorf("StatefulSet %q in Namespace %q has nil Spec.Replicas", statefulSetName, namespace)
	}
	desiredReplicas := *statefulset.Spec.Replicas

	log.Info().Msgf("Waiting for StatefulSet %s in Namespace %s to be ready - this could take up to %v seconds", statefulSetName, namespace, timeoutSeconds)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Error().Msgf("The StatefulSet %s in Namespace %s was not ready within the timeout period", statefulSetName, namespace)
			return false, fmt.Errorf("the StatefulSet %q in Namespace %q was not ready within the timeout period", statefulSetName, namespace)
		case <-ticker.C:
			// Get the latest StatefulSet object
			currentStatefulSet, err := clientset.AppsV1().StatefulSets(namespace).Get(ctx, statefulSetName, metav1.GetOptions{})
			if err != nil {
				log.Error().Msgf("Error when getting StatefulSet %s in Namespace %s: %v", statefulSetName, namespace, err)
				return false, fmt.Errorf("error when getting StatefulSet %q in Namespace %q: %w", statefulSetName, namespace, err)
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
						log.Error().Msgf("Could not find Pods owned by StatefulSet %s in Namespace %s: %v", statefulSetName, namespace, err)
						return false, fmt.Errorf("could not find Pods owned by StatefulSet %q in Namespace %q: %w", statefulSetName, namespace, err)
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
						log.Info().Msgf("All Pods in StatefulSet %s are running.", statefulSetName)
						return true, nil
					}
				}
			} else {
				// Check if ReadyReplicas match desired replicas
				if currentStatefulSet.Status.ReadyReplicas == desiredReplicas {
					log.Info().Msgf("All Pods in StatefulSet %s are ready.", statefulSetName)
					return true, nil
				}
			}
		}
	}
}
