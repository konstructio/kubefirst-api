/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package gitClient //nolint:revive,stylecheck // allowed temporarily during code reorg

import (
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/mod/semver"

	"github.com/go-git/go-git/v5"
	gitConfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	githttps "github.com/go-git/go-git/v5/plumbing/transport/http"
)

func Clone(gitRef, repoLocalPath, repoURL string) (*git.Repository, error) {
	// kubefirst tags do not contain a `v` prefix, to use the library requires the v to be valid
	isSemVer := semver.IsValid(gitRef)

	var refName plumbing.ReferenceName

	if isSemVer {
		refName = plumbing.NewTagReferenceName(gitRef)
	} else {
		refName = plumbing.NewBranchReferenceName(gitRef)
	}

	repo, err := git.PlainClone(repoLocalPath, false, &git.CloneOptions{
		URL:           repoURL,
		ReferenceName: refName,
		SingleBranch:  true,
	})
	if err != nil {
		return nil, fmt.Errorf("error cloning repository from URL %q: %w", repoURL, err)
	}

	return repo, nil
}

func ClonePrivateRepo(gitRef, repoLocalPath, repoURL, userName, token string) (*git.Repository, error) {
	// kubefirst tags do not contain a `v` prefix, to use the library requires the v to be valid
	isSemVer := semver.IsValid(gitRef)

	var refName plumbing.ReferenceName

	if isSemVer {
		refName = plumbing.NewTagReferenceName(gitRef)
	} else {
		refName = plumbing.NewBranchReferenceName(gitRef)
	}

	repo, err := git.PlainClone(repoLocalPath, false, &git.CloneOptions{
		URL:           repoURL,
		ReferenceName: refName,
		SingleBranch:  true,
		Auth: &githttps.BasicAuth{
			Username: userName,
			Password: token,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("error cloning private repository from URL %q: %w", repoURL, err)
	}

	return repo, nil
}

func CloneRefSetMain(gitRef, repoLocalPath, repoURL string) (*git.Repository, error) {
	log.Info().Msgf("cloning url: %s - git ref: %s", repoURL, gitRef)

	repo, err := Clone(gitRef, repoLocalPath, repoURL)
	if err != nil {
		log.Error().Msgf("error cloning repo (%s) at: %s, err: %v", repoURL, repoLocalPath, err)
		return nil, fmt.Errorf("error during clone of repository %q to path %q: %w", repoURL, repoLocalPath, err)
	}

	if gitRef != "main" {
		repo, err = SetRefToMainBranch(repo)
		if err != nil {
			return nil, fmt.Errorf("error setting main branch from git ref: %q: %w", gitRef, err)
		}

		// remove old git ref
		err = repo.Storer.RemoveReference(plumbing.NewBranchReferenceName(gitRef))
		if err != nil {
			return nil, fmt.Errorf("error removing previous git reference for %q: %w", gitRef, err)
		}
	}
	return repo, nil
}

// SetRefToMainBranch sets the provided gitRef (branch or tag) to the main branch
func SetRefToMainBranch(repo *git.Repository) (*git.Repository, error) {
	w, _ := repo.Worktree()
	branchName := plumbing.NewBranchReferenceName("main")
	headRef, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("error getting head reference: %w", err)
	}

	ref := plumbing.NewHashReference(branchName, headRef.Hash())
	err = repo.Storer.SetReference(ref)
	if err != nil {
		return nil, fmt.Errorf("error storing reference for main branch: %w", err)
	}

	err = w.Checkout(&git.CheckoutOptions{Branch: ref.Name()})
	if err != nil {
		return nil, fmt.Errorf("error checking out main branch: %w", err)
	}
	return repo, nil
}

func AddRemote(newGitRemoteURL, remoteName string, repo *git.Repository) error {
	log.Info().Msgf("git remote add %s %s", remoteName, newGitRemoteURL)
	_, err := repo.CreateRemote(&gitConfig.RemoteConfig{
		Name: remoteName,
		URLs: []string{newGitRemoteURL},
	})
	if err != nil {
		log.Info().Msgf("error creating remote %s at: %s", remoteName, newGitRemoteURL)
		return fmt.Errorf("error creating remote %q to URL %q: %w", remoteName, newGitRemoteURL, err)
	}
	return nil
}

func Commit(repo *git.Repository, commitMsg string) error {
	w, err := repo.Worktree()
	if err != nil {
		log.Error().Msgf("error getting worktree: %s", err)
		return fmt.Errorf("error getting worktree: %w", err)
	}

	log.Info().Msg(commitMsg)
	w.AddGlob(".")

	_, err = w.Commit(commitMsg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "kbot",
			Email: "kbot@kubefirst.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		log.Error().Msgf("error committing in repo: %s", err)
		return fmt.Errorf("error committing changes to repo: %w", err)
	}

	return nil
}

func Pull(repo *git.Repository, remote, branch string) error {
	w, _ := repo.Worktree()
	branchName := plumbing.NewBranchReferenceName(branch)
	err := w.Pull(&git.PullOptions{
		RemoteName:    remote,
		ReferenceName: branchName,
	})
	if err != nil {
		return fmt.Errorf("error during git pull from remote %q for branch %q: %w", remote, branch, err)
	}

	return nil
}
