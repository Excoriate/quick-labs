# Multi-Service Docker Microservices Lab

## Overview

This lab demonstrates a robust microservices architecture using Docker Compose, showcasing best practices for:

-   Inter-service communication
-   Authentication
-   Health checking
-   Logging
-   Resource management
-   Service discovery

## Architecture

The lab consists of two Go-based microservices:

### Service A

-   Runs on port 8080
-   Provides a `/greet` endpoint with authentication
-   Implements structured logging
-   Uses JSON-based request/response
-   Requires an authentication key

### Service B

-   Runs on port 8081
-   Calls Service A's `/greet` endpoint
-   Combines responses from both services
-   Implements inter-service communication
-   Handles potential service failures gracefully

## Key Features

-   🔒 Authentication Middleware
-   🌐 Inter-Service Communication
-   📊 Structured Logging
-   🩺 Health Checks
-   🐳 Docker Compose Configuration
-   🚦 Resource Constraints
-   🔍 Error Handling

## Prerequisites

-   Docker
-   Docker Compose (v2+)
-   Make (optional, but recommended)
-   Go 1.21.6+

## Quick Start

### Running the Services

```bash
# Build and start services
make build
make run-detached

# Or using Docker Compose directly
docker compose build
docker compose up -d
```

### Testing Services

```bash
# Run automated tests
make test

# Test Service A directly
curl -H "X-Auth-Key: service-a-secret-key" http://localhost:8080/greet

# Test Service B (which calls Service A)
curl http://localhost:8081/process
```

### Stopping Services

```bash
# Stop and remove containers
make stop
# Or
docker compose down
```

## Local Development

```bash
# Run services locally
make run-local

# Run local tests
make test-local
```

## Logging and Monitoring

-   Services use structured JSON logging
-   Health check endpoints at `/health`
-   Configurable log levels
-   Docker Compose logging configuration

## Security Considerations

-   Authentication via custom headers
-   Masked authentication key logging
-   Default keys for development only

## Performance Characteristics

-   Resource-constrained containers
-   5-second request timeouts
-   Graceful service shutdown
-   Health check mechanisms

## Learning Objectives

1. Microservices design patterns
2. Docker Compose configuration
3. Inter-service communication
4. Go programming best practices
5. Containerization strategies

## Troubleshooting

-   Ensure Docker and Docker Compose are up to date
-   Check container logs: `docker compose logs`
-   Verify network connectivity
-   Confirm authentication keys match

## Contributing

Contributions, issues, and feature requests are welcome!

## License

[Insert appropriate license]
