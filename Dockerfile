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
FROM alpine:3.17.2

COPY --from=builder /build/kubefirst-api /
COPY --from=builder /build/docs /docs

EXPOSE 8081

# Command to run
ENTRYPOINT ["/kubefirst-api"]