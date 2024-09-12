package env

import (
	"fmt"

	env "github.com/caarlos0/env/v10"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

type Env struct {
	ServerPort            string `env:"SERVER_PORT" envDefault:"8081"`
	K1AccessToken         string `env:"K1_ACCESS_TOKEN"`
	KubefirstVersion      string `env:"KUBEFIRST_VERSION" envDefault:"main"`
	CloudProvider         string `env:"CLOUD_PROVIDER"`
	ClusterID             string `env:"CLUSTER_ID"`
	ClusterType           string `env:"CLUSTER_TYPE"`
	DomainName            string `env:"DOMAIN_NAME"`
	GitProvider           string `env:"GIT_PROVIDER"`
	InstallMethod         string `env:"INSTALL_METHOD"`
	KubefirstTeam         string `env:"KUBEFIRST_TEAM" envDefault:"undefined"`
	KubefirstTeamInfo     string `env:"KUBEFIRST_TEAM_INFO"`
	AWSRegion             string `env:"AWS_REGION"`
	AWSProfile            string `env:"AWS_PROFILE"`
	IsClusterZero         string `env:"IS_CLUSTER_ZERO" envDefault:"true"`
	ParentClusterID       string `env:"PARENT_CLUSTER_ID"`
	InCluster             string `env:"IN_CLUSTER" envDefault:"false"`
	EnterpriseAPIURL      string `env:"ENTERPRISE_API_URL"`
	K1LocalDebug          string `env:"K1_LOCAL_DEBUG"`
	K1LocalKubeconfigPath string `env:"K1_LOCAL_KUBECONFIG_PATH"`
}

func GetEnv(silent bool) (Env, error) {
	err := godotenv.Load(".env")

	if err != nil && !silent {
		log.Info().Msg("error loading .env file, using local environment variables")
	}

	environment := Env{}
	if err := env.Parse(&environment); err != nil {
		return Env{}, fmt.Errorf("error parsing environment variables: %w", err)
	}

	return environment, nil
}
