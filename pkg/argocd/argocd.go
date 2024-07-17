package argocd

import "github.com/kubefirst/kubefirst-api/internal/argocd"

var ArgocdSecretClient = argocd.ArgocdSecretClient
var GetArgocdTokenV2 = argocd.GetArgocdTokenV2
var GetArgoCDApplicationObject = argocd.GetArgoCDApplicationObject
var RefreshApplication = argocd.RefreshApplication
var RefreshRegistryApplication = argocd.RefreshRegistryApplication
