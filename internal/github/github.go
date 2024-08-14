/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package github

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/go-github/v45/github"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
)

type Session struct {
	context     context.Context
	staticToken oauth2.TokenSource
	oauthClient *http.Client
	gitClient   *github.Client
}

// New - Create a new client for github wrapper
func New(token string) Session {
	var gSession Session
	gSession.context = context.Background()
	gSession.staticToken = oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	gSession.oauthClient = oauth2.NewClient(gSession.context, gSession.staticToken)
	gSession.gitClient = github.NewClient(gSession.oauthClient)
	return gSession
}

func (g Session) CreateWebhookRepo(org, repo, hookName, hookURL, hookSecret string, hookEvents []string) error {
	input := &github.Hook{
		Name:   &hookName,
		Events: hookEvents,
		Config: map[string]interface{}{
			"content_type": "json",
			"insecure_ssl": 0,
			"url":          hookURL,
			"secret":       hookSecret,
		},
	}

	hook, _, err := g.gitClient.Repositories.CreateHook(g.context, org, repo, input)
	if err != nil {
		return fmt.Errorf("error when creating a webhook for repo %s/%s: %w", org, repo, err)
	}

	log.Info().Msgf("Successfully created hook (id: %v)", hook.GetID())
	return nil
}

// CreatePrivateRepo - Use github API to create a private repo
func (g Session) CreatePrivateRepo(org, name, description string) error {
	if name == "" {
		return fmt.Errorf("no name: New repos must be given a name")
	}

	isPrivate := true
	autoInit := true
	r := &github.Repository{
		Name:        &name,
		Private:     &isPrivate,
		Description: &description,
		AutoInit:    &autoInit,
	}

	repo, _, err := g.gitClient.Repositories.Create(g.context, org, r)
	if err != nil {
		return fmt.Errorf("error creating private repo %q in organization %q: %w", name, org, err)
	}

	log.Info().Msgf("Successfully created new repo: %q", repo.GetName())
	return nil
}

// RemoveRepo Removes a repository based on repository owner and name. It returns github.Response that hold http data,
// as http status code, the caller can make use of the http status code to validate the response.
func (g Session) RemoveRepo(owner, name string) (*github.Response, error) {
	if owner == "" {
		return nil, fmt.Errorf("removal failed: a repository owner is required")
	}

	if name == "" {
		return nil, fmt.Errorf("removal failed: a repository name is required")
	}

	resp, err := g.gitClient.Repositories.Delete(g.context, owner, name)
	if err != nil {
		return resp, fmt.Errorf("error removing private repo %q from owner %q: %w", name, owner, err)
	}

	log.Info().Msgf("Successfully removed repo: %v", name)
	return resp, nil
}

// RemoveTeam - Remove  a team
func (g Session) RemoveTeam(owner, team string) error {
	if team == "" {
		return fmt.Errorf("team removal failed: team name is required")
	}

	_, err := g.gitClient.Teams.DeleteTeamBySlug(g.context, owner, team)
	if err != nil {
		return fmt.Errorf("error removing team %q from owner %q: %w", team, owner, err)
	}

	log.Info().Msgf("Successfully removed team: %v", team)
	return nil
}

// GetRepo - Returns  a repo
func (g Session) GetRepo(owner, name string) (*github.Repository, error) {
	if name == "" {
		return nil, fmt.Errorf("get repo: name is empty")
	}

	repo, _, err := g.gitClient.Repositories.Get(g.context, owner, name)
	if err != nil {
		return nil, fmt.Errorf("error fetching private repo %q for owner %q: %w", name, owner, err)
	}

	log.Info().Msgf("Successfully fetched repo: %q", repo.GetName())
	return repo, nil
}

// AddSSHKey - Add ssh keys to a user account to allow kubefirst installer
// to use its own token during installation
func (g Session) AddSSHKey(keyTitle, publicKey string) (*github.Key, error) {
	log.Printf("Add SSH key to user account on behalf of kubefirst")
	key, _, err := g.gitClient.Users.CreateKey(g.context, &github.Key{Title: &keyTitle, Key: &publicKey})
	if err != nil {
		return nil, fmt.Errorf("error adding SSH Key with title %q: %w", keyTitle, err)
	}
	return key, nil
}

// RemoveSSHKey - Removes SSH Key from github user
func (g Session) RemoveSSHKey(keyID int64) error {
	log.Printf("Remove SSH key to user account on behalf of kubefirst")
	_, err := g.gitClient.Users.DeleteKey(g.context, keyID)
	if err != nil {
		return fmt.Errorf("error removing SSH Key with ID %d: %w", keyID, err)
	}
	return nil
}

