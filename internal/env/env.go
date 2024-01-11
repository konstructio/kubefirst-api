package env

import (
	env "github.com/caarlos0/env/v10"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
)

type Env struct {
	ServerPort        string `env:"SERVER_PORT" envDefault:"8088"`
	K1AccessToken     string `env:"K1_ACCESS_TOKEN"`
	MongoDBHost       string `env:"MONGODB_HOST,notEmpty"`
	MongoDBHostType   string `env:"MONGODB_HOST_TYPE,notEmpty"`
	MongoDBUsername   string `env:"MONGODB_USERNAME,notEmpty"`
	MongoDBPassword   string `env:"MONGODB_PASSWORD,notEmpty"`
	KubefirstVersion  string `env:"KUBEFIRST_VERSION" envDefault:"development"`
	CloudProvider     string `env:"CLOUD_PROVIDER"`
	ClusterId         string `env:"CLUSTER_ID"`
	ClusterType       string `env:"CLUSTER_TYPE"`
	DomainName        string `env:"DOMAIN_NAME"`
	GitProvider       string `env:"GIT_PROVIDER"`
	InstallMethod     string `env:"INSTALL_METHOD"`
	KubefirstTeam     string `env:"KUBEFIRST_TEAM" envDefault:"undefined"`
	KubefirstTeamInfo string `env:"KUBEFIRST_TEAM_INFO"`
	AWSRegion         string `env:"AWS_REGION"`
	AWSProfile        string `env:"AWS_PROFILE"`
	IsClusterZero     string `env:"IS_CLUSTER_ZERO"`
	ParentClusterId   string `env:"PARENT_CLUSTER_ID"`
	InCluster         string `env:"IN_CLUSTER" envDefault:"false"`
	EnterpriseApiUrl  string `env:"ENTERPRISE_API_URL"`
}

func GetEnv(silent bool) (Env, error) {
	err := godotenv.Load(".env")

	if err != nil && !silent {
		log.Info("error loading .env file, using local environment variables")
	}

	environment := Env{}
	err = env.Parse(&environment)
	if err != nil {
		return Env{}, err
	}

	return environment, nil
}
