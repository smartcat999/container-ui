# Registry Agent

Registry Agent is a proxy service for Docker Registry, allowing you to cache and forward Docker image requests to remote registries.

## Features

- Proxy requests to remote Docker registries
- Support for private registry authentication
- OpenTelemetry integration for tracing
- Configurable through environment variables

## Prerequisites

- Go 1.19 or higher
- Docker (optional)

## Configuration

The service can be configured using the following environment variables:

| Variable | Description | Default Value |
|----------|-------------|---------------|
| REGISTRY_PROXY_REMOTE_URL | Remote registry URL | https://registry-1.docker.io |
| REGISTRY_PROXY_USERNAME | Username for remote registry | "" |
| REGISTRY_PROXY_PASSWORD | Password for remote registry | "" |
| OTEL_TRACES_EXPORTER | OpenTelemetry traces exporter (otlp/console/none) | console |

## Local Setup

1. Clone the repository
```bash
git clone https://github.com/smartcat999/registry-agent.git
cd registry-agent
```
2. Build the service
```bash
go build -o registry-agent cmd/agent/main.go
```
3. Run the service
```bash
./registry-agent
```
The service will start on port 5000 by default.


## Usage
1. Pull through the proxy:
```bash
# update docker daemon.json
{
  "insecure-registries" : [
    "127.0.0.1:5000"
  ]
}
```

```bash
docker pull 127.0.0.1:5000/repo/myimage:latest
```