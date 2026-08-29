# Repository Guidelines

## Project Structure & Module Organization

The Go module lives in `kitchen/`; run Go and Task commands there. The executable entry point is `kitchen/cmd/kitchen/main.go`. Application code is grouped by capability under `kitchen/internal/` (`establishment`, `menu`, and `order`), with shared code in `internal/infrastructure/`. Keep domain and application logic independent of adapters; place HTTP and PostgreSQL integrations in each module's `adapter/` package.

Database migrations are embedded from `kitchen/migrations/schemes/` and managed with Goose. Documentation is under `docs/`: the OpenAPI contract is in `docs/api/`, PlantUML sources in `docs/cjm/`, and rendered diagrams in `docs/imgs/`. The root `README.md` records business rules and architecture.

## Build, Test, and Development Commands

From `kitchen/`, use:

- `go run ./cmd/kitchen` — run the service locally (default listen address `:8080`).
- `go build ./...` — compile every package.
- `go test ./...` — run the full test suite.
- `task fmt` — format imports and Go source with golangci-lint.
- `task lint` — run the configured standard and additional linters.
- `task check` — format, lint, and test; run this before opening a PR.

Task commands require Task and `golangci-lint`. The module declares Go 1.26.4.

## Coding Style & Naming Conventions

Use `gofmt`/`goimports` and tabs for Go indentation. Package names are short, lowercase, and singular. Exported identifiers use `PascalCase`; unexported identifiers use `camelCase`. Name constructors `New...`, wrap errors with context using `%w`, and pass `context.Context` through request and database boundaries. Follow `.golangci.yaml`; explain any linter suppression.

## Testing Guidelines

Use Go's `testing` package and colocate `*_test.go` files with their package. Name tests `TestFunction_Scenario`; prefer table-driven cases for business rules and validation. No coverage threshold is enforced, but new behavior and bug fixes should include focused tests.

## Commit & Pull Request Guidelines

History uses concise, imperative subjects such as `Add: migrations` or `Update: documentation`. Keep commits focused and follow that pattern. Pull requests should summarize behavior, note schema or OpenAPI impacts, link the issue, and list verification commands. Update rendered diagrams when PlantUML sources change.

## Configuration & Security

Configuration is environment-based; supported HTTP settings include `LISTEN_ADDRESS`, `READ_TIMEOUT`, `WRITE_TIMEOUT`, `SHUTDOWN_TIMEOUT`, and `HTTP_IDLE_TIMEOUT`. Never commit credentials or local environment files. Review SQL migrations for safe forward application and preserve existing migration history.
