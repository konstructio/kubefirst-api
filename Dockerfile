FROM golang:alpine AS builder

ENV GO111MODULE=on \
  CGO_ENABLED=0 \
  GOOS=linux \
  GOARCH=amd64

WORKDIR /build

# Download go dependencies
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy into the container
COPY . .

# Build the application
RUN go build -o kubefirst-api .

# Build final image using nothing but the binary
FROM alpine:3.18.2

COPY --from=builder /build/kubefirst-api /
COPY --from=builder /build/docs /docs

RUN apk update && \
  apk add git openssh 

RUN mkdir -p /root/.ssh
RUN ssh-keyscan github.com >> /root/.ssh/known_hosts
RUN ssh-keyscan gitlab.com >> /root/.ssh/known_hosts

EXPOSE 8081

# Command to run
ENTRYPOINT ["/kubefirst-api"]
