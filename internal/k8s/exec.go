/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k8s

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/term"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		log.Error().Msgf("error getting Service %q in Namespace %q: %s\n", serviceName, namespace, err)
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

// ReturnDeploymentObject returns a matching appsv1.Deployment object based on the filters
func ReturnDeploymentObject(clientset *kubernetes.Clientset, matchLabel, matchLabelValue, namespace string, timeoutSeconds int) (*appsv1.Deployment, error) {
	// Filter
	deploymentListOptions := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", matchLabel, matchLabelValue),
	}

	log.Info().Msgf("waiting for %s Deployment to be created", matchLabelValue)

	// Create watch operation
	objWatch, err := clientset.
		AppsV1().
		Deployments(namespace).
		Watch(context.Background(), deploymentListOptions)
	if err != nil {
		log.Error().Msgf("error when attempting to search for Deployment %s in Namespace %s: %s", matchLabelValue, namespace, err)
		return nil, fmt.Errorf("error when attempting to search for Deployment %q in Namespace %q: %w", matchLabelValue, namespace, err)
	}

	objChan := objWatch.ResultChan()
	for {
		select {
		case event, ok := <-objChan:
			time.Sleep(time.Second * 15)
			if !ok {
				// Error if the channel closes
				log.Error().Msgf("error waiting for %s Deployment in Namespace %s to be created: %s", matchLabelValue, namespace, err)
				return nil, fmt.Errorf("error waiting for %q Deployment in Namespace %q to be created: %w", matchLabelValue, namespace, err)
			}

			//nolint:forcetypeassert // we are confident this is a Deployment
			if event.
				Object.(*appsv1.Deployment).Status.Replicas > 0 {
				spec, err := clientset.AppsV1().Deployments(namespace).List(context.Background(), deploymentListOptions)
				if err != nil {
					log.Error().Msgf("Error when searching for Deployment %s in Namespace %s: %s", matchLabelValue, namespace, err)
					return nil, fmt.Errorf("error when searching for Deployment %q in Namespace %q: %w", matchLabelValue, namespace, err)
				}
				return &spec.Items[0], nil
			}
		case <-time.After(time.Duration(timeoutSeconds) * time.Second):
			log.Error().Msg("the Deployment was not created within the timeout period")
			return nil, fmt.Errorf("the Deployment %s in Namespace %s was not created within the timeout period", matchLabelValue, namespace)
		}
	}
}

// ReturnPodObject returns a matching v1.Pod object based on the filters
func ReturnPodObject(kubeConfigPath, matchLabel, matchLabelValue, namespace string, timeoutSeconds int) (*v1.Pod, error) {
	clientset, err := GetClientSet(kubeConfigPath)
	if err != nil {
		return nil, fmt.Errorf("error getting client set from kubeConfigPath %q: %w", kubeConfigPath, err)
	}

	// Filter
	podListOptions := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", matchLabel, matchLabelValue),
	}
	log.Info().Msgf("waiting for %s Pod to be created", matchLabelValue)

	// Create watch operation
	objWatch, err := clientset.
		CoreV1().
		Pods(namespace).
		Watch(context.Background(), podListOptions)
	if err != nil {
		log.Error().Msgf("error when attempting to search for Pod %s in Namespace %s: %s", matchLabelValue, namespace, err)
		return nil, fmt.Errorf("error when attempting to search for Pod %q in Namespace %q: %w", matchLabelValue, namespace, err)
	}

	objChan := objWatch.ResultChan()
	for {
		select {
		case event, ok := <-objChan:
			time.Sleep(time.Second * 15)
			if !ok {
				// Error if the channel closes
				log.Error().Msgf("error waiting for %s Pod in Namespace %s to be created: %s", matchLabelValue, namespace, err)
				return nil, fmt.Errorf("error waiting for %q Pod in Namespace %q to be created: %w", matchLabelValue, namespace, err)
			}

			//nolint:forcetypeassert // we are confident this is a Pod
			if event.
				Object.(*v1.Pod).Status.Phase == "Pending" {
				spec, err := clientset.CoreV1().Pods(namespace).List(context.Background(), podListOptions)
				if err != nil {
					log.Error().Msgf("error when searching for Pod %s in Namespace %s: %s", matchLabelValue, namespace, err)
					return nil, fmt.Errorf("error when searching for Pod %q in Namespace %q: %w", matchLabelValue, namespace, err)
				}
				return &spec.Items[0], nil
			}

			//nolint:forcetypeassert // we are confident this is a Pod
			if event.
				Object.(*v1.Pod).Status.Phase == "Running" {
				spec, err := clientset.CoreV1().Pods(namespace).List(context.Background(), podListOptions)
				if err != nil {
					log.Error().Msgf("error when searching for Pod %s in Namespace %s: %s", matchLabelValue, namespace, err)
					return nil, fmt.Errorf("error when searching for Pod %q in Namespace %q: %w", matchLabelValue, namespace, err)
				}
				return &spec.Items[0], nil
			}
		case <-time.After(time.Duration(timeoutSeconds) * time.Second):
			log.Error().Msg("the Pod was not created within the timeout period")
			return nil, fmt.Errorf("the Pod %q in Namespace %q was not created within the timeout period", matchLabelValue, namespace)
		}
	}
}

