# Repository Guidelines

## Project Structure & Module Organization

- `main.go`: service entrypoint.
- `router/`: Gin route wiring (API under `/api/v1`, Swagger at `/swagger/index.html`).
- `internal/`: application code (handlers/controllers in `internal/api`, business logic in `internal/service`, data access in `internal/dal`, models/queries in `internal/model` + `internal/query`, middleware in `internal/middleware`).
- `initialize/`: bootstrapping (Viper config, DB/Redis, logging, cron, caches).
- `configs/`: runtime configuration (`conf.yml`, `conf-dev.yml`, message templates, `casbin.conf`, RSA keys).
- `sql/`: schema/seed SQL (update `sql/1.sql` when adding new SQL).
- `static/`, `files/`: static assets and runtime file/log storage.
- `cmd/`: utilities (notably `cmd/gen` for GORM Gen and `cmd/iot-platform-autotest` for automated MQTT/platform tests).
- `test/`: Go tests (some are integration-style and require a DB).

## Build, Test, and Development Commands

- Run locally: `go run .` (defaults to `:9999`).
- Unit/integration tests: `go test ./...` (may require Postgres/Redis configured).
- Local DB-backed tests: `run_env=localdev go test -v ./...` (expects a local config file; see below).
- Regenerate Swagger docs: `swag init` (run at repo root).
- (Optional) Build container: `docker build -t thingspanel-backend .`

## Coding Style & Naming Conventions

- Format: `gofmt`/`go fmt ./...` (required before PRs).
- Keep exported identifiers in `CamelCase`, packages in `lowercase`.
- Logging: use `logrus`; configuration: use `viper` and set defaults when keys may be missing.
- Prefer RESTful paths/methods (e.g., `POST /resources`, `GET /resources/{id}`, `PUT /resources/{id}`, `DELETE /resources/{id}`).
- For DB models/queries, prefer generating via `cmd/gen` (`cd cmd/gen && go run .`) rather than hand-writing.

## Testing Guidelines

- Framework: Go `testing` with `github.com/stretchr/testify/require`.
- Test files use Go conventions: `*_test.go`, functions `TestXxx(t *testing.T)`.
- If a test expects `configs/conf-localdev.yml` or `configs/conf-push-test.yml`, create local copies from `configs/conf-dev.yml` and keep credentials out of commits.

## Commit & Pull Request Guidelines

- Use Conventional Commits where possible: `feat(scope): …`, `fix: …`, `refactor: …`, `docs: …`, `chore: …` (English preferred).
- PRs should include: summary, impacted modules (`internal/...`, `router/...`, `sql/...`), and notes for config/migration or Swagger regeneration when applicable.

## Security & Configuration Tips

- Default config loads from `configs/conf.yml`; env vars can override with `GOTP_` prefix (e.g., `GOTP_DB_PSQL_HOST`, `GOTP_SERVICE_HTTP_PORT`).
- Never commit real secrets (JWT keys, DB passwords, API keys); use placeholders in config changes.
