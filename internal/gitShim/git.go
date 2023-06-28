/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package gitShim

import (
	"fmt"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
)

// PullWithAuth
func PullWithAuth(repo *git.Repository, remote string, branch string, auth transport.AuthMethod) error {
	w, _ := repo.Worktree()
	branchName := plumbing.NewBranchReferenceName(branch)
	err := w.Pull(&git.PullOptions{
		RemoteName:    remote,
		ReferenceName: branchName,
		Auth:          auth,
	})
	if err != nil {
		return fmt.Errorf("error during git pull: %s", err)
	}

	return nil
}
