package controller

import (
	"os"
	"path/filepath"
	"strings"
)

func (clctrl *ClusterController) TerraformPrep() error {

	path := clctrl.ProviderConfig.GitopsDir + "/terraform"
	err := filepath.Walk(path, detokenizeterraform(path, clctrl))
	if err != nil {
		return err
	}
	return nil
}

func detokenizeterraform(path string, clctrl *ClusterController) filepath.WalkFunc {
	return filepath.WalkFunc(func(path string, fi os.FileInfo, err error) error {

		if fi.IsDir() {
			return nil
		}

		read, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		replacer := strings.NewReplacer(
			"<ADMIN_TEAM>", clctrl.AdminTeamName,
			"<DEVELOPER_TEAM>", clctrl.DeveloperTeamName,
			"<METAPHOR_REPO_NAME>", clctrl.MetaphorRepoName,
			"<GIT_REPO_NAME>", clctrl.GitopsRepoName,
		)

		newContents := replacer.Replace(string(read))

		err = os.WriteFile(path, []byte(newContents), 0)
		if err != nil {
			return err
		}

		return nil
	})

}
