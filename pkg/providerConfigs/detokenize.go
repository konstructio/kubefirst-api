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
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"text/template"
)

const (
	FilePathPattern = "*"
	RegexPattern    = "<([A-Z_0-9]+)>"
	leftDelimeter   = "[["
	rightDelimeter  = "]]"
)

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
			
			newContents := string(read)
			//origin issuer defines which annotations should be on ingresses
			if useCloudflareOriginIssuer {
				tokens.CertManagerIssuerAnnotation1 = "cert-manager.io/issuer: cloudflare-origin-issuer"
				tokens.CertManagerIssuerAnnotation2 = "cert-manager.io/issuer-kind: OriginIssuer"
				tokens.CertManagerIssuerAnnotation3 = "cert-manager.io/issuer-group: cert-manager.k8s.cloudflare.com"
				tokens.CertManagerIssuerAnnotation4 = "external-dns.alpha.kubernetes.io/cloudflare-proxied: \"true\""
			} else {
				tokens.CertManagerIssuerAnnotation1 = "cert-manager.io/cluster-issuer: \"letsencrypt-prod\""
			}
			
			newContents = strings.TrimSpace(newContents)
			
			// The fqdn is used by metaphor/argo to choose the appropriate url for cicd operations.
			if gitProtocol == "https" {
				tokens.GitFqdn = fmt.Sprintf("https://%v.com/", tokens.GitProvider)
			} else {
				tokens.GitFqdn = fmt.Sprintf("git@%v.com:", tokens.GitProvider)
			}
			
			r := regexp.MustCompile(RegexPattern)
			newContents = r.ReplaceAllStringFunc(newContents, toTemplateVariable)
			buff := bytes.NewBufferString(newContents)
			
			parsedTemplate, err := parseTemplate(buff.String())
			if err != nil {
				return err
			}
			
			newBuff := bytes.NewBuffer([]byte{})
			if err = executeTemplate(parsedTemplate, newBuff, tokens); err != nil {
				return err
			}
			
			return os.WriteFile(path, newBuff.Bytes(), 0)
		}
		return nil
	})
}

func parseTemplate(content string) (*template.Template, error) {
	t := template.New("gitops-template").Delims(leftDelimeter, rightDelimeter)
	return t.Parse(content)
}

func executeTemplate(t *template.Template, writer io.Writer, tokens *GitopsDirectoryValues) error {
	return t.Execute(writer, tokens)
}

func replaceTemplateVariables(content string) string {
	regex := regexp.MustCompile(RegexPattern)
	return regex.ReplaceAllStringFunc(content, toTemplateVariable)
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
			return fmt.Sprintf("%s .%s %s", leftDelimeter, field.Name, rightDelimeter)
		}
	}
	
	// If no match found, return an error placeholder
	return "<variable not found>"
}

//// Used this to convert all <REPLACE_TOKEN> to go template-able var using left and right delimeters allowing the tokens
//// ex: <REPLACE_TOKEN> -> << .ReplaceToken >>
//// The issue with this is where
//func toTemplateVariable(v string) string {
//	newVar := ""
//	replacerVarSplit := strings.Split(v, "_")
//	parts := len(replacerVarSplit)
//	r := regexp.MustCompile("<|>")
//	for i, s := range replacerVarSplit {
//		caser := cases.Title(language.English)
//		if i == 0 {
//			newVar = fmt.Sprintf("%s .", leftDelimeter)
//		}
//
//		if s == "URL>" {
//			newVar += "URL"
//		} else {
//			newVar += r.ReplaceAllString(caser.String(s), "")
//		}
//
//		if i == parts-1 {
//			newVar += fmt.Sprintf(" %s", rightDelimeter)
//		}
//	}
//
//	return newVar
//}

// DetokenizeAdditionalPath - Translate tokens by values on a given path
func DetokenizeAdditionalPath(path string, tokens *GitopsDirectoryValues) error {
	err := filepath.Walk(path, detokenizeAdditionalPath(path, tokens))
	if err != nil {
		return err
	}
	
	return nil
}

// detokenizeAdditionalPath temporary addition to handle detokenizing additional files
func detokenizeAdditionalPath(path string, tokens *GitopsDirectoryValues) filepath.WalkFunc {
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
				read, err := ioutil.ReadFile(path)
				if err != nil {
					return err
				}
				
				newContents := string(read)
				newContents = strings.Replace(newContents, "<GITLAB_OWNER>", tokens.GitlabOwner, -1)
				renderedContents, err := renderGoTemplating(tokens, newContents)
				
				err = os.WriteFile(path, renderedContents, 0)
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
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
				newContents = strings.Replace(newContents, "<CLOUD_REGION>", tokens.CloudRegion, -1)
				newContents = strings.Replace(newContents, "<CLUSTER_NAME>", tokens.ClusterName, -1)
				newContents = strings.Replace(newContents, "<CONTAINER_REGISTRY_URL>", tokens.ContainerRegistryURL, -1) // todo need to fix metaphor repo names
				newContents = strings.Replace(newContents, "<DOMAIN_NAME>", tokens.DomainName, -1)
				newContents = strings.Replace(newContents, "<METAPHOR_DEVELOPMENT_INGRESS_URL>", tokens.MetaphorDevelopmentIngressURL, -1)
				newContents = strings.Replace(newContents, "<METAPHOR_PRODUCTION_INGRESS_URL>", tokens.MetaphorProductionIngressURL, -1)
				newContents = strings.Replace(newContents, "<METAPHOR_STAGING_INGRESS_URL>", tokens.MetaphorStagingIngressURL, -1)
				
				renderedContents, err := renderGoTemplating(tokens, newContents)
				
				err = os.WriteFile(path, renderedContents, 0)
				if err != nil {
					return err
				}
				
			}
		}
		
		return nil
	})
}

func renderGoTemplating(tokens interface{}, content string) ([]byte, error) {
	customTokens := make(map[string]interface{})
	// A new Buffer so we have an io.Writer for the CustomTemplateValues
	buff := bytes.NewBuffer([]byte(content))
	fmt.Println("BUFF", string(buff.Bytes()))
	
	if err := template.New("gitops-template").
		Delims(leftDelimeter, rightDelimeter).
		Execute(buff, &customTokens); err != nil {
		return nil, err
	}
	
	return buff.Bytes(), nil
	
}
