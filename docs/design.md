# Design: Service Registry API

## 1. Problem

As an organization grows, backend services multiply and ownership becomes
tribal knowledge — scattered across Slack threads and people's heads. A central
registry gives teams a single source of truth for which group owns which
service, what environment it runs in, and who to contact. This replaces
informal lookup with validated, structured metadata.

## 2. Goals

- Central registration of backend services
- Validated required metadata: `id`, `name`, `owner`
- Deterministic responses (sorted by `id`)
- Thread-safe concurrent access
- Consistent JSON error envelope across all endpoints

## 3. Non-goals

- **Authentication** — out of scope for v1; the focus is validating the
  registry data model, not access control.
- **Persistent storage** — in-memory is sufficient to validate the API
  contract; PostgreSQL is planned for a later milestone.
- **Live service monitoring** — this registry stores *metadata about* services,
  it does not health-check or probe running instances. (This distinguishes a
  registry from service discovery.)
- **Update and delete endpoints** — deliberately deferred to the
  onboarding-platform milestone to keep v1 scope tight.

## 4. API Design

| Method | Endpoint    | Status codes  |
| ------ | ----------- | ------------- |
| GET    | `/health`   | 200           |
| GET    | `/version`  | 200           |
| POST   | `/services` | 201, 400, 409 |
| GET    | `/services` | 200           |

**Conventions:**

- Resource-oriented paths (`/services`, not `/createService`)
- `201 Created` returns the full created object — no second fetch needed
- `409 Conflict` for duplicate IDs, `400 Bad Request` for validation failures
- All errors share a uniform `{"error": {"code", "message"}}` envelope, so
  clients can parse failures generically without per-endpoint error logic

## 5. Data Model

| Field         | Required | Notes                                                |
| ------------- | -------- | ---------------------------------------------------- |
| `id`          | Yes      | Unique service identifier                            |
| `name`        | Yes      | Human-readable name                                  |
| `owner`       | Yes      | Owning team name                                     |
| `environment` | No       | Defaults to `dev`; allowed: `dev`, `staging`, `prod` |

This model is intentionally minimal for v1. It will expand to include `team`,
`repo_url`, `slack_channel`, `tier`, `language`, `created_at`, and `updated_at`
in the onboarding-platform milestone.

## 6. Storage Choice

v1 uses an in-memory `map[string]Service`. Zero infrastructure, fast iteration,
debug-friendly — appropriate for validating the API contract before committing
to a persistence layer.

**Tradeoff:** volatile — all data is lost on restart, and storage cannot scale
beyond a single process.

**Migration path:** introduce a `Store` interface as an abstraction seam, then
swap the in-memory implementation for PostgreSQL without changing handler logic.

## 7. Concurrency Design

Go's built-in `map` is not safe for concurrent read/write — simultaneous access
triggers a fatal runtime **panic**. Since Go's HTTP server handles each request
in its own goroutine, two simultaneous requests can race on the map, making a
mutex mandatory.

**Deliberate ordering in `POST /services`:** field validation runs *before*
acquiring the lock, because it only touches the local request struct — no
shared state is involved, so holding the lock during validation would serialize
concurrent requests for no safety benefit. The duplicate-ID check and the insert
run *inside the same lock acquisition*, making them atomic. Separating them into
two lock acquisitions would create a window where two goroutines could both see
"ID not found" and both proceed to insert — a silent duplicate.

**Future optimization:** v1 uses `sync.Mutex`, which fully serializes all access
including concurrent reads. A `sync.RWMutex` would allow multiple readers to
proceed simultaneously and only serialize writes — a natural fit for a
read-heavy registry workload. This is the planned next step if read contention
is observed.

## 8. Error Handling

All failures return a uniform JSON envelope:

```json
{
  "error": {
    "code": "invalid_request",
    "message": "id is required"
  }
}
```

Error classes map to HTTP status codes:

| Code                 | HTTP status | Meaning                          |
| -------------------- | ----------- | -------------------------------- |
| `invalid_request`    | 400         | Malformed body or failed validation |
| `conflict`           | 409         | Service ID already exists        |
| `method_not_allowed` | 405         | Unsupported HTTP method on route |

A single envelope shape means clients parse errors the same way everywhere,
rather than special-casing each endpoint.

## 9. Known Limitations

This is a minimal v1 implementation.

- Data is stored in memory only and is lost when the server restarts
- No database persistence
- No authentication or authorization
- No update or delete endpoints
- No pagination or filtering
- Single-process only; cannot scale horizontally

## 10. Future Improvements

- Unit tests and CI (formatting, test, build checks)
- Update/delete endpoints (`PUT`/`DELETE /services/{id}`)
- Filtering by environment (`GET /services?environment=prod`)
- Persistent storage with PostgreSQL behind a `Store` interface
- Expanded service metadata (`team`, `repo_url`, `slack_channel`, `tier`)
- Service readiness / onboarding checklist endpoint
- `sync.RWMutex` for concurrent reads under load
- Structured logging and request-logging middleware
- OpenAPI documentation

## Architecture

```text
Client / curl / Postman
          |
          v
  Go HTTP API  (net/http, listening on :8080)
          |
          v
  ServeMux router          ->  /health  /version  /services
          |
          v
  Handlers                 ->  JSON decode, validation, error envelope
          |
          v
  Service Store
          |
          v
  Mutex-protected in-memory  map[string]Service
```
