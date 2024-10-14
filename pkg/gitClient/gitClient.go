package gitClient //nolint:revive,stylecheck // allowing temporarily for better code organization

import "github.com/konstructio/kubefirst-api/internal/gitClient"

var (
	Commit           = gitClient.Commit
	ClonePrivateRepo = gitClient.ClonePrivateRepo
	Clone            = gitClient.Clone
)
