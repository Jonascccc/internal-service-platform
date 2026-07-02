# Service Registry API

A minimal in-memory Service Registry API built with Go.

This project simulates a small internal platform service that allows engineering teams to register backend services and list all registered services. It is designed as a learning project for backend/platform engineering fundamentals, including HTTP routing, JSON APIs, validation, in-memory storage, concurrency safety, and deterministic API responses.

## Features

* Health check endpoint
* Version endpoint
* Register a new internal service
* List all registered services
* Validate required service metadata
* Default service environment to `dev`
* Prevent duplicate service IDs
* Return services sorted by ID
* Thread-safe in-memory storage using `sync.Mutex`
* Consistent JSON response format
* Multi-stage Docker build
* Minimal Alpine runtime image

## Tech Stack

* Go
* Standard `net/http` package
* In-memory map storage
* JSON encoding/decoding
* Mutex-based concurrency protection

## API Endpoints

| Method | Endpoint    | Description                  |
| ------ | ----------- | ---------------------------- |
| GET    | `/health`   | Check if the API is running  |
| GET    | `/version`  | Return API version           |
| POST   | `/services` | Register a new service       |
| GET    | `/services` | List all registered services |

## Service Model

Each service has the following fields:

```json
{
  "id": "payment-service",
  "name": "Payment Service",
  "owner": "payments-team",
  "environment": "prod"
}
```

### Field Rules

| Field         | Required | Description                                                 |
| ------------- | -------- | ----------------------------------------------------------- |
| `id`          | Yes      | Unique service identifier                                   |
| `name`        | Yes      | Human-readable service name                                 |
| `owner`       | Yes      | Owning team name                                            |
| `environment` | No       | Defaults to `dev`; allowed values: `dev`, `staging`, `prod` |

## Run Locally

### 1. Open the project folder

```bash
cd path/to/service-registry-api
```

### 2. Verify Go

```bash
go version
```

### 3. Start the server

```bash
go run .
```

The API is available at `http://localhost:8080`.

## Run with Docker

### 1. Build the image

```bash
docker build -t service-registry-api .
```

### 2. Run the container

```bash
docker run --rm -p 8080:8080 service-registry-api
```

### 3. Verify the API

```bash
curl http://localhost:8080/health
curl http://localhost:8080/version
```

Press `Ctrl+C` to stop the container. The `--rm` option automatically removes the stopped container.

## Example Requests

### Health Check

```bash
curl http://localhost:8080/health
```

Example response:

```json
{
  "status": "ok",
  "service": "service-registry-api"
}
```

### Version

```bash
curl http://localhost:8080/version
```

Example response:

```json
{
  "service": "service-registry-api",
  "version": "v0.0.1"
}
```

### Register a Service

```bash
curl -X POST http://localhost:8080/services \
  -H "Content-Type: application/json" \
  -d '{
    "id": "payment-service",
    "name": "Payment Service",
    "owner": "payments-team",
    "environment": "prod"
  }'
```

Example response:

```json
{
  "id": "payment-service",
  "name": "Payment Service",
  "owner": "payments-team",
  "environment": "prod"
}
```

### Register a Service Without Environment

If `environment` is not provided, it defaults to `dev`.

```bash
curl -X POST http://localhost:8080/services \
  -H "Content-Type: application/json" \
  -d '{
    "id": "user-service",
    "name": "User Service",
    "owner": "user-team"
  }'
```

Example response:

```json
{
  "id": "user-service",
  "name": "User Service",
  "owner": "user-team",
  "environment": "dev"
}
```

### List Services

```bash
curl http://localhost:8080/services
```

Example response:

```json
[
  {
    "id": "payment-service",
    "name": "Payment Service",
    "owner": "payments-team",
    "environment": "prod"
  },
  {
    "id": "user-service",
    "name": "User Service",
    "owner": "user-team",
    "environment": "dev"
  }
]
```

Services are returned sorted by `id`.

## Error Response Format

All errors use a consistent JSON format:

```json
{
  "error": {
    "code": "invalid_request",
    "message": "id is required"
  }
}
```

## Example Error Cases

### Missing Required Field

Request:

```json
{
  "name": "Payment Service",
  "owner": "payments-team"
}
```

Response:

```json
{
  "error": {
    "code": "invalid_request",
    "message": "id is required"
  }
}
```

### Invalid Environment

Allowed environments are:

```text
dev
staging
prod
```

Invalid request:

```json
{
  "id": "bad-service",
  "name": "Bad Service",
  "owner": "test-team",
  "environment": "qa"
}
```

Response:

```json
{
  "error": {
    "code": "invalid_request",
    "message": "environment must be one of: dev, staging, prod"
  }
}
```

### Duplicate Service ID

If a service ID already exists, the API returns:

```json
{
  "error": {
    "code": "conflict",
    "message": "service id already exists"
  }
}
```

## Current Limitations

This is a minimal v1 implementation.

Current limitations:

* Data is stored in memory only
* Data is lost when the server restarts
* No database persistence
* No authentication or authorization
* No update or delete endpoints
* No pagination or filtering
* No automated tests yet
* No CI pipeline yet

## Future Improvements

Possible next steps:

* Add unit tests
* Add structured logging
* Add request logging middleware
* Add update/delete endpoints
* Add persistent storage with PostgreSQL
* Add service ownership metadata such as Slack channel or repository URL
* Add filtering by environment
* Add OpenAPI documentation

## Learning Goals

This project is intended to practice:

* Building HTTP APIs in Go
* Designing simple API contracts
* Handling JSON request/response bodies
* Validating API input
* Using maps for in-memory storage
* Protecting shared state with mutexes
* Returning deterministic API responses
* Writing clean project documentation


## Service Metadata Contract

The platform API stores service ownership and operational metadata used by deployment, observability, and support workflows.

Required fields:
- `id`
- `name`
- `owner`

Defaults:
- `environment`: `dev`
- `tier`: `tier-3`
- `language`: `go`

Validation guardrails:
- `environment` must be one of `dev`, `staging`, `prod`
- `tier` must be one of `tier-1`, `tier-2`, `tier-3`
- `language` must be one of `go`, `python`, `node`, `java`
- `repo_url`, if provided, must start with `http://` or `https://`
- `slack_channel`, if provided, must start with `#`

Server-generated fields:
- `created_at`
- `updated_at`

These defaults and validation rules model platform guardrails: teams can self-serve service registration, while the platform keeps metadata consistent enough for automation, dashboards, alert routing, and release workflows.