# kubefirst-api

Kubefirst API that serves console frontend.

## Updating Docs

Swagger UI is generated using [gin-swagger](https://github.com/swaggo/gin-swagger).

Any time godoc defs for routes are changed, `swag init` should be run.

In order to generate docs:

```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

## Swagger UI

When the app is running, the UI is available via http://:8080/swagger/index.html.

## Running Locally

The API can be run locally for testing. The api is available at `:8080/api/v1`:

```bash
CIVO_TOKEN=mytoken CIVO_REGION=nyc1 AWS_REGION=us-east-1 AWS_PROFILE=myprofile go run main.go
```