// RemoveSSHKeyByPublicKey deletes a GitHub key that matches the provided public key.
func (g Session) RemoveSSHKeyByPublicKey(user, publicKey string) error {
	keys, resp, err := g.gitClient.Users.ListKeys(g.context, user, &github.ListOptions{})
	if err != nil {
		return fmt.Errorf("unable to retrieve SSH keys for user %q: %w", user, err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unable to retrieve SSH keys for user %q, http code is: %d", user, resp.StatusCode)
	}

	for _, key := range keys {
		// as https://pkg.go.dev/golang.org/x/crypto/ssh@v0.0.0-20220722155217-630584e8d5aa#MarshalAuthorizedKey
		// documentation describes, the Marshall ssh key function adds extra new line at the end of the key id
		if key.GetKey()+"\n" == publicKey {
			resp, err := g.gitClient.Users.DeleteKey(g.context, key.GetID())
			if err != nil {
				return fmt.Errorf("error deleting SSH key with ID %d: %w", key.GetID(), err)
			}

			if resp.StatusCode != http.StatusNoContent {
				return fmt.Errorf("unable to delete SSH key with ID %d, http code is: %d", key.GetID(), resp.StatusCode)
			}
		}
	}

	return nil
}

func (g Session) CreatePR(
	branchName string,
	repoName string,
	gitHubUser string,
	baseBranch string,
	title string,
	body string,
) (*github.PullRequest, error) {
	head := branchName
	prData := github.NewPullRequest{
		Title: &title,
		Head:  &head,
		Body:  &body,
		Base:  &baseBranch,
	}

	pullRequest, resp, err := g.gitClient.PullRequests.Create(
		context.Background(),
		gitHubUser,
		repoName,
		&prData,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating pull request for repo %q by user %q: %w", repoName, gitHubUser, err)
	}

	log.Info().Msgf("pull request create response http code: %d", resp.StatusCode)

	return pullRequest, nil
}

func (g Session) CommentPR(pullRequesrt *github.PullRequest, gitHubUser, body string) error {
	issueComment := github.IssueComment{
		Body: &body,
	}

	_, resp, err := g.gitClient.Issues.CreateComment(
		context.Background(),
		gitHubUser,
		"gitops",
		*pullRequesrt.Number,
		&issueComment,
	)
	if err != nil {
		return fmt.Errorf("error creating pull request comment for pull request %d: %w", *pullRequesrt.Number, err)
	}
	log.Printf("pull request comment response http code: %d", resp.StatusCode)

	return nil
}

// SearchWordInPullRequestComment look for a specific sentence in a GitHub Pull Request comment
func (g Session) SearchWordInPullRequestComment(gitHubUser string,
	gitOpsRepo string,
	pullRequest *github.PullRequest,
	searchFor string,
) (bool, error) {
	comments, r, err := g.gitClient.Issues.ListComments(
		context.Background(),
		gitHubUser,
		gitOpsRepo,
		*pullRequest.Number,
		&github.IssueListCommentsOptions{},
	)
	if err != nil {
		return false, fmt.Errorf("error listing comments for pull request %d: %w", *pullRequest.Number, err)
	}

	if r.StatusCode != http.StatusOK {
		return false, fmt.Errorf("error retrieving comments for pull request %d, http code is: %d", *pullRequest.Number, r.StatusCode)
	}

	for _, v := range comments {
		if strings.Contains(*v.Body, searchFor) {
			return true, nil
		}
	}

	return false, nil
}

func (g Session) RetrySearchPullRequestComment(
	gitHubUser string,
	gitOpsRepo string,
	pullRequest *github.PullRequest,
	searchFor string,
	logMessage string,
) (bool, error) {
	for i := 0; i < 30; i++ {
		ok, err := g.SearchWordInPullRequestComment(gitHubUser, gitOpsRepo, pullRequest, searchFor)
		if err != nil || !ok {
			log.Info().Msg(logMessage)
			time.Sleep(10 * time.Second)
			continue
		}
		return true, nil
	}
	return false, fmt.Errorf("failed to find the search term %q in comments for pull request %d after retries", searchFor, *pullRequest.Number)
}

// GetRepo - Always returns a status code for whether a repository exists or not
func (g Session) CheckRepoExists(owner, name string) int {
	_, response, _ := g.gitClient.Repositories.Get(g.context, owner, name)
	return response.StatusCode
}

// GetRepo - Always returns a status code for whether a team exists or not
func (g Session) CheckTeamExists(owner, name string) int {
	_, response, _ := g.gitClient.Teams.GetTeamBySlug(g.context, owner, name)
	return response.StatusCode
}

// DeleteRepositoryWebhook
func (g Session) DeleteRepositoryWebhook(owner, repository, url string) error {
	webhooks, err := g.ListRepoWebhooks(owner, repository)
	if err != nil {
		return fmt.Errorf("error listing webhooks for repo %s/%s: %w", owner, repository, err)
	}

	var hookID int64
	for _, hook := range webhooks {
		if url == hook.Config["url"] {
			hookID = hook.GetID()
		}
	}
	if hookID != 0 {
		_, err := g.gitClient.Repositories.DeleteHook(g.context, owner, repository, hookID)
		if err != nil {
			return fmt.Errorf("error deleting hook from repo %s/%s with URL %s: %w", owner, repository, url, err)
		}
		log.Info().Msgf("deleted hook %s/%s/%s", owner, repository, url)
		return nil
	}

	return fmt.Errorf("hook %s/%s/%s not found", owner, repository, url)
}

// ListRepoWebhooks returns all webhooks for a repository
func (g Session) ListRepoWebhooks(owner, repo string) ([]*github.Hook, error) {
	container := make([]*github.Hook, 0)
	for nextPage := 1; nextPage > 0; {
		hooks, resp, err := g.gitClient.Repositories.ListHooks(g.context, owner, repo, &github.ListOptions{
			Page:    nextPage,
			PerPage: 10,
		})
		if err != nil {
			return nil, fmt.Errorf("error listing hooks for repo %s/%s: %w", owner, repo, err)
		}
		container = append(container, hooks...)
		nextPage = resp.NextPage
	}
	return container, nil
}
