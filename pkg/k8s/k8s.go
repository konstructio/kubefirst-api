package k8s

import internal "github.com/kubefirst/kubefirst-api/internal/k8s"

type KubernetesClient = internal.KubernetesClient

var CheckForExistingPortForwards = internal.CheckForExistingPortForwards

var CreateKubeConfig = internal.CreateKubeConfig
var ReturnDeploymentObject = internal.ReturnDeploymentObject
var WaitForDeploymentReady = internal.WaitForDeploymentReady
var VerifyArgoCDReadiness = internal.VerifyArgoCDReadiness
var GetSecretValue = internal.GetSecretValue
var OpenPortForwardPodWrapper = internal.OpenPortForwardPodWrapper
var ReturnStatefulSetObject = internal.ReturnStatefulSetObject
var WaitForStatefulSetReady = internal.WaitForStatefulSetReady
var ReturnJobObject = internal.ReturnJobObject
var WaitForJobComplete = internal.WaitForJobComplete

var ReadSecretV2 = internal.ReadSecretV2
var ReadService = internal.ReadService
var CreateSecretV2 = internal.CreateSecretV2
