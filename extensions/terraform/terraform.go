/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package terraform

import (
	"fmt"
	"os"

	"github.com/konstructio/kubefirst-api/internal"
	log "github.com/rs/zerolog/log"
)

func initActionAutoApprove(terraformClientPath string, tfAction, tfEntrypoint string, tfEnvs map[string]string) error {
	log.Printf("initActionAutoApprove - action: %s entrypoint: %s", tfAction, tfEntrypoint)

	err := os.Chdir(tfEntrypoint)
	if err != nil {
		log.Error().Msgf("error: could not change to directory %s", tfEntrypoint)
		return fmt.Errorf("error: could not change to directory %s: %w", tfEntrypoint, err)
	}

	err = internal.ExecShellWithVars(tfEnvs, terraformClientPath, "init", "-force-copy")
	if err != nil {
		log.Error().Msgf("error: terraform init for %s failed: %s", tfEntrypoint, err)
		return fmt.Errorf("error: terraform init for %s failed: %w", tfEntrypoint, err)
	}

	err = internal.ExecShellWithVars(tfEnvs, terraformClientPath, tfAction, "-auto-approve")
	if err != nil {
		log.Error().Msgf("error: terraform %s -auto-approve for %s failed %s", tfAction, tfEntrypoint, err)
		return fmt.Errorf("error: terraform %s -auto-approve for %s failed: %w", tfAction, tfEntrypoint, err)
	}

	os.RemoveAll(tfEntrypoint + "/.terraform/")
	os.Remove(tfEntrypoint + "/.terraform.lock.hcl")
	return nil
}

func InitApplyAutoApprove(terraformClientPath string, tfEntrypoint string, tfEnvs map[string]string) error {
	tfAction := "apply"
	err := initActionAutoApprove(terraformClientPath, tfAction, tfEntrypoint, tfEnvs)
	if err != nil {
		return err
	}
	return nil
}

func InitDestroyAutoApprove(terraformClientPath string, tfEntrypoint string, tfEnvs map[string]string) error {
	tfAction := "destroy"
	err := initActionAutoApprove(terraformClientPath, tfAction, tfEntrypoint, tfEnvs)
	if err != nil {
		return err
	}
	return nil
}
