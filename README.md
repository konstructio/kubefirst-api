# kubefirst-api

Kubefirst API runtime implementation.

## Updating Docs

Swagger UI is generated using [gin-swagger](https://github.com/swaggo/gin-swagger). Tagged routes will generate documentation.

Any time godoc defs for routes are changed, `swag init` should be run.

In order to generate docs:

```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

```bash
make updateswagger
```

## Swagger UI

When the app is running, the UI is available via http://:8081/swagger/index.html.

## Running Locally

The API can be run locally for testing. The api is available at `:8081/api/v1`.

## Creating a Cluster

This is in active development. At this time, only `k3d` is supported:

```bash
curl -X POST http://localhost:8081/api/v1/cluster/my-cluster -H "Content-Type: application/json" -d '{"admin_email": "scott@kubeshop.io", "cloud_provider": "k3d", "cloud_region": "us-east-1", "domain_name": "your-dns.io", "git_owner": "your-dns-io", "git_provider": "github", "git_token": "ghp_...", "type": "mgmt"}'
```
