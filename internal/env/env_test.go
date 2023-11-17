package env

import (
	"os"
	"testing"
)

func TestEnv(t *testing.T) {
	os.Setenv("SERVER_PORT", "8081")
	os.Setenv("K1_ACCESS_TOKEN", "k1_access_token")
	os.Setenv("MONGODB_HOST", "mongodb_host")
	os.Setenv("MONGODB_HOST_TYPE", "mongodb_host_type")
	os.Setenv("MONGODB_USERNAME", "mongodb_username")
	os.Setenv("MONGODB_PASSWORD", "mongodb_password")
	os.Setenv("KUBEFIRST_VERSION", "development")
	os.Setenv("CLOUD_PROVIDER", "cloud_provider")
	os.Setenv("CLUSTER_ID", "cluster_id")
	os.Setenv("CLUSTER_TYPE", "cluster_type")
	os.Setenv("DOMAIN_NAME", "domain_name")
	os.Setenv("GIT_PROVIDER", "git_provider")
	os.Setenv("INSTALL_METHOD", "install_method")
	os.Setenv("KUBEFIRST_TEAM", "kubefirst_team")
	os.Setenv("KUBEFIRST_TEAM_INFO", "kubefirst_team_info")
	os.Setenv("AWS_REGION", "aws_region")
	os.Setenv("AWS_PROFILE", "aws_profile")
	os.Setenv("IS_CLUSTER_ZERO", "true")
	os.Setenv("IN_CLUSTER", "false")
	os.Setenv("ENTERPRISE_API_URL", "enterprise_api_url")

	defer func() {
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("K1_ACCESS_TOKEN")
		os.Unsetenv("MONGODB_HOST")
		os.Unsetenv("MONGODB_HOST_TYPE")
		os.Unsetenv("MONGODB_USERNAME")
		os.Unsetenv("MONGODB_PASSWORD")
		os.Unsetenv("KUBEFIRST_VERSION")
		os.Unsetenv("CLOUD_PROVIDER")
		os.Unsetenv("CLUSTER_ID")
		os.Unsetenv("CLUSTER_TYPE")
		os.Unsetenv("DOMAIN_NAME")
		os.Unsetenv("GIT_PROVIDER")
		os.Unsetenv("INSTALL_METHOD")
		os.Unsetenv("KUBEFIRST_TEAM")
		os.Unsetenv("KUBEFIRST_TEAM_INFO")
		os.Unsetenv("AWS_REGION")
		os.Unsetenv("AWS_PROFILE")
		os.Unsetenv("IS_CLUSTER_ZERO")
		os.Unsetenv("IN_CLUSTER")
		os.Unsetenv("ENTERPRISE_API_URL")
	}()

	env := Env{}
	env, err := GetEnv(true)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if env.ServerPort != 8081 {
		t.Errorf("expected ServerPort to be 8081, but got %d", env.ServerPort)
	}

	if env.K1AccessToken != "k1_access_token" {
		t.Errorf("expected K1AccessToken to be 'k1_access_token', but got '%s'", env.K1AccessToken)
	}

	if env.MongoDBHost != "mongodb_host" {
		t.Errorf("expected MongoDBHost to be 'mongodb_host', but got '%s'", env.MongoDBHost)
	}

	if env.MongoDBHostType != "mongodb_host_type" {
		t.Errorf("expected MongoDBHostType to be 'mongodb_host_type', but got '%s'", env.MongoDBHostType)
	}

	if env.MongoDBUsername != "mongodb_username" {
		t.Errorf("expected MongoDBUsername to be 'mongodb_username', but got '%s'", env.MongoDBUsername)
	}

	if env.MongoDBPassword != "mongodb_password" {
		t.Errorf("expected MongoDBPassword to be 'mongodb_password', but got '%s'", env.MongoDBPassword)
	}

	if env.KubefirstVersion != "development" {
		t.Errorf("expected KubefirstVersion to be 'development', but got '%s'", env.KubefirstVersion)
	}

	if env.CloudProvider != "cloud_provider" {
		t.Errorf("expected CloudProvider to be 'cloud_provider', but got '%s'", env.CloudProvider)
	}

	if env.ClusterId != "cluster_id" {
		t.Errorf("expected ClusterId to be 'cluster_id', but got '%s'", env.ClusterId)
	}

	if env.ClusterType != "cluster_type" {
		t.Errorf("expected ClusterType to be 'cluster_type', but got '%s'", env.ClusterType)
	}

	if env.DomainName != "domain_name" {
		t.Errorf("expected DomainName to be 'domain_name', but got '%s'", env.DomainName)
	}

	if env.GitProvider != "git_provider" {
		t.Errorf("expected GitProvider to be 'git_provider', but got '%s'", env.GitProvider)
	}

	if env.InstallMethod != "install_method" {
		t.Errorf("expected InstallMethod to be 'install_method', but got '%s'", env.InstallMethod)
	}

	if env.KubefirstTeam != "kubefirst_team" {
		t.Errorf("expected KubefirstTeam to be 'kubefirst_team', but got '%s'", env.KubefirstTeam)
	}

	if env.KubefirstTeamInfo != "kubefirst_team_info" {
		t.Errorf("expected KubefirstTeamInfo to be 'kubefirst_team_info', but got '%s'", env.KubefirstTeamInfo)
	}

	if env.AWSRegion != "aws_region" {
		t.Errorf("expected AWSRegion to be 'aws_region', but got '%s'", env.AWSRegion)
	}

	if env.AWSProfile != "aws_profile" {
		t.Errorf("expected AWSProfile to be 'aws_profile', but got '%s'", env.AWSProfile)
	}

	if env.IsClusterZero != true {
		t.Errorf("expected IsClusterZero to be true, but got false")
	}

	if env.InCluster != false {
		t.Errorf("expected InCluster to be false, but got true")
	}

	if env.EnterpriseApiUrl != "enterprise_api_url" {
		t.Errorf("expected EnterpriseApiUrl to be 'enterprise_api_url', but got '%s'", env.EnterpriseApiUrl)
	}
}