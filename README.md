<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="images/kubefirst-light.svg" alt="Kubefirst Logo">
    <img alt="" src="images/kubefirst.svg">
  </picture>
</p>
<p align="center">
  GitOps Infrastructure & Application Delivery Platform
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
    - [build the binary](#build-the-binary)
    - [or](#or)
    - [leverage `air` for live reloading locally](#leverage-air-for-live-reloading-locally)
  - [Prerequisites](#prerequisites)
  - [Provider Support](#provider-support)
  - [Creating a Cluster](#creating-a-cluster)
    - [AWS](#aws)
    - [Civo](#civo)
    - [Digital Ocean](#digital-ocean)
    - [Vultr](#vultr)
    - [Deleting a Cluster](#deleting-a-cluster)
  - [Swagger UI](#swagger-ui)
  - [Updating Swagger Docs](#updating-swagger-docs)

## Running Locally

### build the binary
The API can be run locally for testing. It can be run by using `make build` and then calling the binary in the `bin/` directory or by using `go run .`.
   
### or   
   
### leverage `air` for live reloading locally
**Prerequsite** - install air
[air](https://github.com/cosmtrek/air) is a lightweight golang utility   
```golang
go install github.com/cosmtrek/air@latest
```
run `air` from the root of the repository. this will watch go files and live rebuild a local running instance of `kubefirst-api`.   

The API is available at `:8081/api/v1` while running.

## Prerequisites

The API uses MongoDB for storing records.

For local development, it's recommended to install [MongoDB Community Edition](https://www.mongodb.com/docs/manual/tutorial/install-mongodb-on-os-x/).

It is also recommended to install [MongoDB Compass](https://www.mongodb.com/try/download/atlascli).

The host:port for MongoDB should be supplied as the environment variable `MONGODB_HOST`. When testing locally, use `localhost:27017`.

## Provider Support

The following providers are available for use with the API.

| Provider      | Status | Supported Operations | Supported Git Providers |
| ------------- | ------ | -------------------- | ----------------------- |
| AWS           | Beta   | Create, Delete       | GitHub, GitLab          |
| Civo          | Beta   | Create, Delete       | GitHub, GitLab          |
| Digital Ocean | Beta   | Create, Delete       | GitHub, GitLab          |
| K3d           | Beta   | Create, Delete       | GitHub, GitLab          |
| Vultr         | Beta   | Create, Delete       | GitHub, GitLab          |

## Creating a Cluster

*Note:* This is under active development. As such, there are limitations. At this time, we still depend on environment variables for cloud provider authentication. Git authentication is passed in the API call.

In a future update, cloud provider authentication will generate an in-cluster Secret which the API will leverage.

GitHub has been tested and works. GitLab has not been tested yet so success may be spotty.

When starting the API, you have to have certain OS environment variables set in order for it to work. In the future, this won't be a requirement.

### AWS

You must have the `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, and `AWS_SESSION_TOKEN` environment variables set.

```bash
curl -X POST http://localhost:8081/api/v1/cluster/kf-api-scott-test -H "Content-Type: application/json" -d '{"admin_email": "scott@kubeshop.io", "cloud_provider": "aws", "cloud_region": "us-east-1", "domain_name": "kubefirst.cloud", "git_owner": "kubefirst-cloud", "git_provider": "github", "git_token": "ghp_...", "type": "mgmt"}'
```

### Civo

You must have the `CIVO_TOKEN` environment variable set containing your API key.

```bash
curl -X POST http://localhost:8081/api/v1/cluster/my-cool-cluster -H "Content-Type: application/json" -d '{"admin_email": "scott@kubeshop.io", "cloud_provider": "civo", "cloud_region": "nyc1", "domain_name": "your-dns.io", "git_owner": "your-dns-io", "git_provider": "github", "git_token": "ghp_...", "type": "mgmt"}'
```

### Digital Ocean

You must have the `DO_TOKEN`, `DO_SPACES_KEY`, and `DO_SPACES_SECRET` environment variables set containing your API key, spaces key, and spaces secret respectively. Kubefirst does not create a Digital Ocean space for you. You must create one ahead of time and provide the key and secret when creating a Digital Ocean cluster. The space acts as an S3-compatible storage bucket for Terraform state and other cluster operations.

```bash
curl -X POST http://localhost:8081/api/v1/cluster/my-cool-cluster -H "Content-Type: application/json" -d '{"admin_email": "scott@kubeshop.io", "cloud_provider": "digitalocean", "cloud_region": "nyc3", "domain_name": "kubefunk.de", "git_owner": "kubefunk-de", "git_provider": "github", "git_token": "ghp_...", "type": "mgmt"}'
```

### Vultr

You must have the `VULTR_API_KEY` environment variable set containing your API key.

```bash
curl -X POST http://localhost:8081/api/v1/cluster/my-cool-cluster -H "Content-Type: application/json" -d '{"admin_email": "scott@kubeshop.io", "cloud_provider": "vultr", "cloud_region": "ewr", "domain_name": "kubesecond.com", "git_owner": "your-dns-io", "git_provider": "github", "git_token": "ghp_...", "type": "mgmt"}'
```

### Deleting a Cluster

```bash
curl -X DELETE http://localhost:8081/api/v1/cluster/my-cool-cluster
```

## Swagger UI

When the app is running, the UI is available via http://:8081/swagger/index.html.

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
