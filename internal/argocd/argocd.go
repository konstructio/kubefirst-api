/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package argocd

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	v1alpha1ArgocdApplication "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	pkg "github.com/kubefirst/kubefirst-api/internal"
	"github.com/kubefirst/kubefirst-api/internal/argocdModel"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	coreV1Types "k8s.io/client-go/kubernetes/typed/core/v1"
)

var ArgocdSecretClient coreV1Types.SecretInterface

// todo call this ArgocdConfig or something not so generic
// Config ArgoCD configuration
type Config struct {
	Configs struct {
		Repositories struct {
			SoftServeGitops struct {
				URL      string `yaml:"url,omitempty"`
				Insecure string `json:"insecure,omitempty"`
				Type     string `json:"type,omitempty"`
				Name     string `json:"name,omitempty"`
			} `yaml:"soft-serve-gitops,omitempty"`
			RepoGitops struct {
				URL  string `yaml:"url,omitempty"`
				Type string `yaml:"type,omitempty"`
				Name string `yaml:"name,omitempty"`
			} `yaml:"github-serve-gitops,omitempty"`
		} `yaml:"repositories,omitempty"`
		CredentialTemplates struct {
			SSHCreds struct {
				URL           string `yaml:"url,omitempty"`
				SSHPrivateKey string `yaml:"sshPrivateKey,omitempty"`
			} `yaml:"ssh-creds,omitempty"`
		} `yaml:"credentialTemplates,omitempty"`
	} `yaml:"configs,omitempty"`
	Server struct {
		ExtraArgs []string `yaml:"extraArgs,omitempty"`
		Ingress   struct {
			Enabled     string `yaml:"enabled,omitempty"`
			Annotations struct {
				IngressKubernetesIoRewriteTarget   string `yaml:"ingress.kubernetes.io/rewrite-target,omitempty"`
				IngressKubernetesIoBackendProtocol string `yaml:"ingress.kubernetes.io/backend-protocol,omitempty"`
			} `yaml:"annotations,omitempty"`
			Hosts []string    `yaml:"hosts,omitempty"`
			TLS   []TLSConfig `yaml:"tls,omitempty"`
		} `yaml:"ingress,omitempty"`
	} `yaml:"server,omitempty"`
}

type TLSConfig struct {
	Hosts      []string `yaml:"hosts,omitempty"`
	SecretName string   `yaml:"secretName,omitempty"`
}

// Sync request ArgoCD to manual sync an application.
func DeleteApplication(httpClient pkg.HTTPDoer, applicationName, argoCDToken, cascade string) (httpCodeResponse int, syncStatus string, Error error) {

	params := url.Values{}
	params.Add("cascade", cascade)
	paramBody := strings.NewReader(params.Encode())

	url := fmt.Sprintf("%s/api/v1/applications/%s", GetArgoEndpoint(), applicationName)
	log.Info().Msg(url)
	req, err := http.NewRequest(http.MethodDelete, url, paramBody)
	if err != nil {
		log.Error().Err(err).Msg("")
		return 0, "", err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", argoCDToken))
	res, err := httpClient.Do(req)
	if err != nil {
		log.Error().Err(err).Msgf("error sending DELETE request to ArgoCD for application (%s)", applicationName)
		return res.StatusCode, "", err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Warn().Err(err).Msgf("argocd http response code is: %d", res.StatusCode)
		return res.StatusCode, "", nil
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return res.StatusCode, "", err
	}

	var syncResponse argocdModel.V1alpha1Application
	err = json.Unmarshal(body, &syncResponse)
	if err != nil {
		return res.StatusCode, "", err
	}

	return res.StatusCode, syncResponse.Status.Sync.Status, nil
}

// GetArgoCDApplication by receiving the ArgoCD token, and the application name, this function returns the full
// application data Application struct. This can be used when a resource needs to be updated, we firstly collect all
// Application data, update what is necessary and then request the PUT function to update the resource.
func GetArgoCDApplication(token string, applicationName string) (argocdModel.V1alpha1Application, error) {

	// todo: instantiate a new client on every http request isn't a good idea, we might want to work with methods and
	//       provide resources via structs.
	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	httpClient := http.Client{Transport: customTransport}

	url := pkg.ArgoCDLocalBaseURL + "/applications/" + applicationName
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Error().Err(err).Msg("")
	}

	req.Header.Add("Authorization", "Bearer "+token)

	res, err := httpClient.Do(req)
	if err != nil {
		return argocdModel.V1alpha1Application{}, err
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return argocdModel.V1alpha1Application{}, err
	}

	var response argocdModel.V1alpha1Application
	err = json.Unmarshal(body, &response)
	if err != nil {
		return argocdModel.V1alpha1Application{}, err
	}

	return response, nil
}

// GetArgoEndpoint provides a solution in the interim for returning the correct
// endpoint address
func GetArgoEndpoint() string {
	var argoCDLocalEndpoint string
	if viper.GetString("argocd.local.service") != "" {
		argoCDLocalEndpoint = viper.GetString("argocd.local.service")
	} else {
		argoCDLocalEndpoint = pkg.ArgocdPortForwardURL
	}
	return argoCDLocalEndpoint
}

func GetArgoCDApplicationObject(gitopsRepoURL, registryPath string) *v1alpha1ArgocdApplication.Application {
	return &v1alpha1ArgocdApplication.Application{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Application",
			APIVersion: "argoproj.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "registry",
			Namespace:   "argocd",
			Annotations: map[string]string{"argocd.argoproj.io/sync-wave": "1"},
		},
		Spec: v1alpha1ArgocdApplication.ApplicationSpec{
			Source: &v1alpha1ArgocdApplication.ApplicationSource{
				RepoURL:        gitopsRepoURL,
				Path:           registryPath,
				TargetRevision: "HEAD",
			},
			Destination: v1alpha1ArgocdApplication.ApplicationDestination{
				Server:    "https://kubernetes.default.svc",
				Namespace: "argocd",
			},
			Project: "default",
			SyncPolicy: &v1alpha1ArgocdApplication.SyncPolicy{
				Automated: &v1alpha1ArgocdApplication.SyncPolicyAutomated{
					Prune:    true,
					SelfHeal: true,
				},
				SyncOptions: []string{"CreateNamespace=true"},
				Retry: &v1alpha1ArgocdApplication.RetryStrategy{
					Limit: 5,
					Backoff: &v1alpha1ArgocdApplication.Backoff{
						Duration:    "5s",
						Factor:      new(int64),
						MaxDuration: "5m0s",
					},
				},
			},
		},
	}
}
