/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package providerConfigs

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"text/template"
)

func ToTemplateVars(input string, instance Tokens) string {
	value := reflect.ValueOf(instance)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}
	sanitizer := strings.NewReplacer("<", "", "_", "", ">", "")
	
	fields := value.Type()
	normalizedName := strings.ToLower(sanitizer.Replace(input))
	for i := 0; i < fields.NumField(); i++ {
		field := fields.Field(i)
		val := value.Field(i) // Use value.Field(i) instead of value.FieldByName(field.Name)
		
		if normalizedName == strings.ToLower(field.Name) {
			if val.IsZero() {
				return ""
			}
			// if field name matches return the correct formatted name
			return fmt.Sprintf("%s .%s %s", leftDelimiter, field.Name, rightDelimiter)
		}
	}
	
	// If no match found, return an error placeholder
	return "<variable not found>"
}

// DetokenizeGitGitops - Translate tokens by values on a given path
func DetokenizeGitGitops(path string, tokens *GitopsDirectoryValues, gitProtocol string, useCloudflareOriginIssuer bool) error {
	err := filepath.Walk(path, detokenizeGitops(path, tokens, gitProtocol, useCloudflareOriginIssuer))
	if err != nil {
		return err
	}
	
	return nil
}

func detokenizeGitops(path string, tokens *GitopsDirectoryValues, gitProtocol string, useCloudflareOriginIssuer bool) filepath.WalkFunc {
	return filepath.WalkFunc(func(path string, fi os.FileInfo, err error) error {
		if fi.IsDir() && fi.Name() == ".git" {
			return filepath.SkipDir
		}
		
		if fi.IsDir() {
			return nil
		}
		
		if strings.Contains(fi.Name(), ".git") {
			return nil
		}
		//
		//metaphorDevelopmentIngressURL := fmt.Sprintf("https://metaphor-development.%s", tokens.DomainName)
		//metaphorStagingIngressURL := fmt.Sprintf("https://metaphor-staging.%s", tokens.DomainName)
		//metaphorProductionIngressURL := fmt.Sprintf("https://metaphor-production.%s", tokens.DomainName)
		
		// var matched bool
		matched, _ := filepath.Match("*", fi.Name())
		
		if matched {
			// ignore .git files
			read, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			
			if tokens.SubdomainName != "" {
				tokens.DomainName = fmt.Sprintf("%s.%s", tokens.SubdomainName, tokens.DomainName)
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
			// The fqdn is used by metaphor/argo to choose the appropriate url for cicd operations.
			if gitProtocol == "https" {
				tokens.GitFqdn = fmt.Sprintf("https://%v.com/", tokens.GitProvider)
			} else {
				tokens.GitFqdn = fmt.Sprintf("git@%v.com:", tokens.GitProvider)
			}
			
			newContents := strings.TrimSpace(string(read))
			newContentData, err := renderGoTemplating(tokens, newContents)
			
			if err != nil {
				return err
			}
			
			return os.WriteFile(path, newContentData, 0)
		}
		return nil
	})
}

func parseTemplate(content string) (*template.Template, error) {
	t := template.New("gitops-template").Delims(leftDelimiter, rightDelimiter)
	return t.Parse(content)
}

func executeTemplate(t *template.Template, writer io.Writer, tokens Tokens) error {
	return t.Execute(writer, tokens)
}

func replaceTemplateVariables(content string, tokens Tokens) string {
	regex := regexp.MustCompile(TokenRegexPattern)
	return regex.ReplaceAllStringFunc(content, tokens.ToTemplateVars)
}

func toTemplateVariable(v string) string {
	fields := reflect.TypeOf(GitopsDirectoryValues{})
	r := regexp.MustCompile("<|>")
	strippedVar := r.ReplaceAllString(strings.ToLower(v), "")
	strippedVar = strings.ReplaceAll(strippedVar, "_", "")
	for i := 0; i < fields.NumField(); i++ {
		field := fields.Field(i)
		val := reflect.ValueOf(field)
		if strippedVar == strings.ToLower(field.Name) {
			
			if val.IsZero() {
				return ""
			}
			// if field name matches return the correct formatted name
			return fmt.Sprintf("%s .%s %s", leftDelimiter, field.Name, rightDelimiter)
		}
	}
	
	// If no match found, return an error placeholder
	return "<variable not found>"
}

// DetokenizeGitMetaphor - Translate tokens by values on a given path
func DetokenizeGitMetaphor(path string, tokens *MetaphorTokenValues) error {
	err := filepath.Walk(path, detokenizeGitopsMetaphor(path, tokens))
	if err != nil {
		return err
	}
	return nil
}

// DetokenizeDirectoryGithubMetaphor - Translate tokens by values on a directory level.
func detokenizeGitopsMetaphor(path string, tokens *MetaphorTokenValues) filepath.WalkFunc {
	return filepath.WalkFunc(func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if !!fi.IsDir() {
			return nil
		}
		
		// var matched bool
		matched, _ := filepath.Match("*", fi.Name())
		
		if matched {
			// ignore .git files
			if !strings.Contains(path, "/.git/") {
				
				read, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				
				// todo reduce to terraform tokens by moving to helm chart?
				newContents := string(read)
				newContentData, err := renderGoTemplating(tokens, newContents)
				
				if err != nil {
					return err
				}
				
				return os.WriteFile(path, newContentData, 0)
				
				//newContents = strings.Replace(newContents, "<CLOUD_REGION>", tokens.CloudRegion, -1)
				//newContents = strings.Replace(newContents, "<CLUSTER_NAME>", tokens.ClusterName, -1)
				//newContents = strings.Replace(newContents, "<CONTAINER_REGISTRY_URL>", tokens.ContainerRegistryURL, -1) // todo need to fix metaphor repo names
				//newContents = strings.Replace(newContents, "<DOMAIN_NAME>", tokens.DomainName, -1)
				//newContents = strings.Replace(newContents, "<METAPHOR_DEVELOPMENT_INGRESS_URL>", tokens.MetaphorDevelopmentIngressURL, -1)
				//newContents = strings.Replace(newContents, "<METAPHOR_PRODUCTION_INGRESS_URL>", tokens.MetaphorProductionIngressURL, -1)
				//newContents = strings.Replace(newContents, "<METAPHOR_STAGING_INGRESS_URL>", tokens.MetaphorStagingIngressURL, -1)
				//
				
			}
		}
		
		return nil
	})
}

func renderGoTemplating(tokens Tokens, content string) ([]byte, error) {
	content = replaceTemplateVariables(content, tokens)
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