// ReturnStatefulSetObject returns a matching appsv1.StatefulSet object based on the filters
func ReturnStatefulSetObject(clientset *kubernetes.Clientset, matchLabel, matchLabelValue, namespace string, timeoutSeconds int) (*appsv1.StatefulSet, error) {
	// Filter
	statefulSetListOptions := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", matchLabel, matchLabelValue),
	}

	log.Info().Msgf("waiting for %s StatefulSet to be created using label %s=%s", matchLabelValue, matchLabel, matchLabelValue)

	// Create watch operation
	objWatch, err := clientset.
		AppsV1().
		StatefulSets(namespace).
		Watch(context.Background(), statefulSetListOptions)
	if err != nil {
		log.Error().Msgf("error when attempting to search for StatefulSet with label %s=%s in Namespace %s: %s", matchLabel, matchLabelValue, namespace, err)
	}

	objChan := objWatch.ResultChan()
	for {
		select {
		case event, ok := <-objChan:
			time.Sleep(time.Second * 15)
			if !ok {
				// Error if the channel closes
				log.Error().Msgf("error not ok waiting %s StatefulSet to be created: %s", matchLabelValue, err)
				return nil, fmt.Errorf("error not ok waiting %s StatefulSet to be created: %w", matchLabelValue, err)
			}

			//nolint:forcetypeassert // we are confident this is a StatefulSet
			if event.Object.(*appsv1.StatefulSet).Status.Replicas > 0 {
				spec, err := clientset.AppsV1().StatefulSets(namespace).List(context.Background(), statefulSetListOptions)
				if err != nil {
					log.Error().Msgf("error when searching for StatefulSet %s with label %s=%s in Namespace %s: %s", matchLabelValue, matchLabel, matchLabelValue, namespace, err)
					return nil, fmt.Errorf("error when searching for StatefulSet %q with label %s=%s in Namespace %s: %w", matchLabelValue, matchLabel, matchLabelValue, namespace, err)
				}
				return &spec.Items[0], nil
			}
		case <-time.After(time.Duration(timeoutSeconds) * time.Second):
			log.Error().Msg("the StatefulSet was not created within the timeout period")
			return nil, fmt.Errorf("the StatefulSet %q in Namespace %q was not created within the timeout period", matchLabelValue, namespace)
		}
	}
}

