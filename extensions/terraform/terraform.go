/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package terraform

import (
	"fmt"
	"os"

	log "github.com/rs/zerolog/log"
)

func initActionAutoApprove(terraformClientPath string, tfAction, tfEntrypoint string, tfEnvs map[string]string) error {
	log.Printf("initActionAutoApprove - action: %s entrypoint: %s", tfAction, tfEntrypoint)

	err := os.Chdir(tfEntrypoint)
	if err != nil {
		log.Printf("error: could not change to directory %s", tfEntrypoint)
		return err
	}
	err = ExecShellWithVars(tfEnvs, terraformClientPath, "init", "-force-copy")
	if err != nil {
		log.Printf("error: terraform init for %s failed: %s", tfEntrypoint, err)
		return err
	}

	err = ExecShellWithVars(tfEnvs, terraformClientPath, tfAction, "-auto-approve")
	if err != nil {
		log.Printf("error: terraform %s -auto-approve for %s failed %s", tfAction, tfEntrypoint, err)
		return err
	}
	os.RemoveAll(fmt.Sprintf("%s/.terraform/", tfEntrypoint))
	os.Remove(fmt.Sprintf("%s/.terraform.lock.hcl", tfEntrypoint))
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
