<!-- markdownlint-disable MD033 MD041 MD024 -->
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
    - [Use with the CLI](#use-with-the-cli)
  - [Prerequisites for local development](#prerequisites-for-local-development)
    - [Environment Variables](#environment-variables)
  - [local environment variables](#local-environment-variables)
  - [Provider Support](#provider-support)
  - [Creating a Cluster](#creating-a-cluster)
    - [Authentication Credentials](#authentication-credentials)
      - [Kubernetes Secret](#kubernetes-secret)
      - [API Call Parameters](#api-call-parameters)
        - [Akamai](#akamai)
        - [AWS](#aws)
        - [Civo](#civo)
        - [DigitalOcean](#digitalocean)
        - [Google Cloud](#google-cloud)
        - [Vultr](#vultr)
      - [API Call](#api-call)
        - [Akamai](#akamai-1)
        - [AWS](#aws-1)
        - [Civo](#civo-1)
        - [DigitalOcean](#digitalocean-1)
        - [Vultr](#vultr-1)
    - [Deleting a Cluster](#deleting-a-cluster)
  - [Authentication](#authentication)
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

### Use with the CLI

If you want to use your local API version with the CLI, you need to do two things:

1. Set the `K1_LOCAL_DEBUG` environment variable to `true` with `export K1_LOCAL_DEBUG=true`.
2. Run the [console application](https://github.com/kubefirst/console) locally also by following [those instructions](https://github.com/kubefirst/console#setup-instructions).

Be sure that you do not change the default port for the console (3000), and the default one for the API (8081) for this to work.

## Prerequisites for local development

For local development, we need to have a k3d cluster where the kubefirst api can store information in secrets

- Download [k3d](https://k3d.io/)
- Create a cluster ```k3d cluster create dev```
- Dowload the kubeconfig ```k3d kubeconfig write dev```
- Update the `K1_LOCAL_KUBECONFIG_PATH` environment variable with the kubeconfig location
- Enjoy!

### Environment Variables

Some variables are required, others are optional depending on deployment type.

| Variable                    | Description                                                                                                                                      | Required                       |
| -------------------         | ------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------ |
| `IN_CLUSTER`                | Specify whether or not the API is running inside a Kubernetes cluster. By default, this is assumed `false`.                                      | No                             |
| `CLUSTER_ID`                | The ID of the cluster running API.                                                                                                               | Yes                            |
| `CLUSTER_TYPE`              | Cluster type.                                                                                                                                    | Yes                            |
| `INSTALL_METHOD`            | Description of the method through which the API was deployed. Example: `helm`                                                                    | Yes                            |
| `K1_ACCESS_TOKEN`           | Access token in authorization header to prevent unsolicited in-cluster access                                                                    | Yes                            |
| `K1_LOCAL_DEBUG`            | Identifies the api execution as local debug mode                                                                                                 | Yes                             |
| `K1_LOCAL_KUBECONFIG_PATH`  | kubeconfig path location for k3d local cluster                                                                                                   | Yes                            |

## local environment variables

see [this .env example](./.env.example) for the necessary values

Be sure to set `IS_CLUSTER_ZERO` to `true` if you want to run the API without having console running.

## Provider Support

The following providers are available for use with the API.

| Provider      | Status | Supported Operations | Supported Git Providers |
| ------------- | ------ | -------------------- | ----------------------- |
| AWS           | Beta   | Create, Delete       | GitHub, GitLab          |
| Civo          | Beta   | Create, Delete       | GitHub, GitLab          |
| DigitalOcean  | Beta   | Create, Delete       | GitHub, GitLab          |
| Vultr         | Beta   | Create, Delete       | GitHub, GitLab          |

## Creating a Cluster

### Authentication Credentials

In order to create a cluster, authentication credentials must be provided in one of two ways:

#### Kubernetes Secret

If a Kubernetes `Secret` called `kubefirst-auth` exists, the API will attempt to read cloud provider credentials from this `Secret`.

The `Secret` format is expected to have the following keys based on which clouds you are deploying to:

```text
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

##### Akamai

```json
{
  "akamai_auth": {
    "token": "my-akamai-token"
  }
}
```

##### AWS

```json
{
  "aws_auth": {
    "access_key_id": "foo",
    "secret_access_key": "bar",
    "session_token": "baz"
  }
}
```

##### Civo

```json
{
  "civo_auth": {
    "token": "my-civo-token"
  }
}
```

##### DigitalOcean

```json
{
  "do_auth": {
    "token": "my-do-token",
    "spaces_key": "foo",
    "spaces_secret": "bar"
  }
}
```

##### Google Cloud

```json
{
  "gcp_auth": {
    "key_file": "my-google-credentials-json-keyfile-stringified-no-newline-characters",
    "project_id": "google cloud project id"
  }
}
```

##### Vultr

```json
{
  "vultr_auth": {
    "token": "my-vultr-api-key"
  }
}
```

If either of these options is missing, the API will return an error.

#### API Call

##### Akamai

You must use the authentication strategy above to set credentials before running.

```shell
curl -X POST http://localhost:8081/api/v1/cluster/my-cool-cluster -H "Content-Type: application/json" -d '{"admin_email": "your@email.com", "cloud_provider": "akamai", "domain_name": "kubefirst.cloud", "git_owner": "kubefirst-cloud", "git_provider": "github", "git_token": "ghp_...", "type": "mgmt"}'
```

##### AWS

You must use the authentication strategy above to set credentials before running.

```shell
curl -X POST http://localhost:8081/api/v1/cluster/my-cool-cluster -H "Content-Type: application/json" -d '{"admin_email": "your@email.com", "cloud_provider": "aws", "cloud_region": "us-east-1", "domain_name": "kubefirst.cloud", "git_owner": "kubefirst-cloud", "git_provider": "github", "git_token": "ghp_...", "type": "mgmt"}'
```

##### Civo

You must use the authentication strategy above to set credentials before running.

```shell
curl -X POST http://localhost:8081/api/v1/cluster/my-cool-cluster -H "Content-Type: application/json" -d '{"admin_email": "your@email.com", "cloud_provider": "civo", "cloud_region": "nyc1", "domain_name": "your-dns.io", "git_owner": "your-dns-io", "git_provider": "github", "git_token": "ghp_...", "type": "mgmt"}'
```

##### DigitalOcean

Kubefirst does not create a DigitalOcean space for you. You must create one ahead of time and provide the key and secret when creating a DigitalOcean cluster. The space acts as an S3-compatible storage bucket for Terraform state and other cluster operations.

You must use the authentication strategy above to set credentials before running.

```shell
curl -X POST http://localhost:8081/api/v1/cluster/my-cool-cluster -H "Content-Type: application/json" -d '{"admin_email": "your@email.com", "cloud_provider": "digitalocean", "cloud_region": "nyc3", "domain_name": "kubefunk.de", "git_owner": "kubefunk-de", "git_provider": "github", "git_token": "ghp_...", "type": "mgmt"}'
```

##### Vultr

You must use the authentication strategy above to set credentials before running.

```shell
curl -X POST http://localhost:8081/api/v1/cluster/my-cool-cluster -H "Content-Type: application/json" -d '{"admin_email": "your@email.com", "cloud_provider": "vultr", "cloud_region": "ewr", "domain_name": "kubesecond.com", "git_owner": "your-dns-io", "git_provider": "github", "git_token": "ghp_...", "type": "mgmt"}'
```

### Deleting a Cluster

```shell
curl -X DELETE http://localhost:8081/api/v1/cluster/my-cool-cluster
```

## Authentication

The API expects an `Authorization` header with the content `Bearer <API key>`. For example:

```shell
❯ curl -X GET "localhost:8081/api/v1/cluster" \
     -H "Authorization: Bearer my-api-key" \
     -H "Content-Type:application/json"
```

The provided bearer token is validated against an auto-generated key that gets stored in secret `kubefirst-initial-secrets` provided by this chart. It's then consumed by this same chart's deployment as an environment variable `K1_ACCESS_TOKEN` for the comparison. The console application will have access to this same namespaced secret and can leverage the bearer token to authorize calls to the `kubefirst-api` and `kubefirst-api-ee` services.

## Swagger UI

When the app is running, the UI is available via <http://localhost:8081/swagger/index.html>.

## Updating Swagger Docs

Swagger UI is generated using [gin-swagger](https://github.com/swaggo/gin-swagger). Tagged routes will generate documentation.

Any time godoc defs for routes are changed, `swag init` should be run.

In order to generate docs:

```shell
go install github.com/swaggo/swag/cmd/swag@latest
```

```shell
make updateswagger
```
