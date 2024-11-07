/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package terraform

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	pkg "github.com/konstructio/kubefirst-api/internal"
	"github.com/rs/zerolog/log"
)

func initActionAutoApprove(terraformClientPath, tfAction, tfEntrypoint string, tfEnvs map[string]string) error {
	log.Printf("initActionAutoApprove - action: %s entrypoint: %s", tfAction, tfEntrypoint)

	err := os.Chdir(tfEntrypoint)
	if err != nil {
		log.Info().Msg("error: could not change to directory " + tfEntrypoint)
		return fmt.Errorf("failed to change directory to %q: %w", tfEntrypoint, err)
	}
	err = pkg.ExecShellWithVars(tfEnvs, terraformClientPath, "init", "-force-copy", "-no-color")
	if err != nil {
		log.Printf("error: terraform init for %s failed: %s", tfEntrypoint, err)
		return fmt.Errorf("terraform init for %q failed: %w", tfEntrypoint, err)
	}

	err = pkg.ExecShellWithVars(tfEnvs, terraformClientPath, tfAction, "-auto-approve", fmt.Sprintf("-parallelism=%d", runtime.NumCPU()*2), "-no-color")
	if err != nil {
		log.Printf("error: terraform %s -auto-approve for %s failed %s", tfAction, tfEntrypoint, err)
		return fmt.Errorf("terraform %s -auto-approve for %q failed: %w", tfAction, tfEntrypoint, err)
	}
	os.RemoveAll(fmt.Sprintf("%s/.terraform/", tfEntrypoint))
	os.Remove(fmt.Sprintf("%s/.terraform.lock.hcl", tfEntrypoint))
	return nil
}

func InitApplyAutoApprove(terraformClientPath, tfEntrypoint string, tfEnvs map[string]string) error {
	tfAction := "apply"
	err := initActionAutoApprove(terraformClientPath, tfAction, tfEntrypoint, tfEnvs)
	if err != nil {
		return fmt.Errorf("unable to init and auto-apply terraform configuration: %w", err)
	}
	return nil
}

func InitDestroyAutoApprove(terraformClientPath, tfEntrypoint string, tfEnvs map[string]string) error {
	tfAction := "destroy"
	err := initActionAutoApprove(terraformClientPath, tfAction, tfEntrypoint, tfEnvs)
	if err != nil {
		return fmt.Errorf("unable to init and auto-destroy terraform configuration: %w", err)
	}
	return nil
}

// todo need to write something that outputs -json type and can get multiple values
func OutputSingleValue(terraformClientPath, directory, outputName string) {
	os.Chdir(directory)

	var tfOutput bytes.Buffer
	tfOutputCmd := exec.Command(terraformClientPath, "output", outputName)
	tfOutputCmd.Stdout = &tfOutput
	tfOutputCmd.Stderr = os.Stderr
	err := tfOutputCmd.Run()
	if err != nil {
		log.Error().Err(err).Msg("failed to call tfOutputCmd.Run()")
	}

	log.Print("tfOutput is: ", tfOutput.String())
}
