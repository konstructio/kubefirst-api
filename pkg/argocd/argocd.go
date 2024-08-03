package argocd

import "github.com/kubefirst/kubefirst-api/internal/argocd"

var (
	ArgocdSecretClient         = argocd.ArgocdSecretClient
	GetArgocdTokenV2           = argocd.GetArgocdTokenV2
	GetArgoCDApplicationObject = argocd.GetArgoCDApplicationObject
	RefreshApplication         = argocd.RefreshApplication
	RefreshRegistryApplication = argocd.RefreshRegistryApplication
)
