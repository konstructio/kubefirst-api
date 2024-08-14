/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k8s

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ReturnJobObject returns a matching appsv1.StatefulSet object based on the filters
func ReturnJobObject(clientset *kubernetes.Clientset, namespace, jobName string) (*batchv1.Job, error) {
	job, err := clientset.BatchV1().Jobs(namespace).Get(context.Background(), jobName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve Job %q in namespace %q: %w", jobName, namespace, err)
	}

	return job, nil
}

// WaitForJobComplete waits for a target Job to reach completion
func WaitForJobComplete(clientset *kubernetes.Clientset, jobName, jobNamespace string, timeoutSeconds int64) (bool, error) {
	// Format list for metav1.ListOptions for watch
	watchOptions := metav1.ListOptions{
		FieldSelector: fmt.Sprintf(
			"metadata.name=%s", jobName),
	}

	// Create watch operation
	objWatch, err := clientset.
		BatchV1().
		Jobs(jobNamespace).
		Watch(context.Background(), watchOptions)
	if err != nil {
		log.Error().Msgf("error when attempting to wait for Job: %s", err)
		return false, fmt.Errorf("unable to create watch for Job %q in namespace %q: %w", jobName, jobNamespace, err)
	}
	log.Info().Msgf("waiting for %s Job completion. This could take up to %v seconds.", jobName, timeoutSeconds)

	// Feed events using provided channel
	objChan := objWatch.ResultChan()

	// Listen until the Job is complete
	// Timeout if it isn't complete within timeoutSeconds
	for {
		select {
		case event, ok := <-objChan:
			if !ok {
				// Error if the channel closes
				log.Error().Msgf("failed to wait for job %s to complete", jobName)
				return false, fmt.Errorf("job %q channel closed unexpectedly while waiting for completion", jobName)
			}
			if event.
				Object.(*batchv1.Job).
				Status.Succeeded > 0 {
				log.Info().Msgf("job %s completed at %s.", jobName, event.Object.(*batchv1.Job).Status.CompletionTime)
				return true, nil
			}
		case <-time.After(time.Duration(timeoutSeconds) * time.Second):
			log.Error().Msg("the operation timed out while waiting for the Job to complete")
			return false, fmt.Errorf("the operation timed out while waiting for Job %q in namespace %s to complete", jobName, jobNamespace)
		}
	}
}
