package k8s

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type UpdateSecretArgs struct {
	ConsoleTour string `json:"console-tour"`
}

type CloudAccountsArgs struct {
	CloudAccounts string `json:"cloud_accounts"`
}

type KubernetesClient struct {
	Clientset      *kubernetes.Clientset
	RestConfig     *rest.Config
	KubeConfigPath string
}
