<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="images/kubefirst-light.svg" alt="Kubefirst Logo">
    <img alt="" src="images/kubefirst.svg">
  </picture>
</p>
<p align="center">
  Gitops Infrastructure & Application Delivery Platform
</p>

<p align="center">
  <a href="https://docs.kubefirst.io/">Install</a>&nbsp;|&nbsp;
  <a href="https://twitter.com/kubefirst">Twitter</a>&nbsp;|&nbsp;
  <a href="https://www.linkedin.com/company/kubefirst">LinkedIn</a>&nbsp;|&nbsp;
  <a href="https://join.slack.com/t/kubefirst/shared_invite/zt-r0r9cfts-OVnH0ooELDLm9n9p2aU7fw">Slack</a>&nbsp;|&nbsp;
  <a href="https://kubeshop.io/blog-projects/kubefirst">Blog</a>
</p>

# kubefirst-api

Kubefirst API runtime implementation.

- [kubefirst-api](#kubefirst-api)
  - [Running Locally](#running-locally)
    - [Build the Binary](#build-the-binary)
    - [Leverage `air` for Live Reloading Locally](#leverage-air-for-live-reloading-locally)
  - [Prerequisites](#prerequisites)
    - [Environment Variables](#environment-variables)
  - [Provider Support](#provider-support)
  - [Creating a Cluster](#creating-a-cluster)
      - [Kubernetes Secret](#kubernetes-secret)
      - [API Call Parameters](#api-call-parameters)
    - [AWS](#aws)
    - [Civo](#civo)
    - [Digital Ocean](#digital-ocean)
    - [Vultr](#vultr)
    - [Deleting a Cluster](#deleting-a-cluster)
  - [Authentication](#authentication)
    - [Creating a User](#creating-a-user)
    - [Authenticating](#authenticating)
  - [Swagger UI](#swagger-ui)
  - [Updating Swagger Docs](#updating-swagger-docs)

## Running Locally

The API is available at `http://localhost:8081/api/v1` while running.

### Build the Binary

The API can be run locally for testing. It can be run by using `make build` and then calling the binary in the `bin/` directory or by using `go run .`.
   
### Leverage `air` for Live Reloading Locally

**Prerequsite** - Install [air](https://github.com/cosmtrek/air).

```go
go install github.com/cosmtrek/air@latest
```

Run `air` from the root of the repository. This will watch go files and live rebuild a local running instance of `kubefirst-api`.   

## Prerequisites

The API uses MongoDB for storing records.

The best option is to use [MongoDB Atlas](https://www.mongodb.com/atlas). This is the recommended approach.

For local development, you can install [MongoDB Community Edition](https://www.mongodb.com/docs/manual/tutorial/install-mongodb-on-os-x/) - this is not production-quality.

It is also recommended to install [MongoDB Compass](https://www.mongodb.com/try/download/atlascli).

The host:port for MongoDB should be supplied as the environment variable `MONGODB_HOST`. When testing locally, use `localhost:27017`.

### Environment Variables

Some variables are required, others are optional depending on deployment type.

| Variable            | Description                                                                                                                                      | Required       |
| ------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------ | -------------- |
| `MONGODB_HOST_TYPE` | Can be either `atlas` or `local`.                                                                                                                | Yes            |
| `MONGODB_HOST`      | The host to connect to. For Atlas, use only the portion of the string not containing username or password. For all other types, append the port. | Yes            |
| `MONGODB_USERNAME`  | Required when using Atlas.                                                                                                                       | If using Atlas |
| `MONGODB_PASSWORD`  | Required when using Atlas.                                                                                                                       | If using Atlas |
| `IN_CLUSTER`        | Specify whether or not the API is running inside a Kubernetes cluster. By default, this is assumed `false`.                                      | No             |
| `CLUSTER_ID`        | The ID of the cluster running API.                                                                                                               | Yes            |
| `CLUSTER_TYPE`      | Cluster type.                                                                                                                                    | Yes            |
| `INSTALL_METHOD`    | Description of the method through which the API was deployed. Example: `helm`                                                                    | Yes            |
| `K1_ACCESS_TOKEN`    | Access token in authorization header to prevent unsolicited in-cluster access | Yes            |

To run locally: 

```bash
export MONGODB_USERNAME=
export MONGODB_PASSWORD=
export MONGODB_HOST_TYPE=atlas / local
export MONGODB_HOST=
export CLUSTER_TYPE=
export CLUSTER_ID=
export INSTALL_METHOD=
export K1_ACCESS_TOKEN=localexample
```

## Provider Support

The following providers are available for use with the API.

| Provider      | Status | Supported Operations | Supported Git Providers |
| ------------- | ------ | -------------------- | ----------------------- |
| AWS           | Beta   | Create, Delete       | GitHub, GitLab          |
| Civo          | Beta   | Create, Delete       | GitHub, GitLab          |
| Digital Ocean | Beta   | Create, Delete       | GitHub, GitLab          |
| Vultr         | Beta   | Create, Delete       | GitHub, GitLab          |

## Creating a Cluster

In order to create a cluster, authentication credentials must be provided in one of two ways:

#### Kubernetes Secret

If a Kubernetes `Secret` called `kubefirst-auth` exists, the API will attempt to read cloud provider credentials from this `Secret`.

The `Secret` format is expected to have the following keys based on which clouds you are deploying to:

```
aws-access-key-id
aws-secret-access-key
aws-session-token

civo-token

do-token
do-spaces-key
do-spaces-token

vultr-api-key
```

Each key must have the appropriate value set in order for the API to successfully create a cluster.

#### API Call Parameters

If there is no `Secret`, the API will then attempt to read from the parameters passed in via the API call.

This would require the following parameters added to the API call depending on which cloud provider is being used:

```json
{
	"aws_auth": {
		"access_key_id": "foo",
		"secret_access_key": "bar",
		"session_token": "baz"
	}
}
```

```json
{
	"civo_auth": {
		"token": "my-civo-token"
	}
}
```

```json
{
	"do_auth": {
		"token": "my-do-token",
		"spaces_key": "foo",
		"spaces_secret": "bar"
	}
}
```

```json
{
	"vultr_auth": {
		"token": "my-vultr-api-key"
	}
}
```

If either of these options is missing, the API will return an error.

### AWS

You must use the authentication strategy above to set credentials before running.

```bash
curl -X POST http://localhost:8081/api/v1/cluster/kf-api-scott-test -H "Content-Type: application/json" -d '{"admin_email": "scott@kubeshop.io", "cloud_provider": "aws", "cloud_region": "us-east-1", "domain_name": "kubefirst.cloud", "git_owner": "kubefirst-cloud", "git_provider": "github", "git_token": "ghp_...", "type": "mgmt"}'
```

### Civo

You must use the authentication strategy above to set credentials before running.

```bash
curl -X POST http://localhost:8081/api/v1/cluster/my-cool-cluster -H "Content-Type: application/json" -d '{"admin_email": "scott@kubeshop.io", "cloud_provider": "civo", "cloud_region": "nyc1", "domain_name": "your-dns.io", "git_owner": "your-dns-io", "git_provider": "github", "git_token": "ghp_...", "type": "mgmt"}'
```

### Digital Ocean

Kubefirst does not create a Digital Ocean space for you. You must create one ahead of time and provide the key and secret when creating a Digital Ocean cluster. The space acts as an S3-compatible storage bucket for Terraform state and other cluster operations.

You must use the authentication strategy above to set credentials before running.

```bash
curl -X POST http://localhost:8081/api/v1/cluster/my-cool-cluster -H "Content-Type: application/json" -d '{"admin_email": "scott@kubeshop.io", "cloud_provider": "digitalocean", "cloud_region": "nyc3", "domain_name": "kubefunk.de", "git_owner": "kubefunk-de", "git_provider": "github", "git_token": "ghp_...", "type": "mgmt"}'
```

### Vultr

You must use the authentication strategy above to set credentials before running.

```bash
curl -X POST http://localhost:8081/api/v1/cluster/my-cool-cluster -H "Content-Type: application/json" -d '{"admin_email": "scott@kubeshop.io", "cloud_provider": "vultr", "cloud_region": "ewr", "domain_name": "kubesecond.com", "git_owner": "your-dns-io", "git_provider": "github", "git_token": "ghp_...", "type": "mgmt"}'
```

### Deleting a Cluster

```bash
curl -X DELETE http://localhost:8081/api/v1/cluster/my-cool-cluster
```

## Authentication

The API expects an `Authorization` header with the content `Bearer <API key>`. For example:

```bash
❯ curl -X GET "localhost:8081/api/v1/cluster" \
     -H "Authorization: Bearer my-api-key" \
     -H "Content-Type:application/json"
```

The provided bearer token is validated against an auto-generated key that gets stored in secret `kubefirst-initial-secrets` provided by this chart. It's then consumed by this same chart's deployment as an environment variable `K1_ACCESS_TOKEN` for the comparison. The console application will have access to this same namespaced secret and can leverage the bearer token to authorize calls to the `kubefirst-api` and `kubefirst-api-ee` services.

## Swagger UI

When the app is running, the UI is available via http://localhost:8081/swagger/index.html.

## Updating Swagger Docs

Swagger UI is generated using [gin-swagger](https://github.com/swaggo/gin-swagger). Tagged routes will generate documentation.

Any time godoc defs for routes are changed, `swag init` should be run.

In order to generate docs:

```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

```bash
make updateswagger
```

