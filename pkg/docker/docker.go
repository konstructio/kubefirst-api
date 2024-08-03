package docker

import internal "github.com/kubefirst/kubefirst-api/internal/docker"

type ClientWrapper = internal.DockerClientWrapper

var NewDockerClient = internal.NewDockerClient
