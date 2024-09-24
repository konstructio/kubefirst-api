package azure

import (
	"fmt"

	"github.com/konstructio/kubefirst-api/internal/controller"
	"github.com/konstructio/kubefirst-api/internal/secrets"
	pkgtypes "github.com/konstructio/kubefirst-api/pkg/types"
	"github.com/rs/zerolog/log"
)

func CreateAKSCluster(definition *pkgtypes.ClusterDefinition) error {
	ctrl := controller.ClusterController{}

	log.Info().Msg("initializing controller")
	err := ctrl.InitController(definition)
	if err != nil {
		return err
	}

	ctrl.Cluster.InProgress = true
	log.Info().Msg("updating cluster secrets")
	err = secrets.UpdateCluster(ctrl.KubernetesClient, ctrl.Cluster)
	if err != nil {
		return err
	}

	log.Info().Msg("downling tools")
	err = ctrl.DownloadTools(ctrl.ProviderConfig.ToolsDir)
	if err != nil {
		ctrl.HandleError(err.Error())
		return err
	}

	log.Info().Msg("checking domain liveness")
	err = ctrl.DomainLivenessTest()
	if err != nil {
		ctrl.HandleError(err.Error())
		return err
	}

	log.Info().Msg("creating state store")
	err = ctrl.StateStoreCredentials()
	if err != nil {
		ctrl.HandleError(err.Error())
		return err
	}

	log.Info().Msg("initializing git")
	err = ctrl.GitInit()
	if err != nil {
		ctrl.HandleError(err.Error())
		return err
	}

	log.Info().Msg("initializing bot")
	err = ctrl.InitializeBot()
	if err != nil {
		ctrl.HandleError(err.Error())
		return err
	}

	log.Info().Msg("repository prep")
	fmt.Println("repo prep")
	err = ctrl.RepositoryPrep()
	if err != nil {
		ctrl.HandleError(err.Error())
		return err
	}

	log.Info().Msg("running git terraform")
	err = ctrl.RunGitTerraform()
	if err != nil {
		ctrl.HandleError(err.Error())
		return err
	}

	log.Info().Msg("pushing repository")
	err = ctrl.RepositoryPush()
	if err != nil {
		ctrl.HandleError(err.Error())
		return err
	}

	fmt.Println(333)

	return nil
}