// WaitForDeploymentReady waits for a target Deployment to become ready
func WaitForDeploymentReady(clientset *kubernetes.Clientset, deployment *appsv1.Deployment, timeoutSeconds int) (bool, error) {
	// Format list for metav1.ListOptions for watch
	configuredReplicas := deployment.Status.Replicas
	watchOptions := metav1.ListOptions{
		FieldSelector: fmt.Sprintf(
			"metadata.name=%s", deployment.Name),
	}

	// Create watch operation
	objWatch, err := clientset.
		AppsV1().
		Deployments(deployment.ObjectMeta.Namespace).
		Watch(context.Background(), watchOptions)
	if err != nil {
		log.Error().Msgf("error when attempting to wait for Deployment %s in Namespace %s: %s", deployment.Name, deployment.Namespace, err)
		return false, fmt.Errorf("error when attempting to wait for Deployment %q in Namespace %q: %w", deployment.Name, deployment.Namespace, err)
	}
	log.Info().Msgf("waiting for %s Deployment to be ready - this could take up to %v seconds", deployment.Name, timeoutSeconds)

	objChan := objWatch.ResultChan()
	for {
		select {
		case event, ok := <-objChan:
			time.Sleep(time.Second * 15)
			if !ok {
				// Error if the channel closes
				log.Error().Msgf("error waiting for Deployment %s in Namespace %s: %s", deployment.Name, deployment.Namespace, err)
				return false, fmt.Errorf("error waiting for Deployment %q in Namespace %q: %w", deployment.Name, deployment.Namespace, err)
			}

			//nolint:forcetypeassert // we are confident this is a Deployment
			if event.
				Object.(*appsv1.Deployment).
				Status.ReadyReplicas == configuredReplicas {
				log.Info().Msgf("all Pods in Deployment %s are ready", deployment.Name)
				return true, nil
			}
		case <-time.After(time.Duration(timeoutSeconds) * time.Second):
			log.Error().Msg("the Deployment was not ready within the timeout period")
			return false, fmt.Errorf("the Deployment %q in Namespace %q was not ready within the timeout period", deployment.Name, deployment.Namespace)
		}
	}
}

// WaitForPodReady waits for a target Pod to become ready
func WaitForPodReady(clientset *kubernetes.Clientset, pod *v1.Pod, timeoutSeconds int) (bool, error) {
	// Format list for metav1.ListOptions for watch
	watchOptions := metav1.ListOptions{
		FieldSelector: fmt.Sprintf(
			"metadata.name=%s", pod.Name),
	}

	// Create watch operation
	objWatch, err := clientset.
		CoreV1().
		Pods(pod.ObjectMeta.Namespace).
		Watch(context.Background(), watchOptions)
	if err != nil {
		log.Error().Msgf("error when attempting to wait for Pod %s in Namespace %s: %s", pod.Name, pod.Namespace, err)
		return false, fmt.Errorf("error when attempting to wait for Pod %q in Namespace %q: %w", pod.Name, pod.Namespace, err)
	}
	log.Info().Msgf("waiting for %s Pod to be ready - this could take up to %v seconds", pod.Name, timeoutSeconds)

	// Feed events using provided channel
	objChan := objWatch.ResultChan()

	// Listen until the Pod is ready
	// Timeout if it isn't ready within timeoutSeconds
	for {
		select {
		case event, ok := <-objChan:
			if !ok {
				// Error if the channel closes
				log.Error().Msgf("error waiting for Pod %s in Namespace %s: %s", pod.Name, pod.Namespace, err)
				return false, fmt.Errorf("error waiting for Pod %q in Namespace %q: %w", pod.Name, pod.Namespace, err)
			}
			if event.
				Object.(*v1.Pod).
				Status.
				Phase == "Running" {
				log.Info().Msgf("Pod %s is %s.", pod.Name, event.Object.(*v1.Pod).Status.Phase)
				return true, nil
			}
		case <-time.After(time.Duration(timeoutSeconds) * time.Second):
			log.Error().Msg("the operation timed out while waiting for the Pod to become ready")
			return false, fmt.Errorf("the operation timed out while waiting for Pod %q in Namespace %q", pod.Name, pod.Namespace)
		}
	}
}

