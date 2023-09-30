package types

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type KubernetesClient struct {
	Clientset      *kubernetes.Clientset
	RestConfig     *rest.Config
	KubeConfigPath string
}
