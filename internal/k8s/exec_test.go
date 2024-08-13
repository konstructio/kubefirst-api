/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k8s

import (
	"syscall"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stest "k8s.io/client-go/testing"
)

func TestReturnDeploymentObjectV2(t *testing.T) {
	client := fake.NewSimpleClientset()

	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "default",
			Labels: map[string]string{
				"app": "test",
			},
		},
		Status: appsv1.DeploymentStatus{
			Replicas: 1, // we're ensuring this is set to non-zero in the function
		},
	}

	counter := 0
	client.PrependReactor("list", "deployments", func(action k8stest.Action) (handled bool, ret runtime.Object, err error) {
		if counter == 0 {
			counter++
			return true, nil, syscall.ECONNREFUSED
		}

		return true, &appsv1.DeploymentList{
			Items: []appsv1.Deployment{deployment},
		}, nil
	})

	ch := Checker{Interval: 100 * time.Millisecond}

	deploymentObject, err := ch.ReturnDeploymentObjectV2(client, map[string]string{"app": "test"}, deployment.ObjectMeta.Namespace, 30)
	if err != nil {
		t.Errorf("unable to get deployment object: %v", err)
		return
	}

	if deploymentObject.Name != deployment.ObjectMeta.Name {
		t.Errorf("expected deployment name %s, got %s", deployment.ObjectMeta.Name, deploymentObject.Name)
	}

	if deploymentObject.Namespace != deployment.ObjectMeta.Namespace {
		t.Errorf("expected deployment namespace %s, got %s", deployment.ObjectMeta.Namespace, deploymentObject.Namespace)
	}
}
