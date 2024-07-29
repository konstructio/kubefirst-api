/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package providerConfigs

import (
	"bytes"
	"fmt"
	"github.com/Masterminds/sprig/v3"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"text/template"
)

// ToTemplateVars - converts a string to a template variable
func ToTemplateVars(input string, instance Tokens) string {
	value := reflect.ValueOf(instance)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}
	sanitizer := strings.NewReplacer("<", "", "_", "", ">", "", "-", "")

	fields := value.Type()
	normalizedName := strings.ToLower(sanitizer.Replace(input))
	for i := 0; i < fields.NumField(); i++ {
		field := fields.Field(i)
		val := value.Field(i)

		if normalizedName == strings.ToLower(field.Name) {
			// If the field value is the zero value of the field template
			// will add - to the template variable to clean up extra whitespace.
			// We do not have any bool values in the tokens, so we should be ok.
			// TODO: Find a better way to handle this.
			if val.IsZero() {
				return fmt.Sprintf("%s- .%s %s", leftDelimiter, field.Name, rightDelimiter)
			}
			// if field name matches return the correct formatted name
			return fmt.Sprintf("%s .%s %s", leftDelimiter, field.Name, rightDelimiter)
		}
	}

	// If no match found, return <variable-not-found> so we can have a visual indication
	// that the token was not found in the Token Struct without erroring out
	return "<variable-not-found>"
}

// Detokenize - Translate tokens by values on a given path
func Detokenize(path string, tokens Tokens, gitProtocol string, useCloudflareOriginIssuer bool) error {
	err := filepath.Walk(path, detokenize(tokens, gitProtocol, useCloudflareOriginIssuer))
	if err != nil {
		return err
	}

	return nil
}

func detokenize(tokens Tokens, gitProtocol string, useCloudflareOriginIssuer bool) filepath.WalkFunc {
	return filepath.WalkFunc(func(path string, fi os.FileInfo, err error) error {
		if fi.IsDir() && fi.Name() == ".git" {
			return filepath.SkipDir
		}

		if fi.IsDir() {
			return nil
		}
		// Incase there is a .gitignore or .gitkeep etc file in the directory, skip it.
		if strings.Contains(fi.Name(), ".git") {
			return nil
		}

		var (
			newContentData []byte
		)

		switch tokenType := tokens.(type) {
		case *GitopsDirectoryValues:
			setGitOpsTokens(tokenType, gitProtocol, useCloudflareOriginIssuer)
			tokens = tokenType
		case *MetaphorTokenValues:
			setMetaphorTokens(tokenType, gitProtocol)
			tokens = tokenType
		}

		newContentData, err = renderGoTemplating(path, tokens)
		if err != nil {
			return err
		}
		return os.WriteFile(path, newContentData, 0644)

	})
}

func setMetaphorTokens(tokens *MetaphorTokenValues, gitProtocol string) {
	tokens.GitProtocol = gitProtocol
	tokens.MetaphorDevelopmentIngressURL = fmt.Sprintf("https://metaphor-dev.%s", tokens.DomainName)
	tokens.MetaphorProductionIngressURL = fmt.Sprintf("https://metaphor.%s", tokens.DomainName)
	tokens.MetaphorStagingIngressURL = fmt.Sprintf("https://metaphor-stage.%s", tokens.DomainName)

}

func setGitOpsTokens(tokens *GitopsDirectoryValues, gitProtocol string, useCloudflareOriginIssuer bool) {
	if tokens.SubdomainName != "" {
		if !strings.Contains(tokens.DomainName, tokens.SubdomainName) {
			tokens.DomainName = fmt.Sprintf("%s.%s", tokens.SubdomainName, tokens.DomainName)
		}
	}
	//origin issuer defines which annotations should be on ingresses
	if useCloudflareOriginIssuer {
		tokens.CertManagerIssuerAnnotation1 = "cert-manager.io/issuer: cloudflare-origin-issuer"
		tokens.CertManagerIssuerAnnotation2 = "cert-manager.io/issuer-kind: OriginIssuer"
		tokens.CertManagerIssuerAnnotation3 = "cert-manager.io/issuer-group: cert-manager.k8s.cloudflare.com"
		tokens.CertManagerIssuerAnnotation4 = "external-dns.alpha.kubernetes.io/cloudflare-proxied: \"true\""
	} else {
		tokens.CertManagerIssuerAnnotation1 = "cert-manager.io/cluster-issuer: \"letsencrypt-prod\""
	}

	if tokens.CloudProvider == "k3s" {
		tokens.K3sEndpoint = tokens.K3sServersPrivateIps[0]
	}

	// The fqdn is used by metaphor/argo to choose the appropriate url for cicd operations.
	if gitProtocol == "https" {
		tokens.GitFqdn = fmt.Sprintf("https://%v.com/", tokens.GitProvider)
	} else {
		tokens.GitFqdn = fmt.Sprintf("git@%v.com:", tokens.GitProvider)
	}
}

// renderGoTemplating - Renders the template with the given values
// it also includes the sprig GenericFuncMap functions listed here: https://masterminds.github.io/sprig/.
func parseTemplate(content string) (*template.Template, error) {
	funcs := sprig.GenericFuncMap()
	t := template.New("gitops-template").
		Funcs(funcs).
		Delims(leftDelimiter, rightDelimiter)
	return t.Parse(content)
}

func executeTemplate(t *template.Template, writer io.Writer, tokens Tokens) error {
	switch tmplTokens := tokens.(type) {
	case *GitopsDirectoryValues:
		return t.Execute(writer, tmplTokens)
	case *MetaphorTokenValues:
		// Handle tokens of type *MetaphorTokenValues
		return t.Execute(writer, tmplTokens)
	default:
		return fmt.Errorf("invalid type for tokens: %T", tokens)
	}
}

func replaceTemplateVariables(content string, regex *regexp.Regexp, tokens Tokens) string {
	return regex.ReplaceAllStringFunc(content, tokens.ToTemplateVars)
}

// renderGoTemplating - Render a template with the given tokens.
func renderGoTemplating(path string, tokens Tokens) ([]byte, error) {
	read, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	// Replace all tokens with their values
	regex := regexp.MustCompile(TokenRegexPattern)
	content := replaceTemplateVariables(string(read), regex, tokens)
	buff := bytes.NewBufferString(content)

	parsedTemplate, err := parseTemplate(buff.String())
	if err != nil {
		return nil, err
	}

	newBuff := bytes.NewBuffer([]byte{})
	if err = executeTemplate(parsedTemplate, newBuff, tokens); err != nil {
		return nil, err
	}

	return newBuff.Bytes(), nil
}
