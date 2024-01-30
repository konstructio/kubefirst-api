package k8s

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type UpdateSecretArgs struct {
	ConsoleTour string `json:"console-tour"`
}

type KubernetesClient struct {
	Clientset      *kubernetes.Clientset
	RestConfig     *rest.Config
	KubeConfigPath string
}