// WaitForStatefulSetReady waits for a target StatefulSet to become ready
func WaitForStatefulSetReady(clientset *kubernetes.Clientset, statefulset *appsv1.StatefulSet, timeoutSeconds int, ignoreReady bool) (bool, error) {
	// Format list for metav1.ListOptions for watch
	configuredReplicas := statefulset.Status.Replicas

	// Create watch operation
	objWatch, err := clientset.AppsV1().StatefulSets(statefulset.ObjectMeta.Namespace).Watch(context.Background(), metav1.ListOptions{
		FieldSelector: fmt.Sprintf(
			"metadata.name=%s", statefulset.Name),
	})
	if err != nil {
		log.Error().Msgf("error when attempting to wait for StatefulSet %s in Namespace %s: %s", statefulset.Name, statefulset.Namespace, err)
		return false, fmt.Errorf("error when attempting to wait for StatefulSet %q in Namespace %q: %w", statefulset.Name, statefulset.Namespace, err)
	}
	log.Info().Msgf("waiting for %s StatefulSet to be ready - this could take up to %v seconds", statefulset.Name, timeoutSeconds)

	objChan := objWatch.ResultChan()
	for {
		select {
		case event, ok := <-objChan:
			time.Sleep(time.Second * 15)
			if !ok {
				// Error if the channel closes
				log.Error().Msgf("error waiting for StatefulSet %s in Namespace %s: %s", statefulset.Name, statefulset.Namespace, err)
				return false, fmt.Errorf("error waiting for StatefulSet %q in Namespace %q: %w", statefulset.Name, statefulset.Namespace, err)
			}
			if ignoreReady {
				// Under circumstances where Pods may be running but not ready
				// These may require additional setup before use, etc.
				currentRevision := event.Object.(*appsv1.StatefulSet).Status.CurrentRevision
				if event.Object.(*appsv1.StatefulSet).Status.CurrentReplicas == configuredReplicas {
					// Get Pods owned by the StatefulSet
					pods, err := clientset.CoreV1().Pods(statefulset.ObjectMeta.Namespace).List(context.Background(), metav1.ListOptions{
						LabelSelector: fmt.Sprintf("controller-revision-hash=%s", currentRevision),
					})
					if err != nil {
						log.Error().Msgf("could not find Pods owned by StatefulSet %s in Namespace %s: %s", statefulset.Name, statefulset.Namespace, err)
						return false, fmt.Errorf("could not find Pods owned by StatefulSet %q in Namespace %q: %w", statefulset.Name, statefulset.Namespace, err)
					}

					// Determine when the Pods are running
					for _, pod := range pods.Items {
						err := watchForStatefulSetPodReady(clientset, statefulset.Namespace, pod.Name, timeoutSeconds)
						if err != nil {
							log.Error().Msgf("error waiting for Pod %q in StatefulSet %q: %s", pod.Name, statefulset.Name, err)
							return false, err
						}
						log.Info().Msgf("pod %s in statefulset %s is running", pod.Name, statefulset.Name)
					}
					objWatch.Stop()
					return true, nil
				}
			} else if event.Object.(*appsv1.StatefulSet).Status.AvailableReplicas == configuredReplicas {
				log.Info().Msgf("All Pods in StatefulSet %s are ready.", statefulset.Name)
				objWatch.Stop()
				return true, nil
			}
		case <-time.After(time.Duration(timeoutSeconds) * time.Second):
			log.Error().Msg("the StatefulSet was not ready within the timeout period")
			return false, fmt.Errorf("the StatefulSet %q in Namespace %q was not ready within the timeout period", statefulset.Name, statefulset.Namespace)
		}
	}
}

// watchForStatefulSetPodReady inspects a Pod associated with a StatefulSet and
// uses a channel to determine when it's ready
// The channel will timeout if the Pod isn't ready by timeoutSeconds
func watchForStatefulSetPodReady(clientset *kubernetes.Clientset, namespace, podName string, timeoutSeconds int) error {
	podObjWatch, err := clientset.CoreV1().Pods(namespace).Watch(context.Background(), metav1.ListOptions{
		FieldSelector: fmt.Sprintf(
			"metadata.name=%s", podName),
	})
	if err != nil {
		log.Error().Msgf("error when attempting to wait for Pod %s in Namespace %s: %s", podName, namespace, err)
		return fmt.Errorf("error when attempting to wait for Pod %q in Namespace %q: %w", podName, namespace, err)
	}

	podObjChan := podObjWatch.ResultChan()
	for {
		select {
		case podEvent, ok := <-podObjChan:
			time.Sleep(time.Second * 15)
			if !ok {
				// Error if the channel closes
				log.Error().Msgf("error waiting for Pod %s in Namespace %s: %s", podName, namespace, err)
				return fmt.Errorf("error waiting for Pod %q in Namespace %q: %w", podName, namespace, err)
			}
			if podEvent.Object.(*v1.Pod).Status.Phase == "Running" {
				podObjWatch.Stop()
				return nil
			}
		case <-time.After(time.Duration(timeoutSeconds) * time.Second):
			log.Error().Msg("the StatefulSet Pod was not ready within the timeout period")
			return fmt.Errorf("the StatefulSet Pod %q in Namespace %q was not ready within the timeout period", podName, namespace)
		}
	}
}
