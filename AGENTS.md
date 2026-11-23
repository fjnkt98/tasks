# Agent guide

Learning project: a small Go ToDo web app (SQLite + HTMX). Prefer small, reviewable changes. Architecture and security still need deliberate improvement; do not "enterprise-rewrite" in one pass.

## Principles

Build with the Go standard library as far as practical. Prefer `net/http`, `database/sql`, `html/template`, `log/slog`, and other stdlib packages over application frameworks or heavy middleware stacks. Keep third-party dependencies minimal and intentional (drivers, OTel, test helpers, etc.). Feature work and rearchitecture must keep this goal; do not introduce a web framework or similar abstraction layer unless explicitly asked.

Prefer self-explanatory code over narrative comments. Do not add comments that only restate context, intent, or history for the reader (there is no golangci-lint rule for this yet; treat it as a project convention). Allowed exceptions: GoDoc on exported APIs when useful, `//nolint` / `//go:` / build and codegen directives, and brief comments required by a tool or a non-obvious invariant that the code cannot express.

## Layout (clean architecture)

Dependency direction is inward: `adapter/web → usecase → domain`, and `repository` implements ports defined in `usecase`. `domain` has no I/O.

```
cmd/tasks/                 # process entrypoint (calls app)
internal/
  domain/                  # pure, immutable value objects (no I/O)
  usecase/                 # orchestration + repository ports (Save domain objects)
  repository/              # SQLite implementations of usecase ports
  app/                     # composition root (wires repository + usecase + web)
  adapter/
    web/                   # HTTP controllers (request/response only)
  schema/                  # SQL schema for sqlite3def
  testutil/                # shared test helpers
e2e/                       # HTTP+DB end-to-end tests
```

Domain objects are immutable: fields are unexported, and changes go through methods that return a new value (for example `WithTitle`). Repository ports accept/return domain objects (`Save`, `Find*`). Do not import `adapter/*` or `repository` from `domain` or `usecase` (except that `repository` implements `usecase` ports). Only `app` composes them.

## Testing

Before and during clean-architecture work, prefer **E2E tests** that exercise real HTTP endpoints (request/response shape, status, redirects, cookies) against a real SQLite DB and assert durable writes. Layer unit tests (repository / usecase / middleware in isolation) are not the source of truth for refactor safety; do not rely on them to prove endpoint or persistence compatibility.

E2E tests live under `e2e/`. They should fail if routes, form fields, response codes/locations, or DB side effects change unintentionally.

Always write table-driven tests (`[]struct{...}` + `t.Run`), even when there is only one case. There is no golangci-lint rule that enforces TDT yet; treat this as a project convention.

Subtest names (`t.Run` の第1引数) must be Japanese descriptions of the expected state or behavior, prefixed as follows:

- Success / happy path: start with `ok: `
- Failure / error path: start with `ng: `

Prefer forms such as「◯◯は◯◯である」or「◯◯すると、◯◯になる」(例: `ok: 正しいパスワードを入力すると、サインインに成功する`, `ng: パスワードを空にすると、登録に失敗する`).

## Tooling

Use [mise](https://mise.jdx.dev/). Required env (also used by CI):

```bash
export ADMIN_USERNAME=admin ADMIN_PASSWORD=admin
```

Install pinned tools:

```bash
mise install
```

## Verify before finishing work

```bash
mise run 'lint:*'
mise run vuln
mise run test
```

Go lint config lives in `.golangci.yml` (golangci-lint v2). Format with:

```bash
mise run fmt:go
```

Dependency updates: Dependabot (`.github/dependabot.yml`). Vulnerability scan: `govulncheck` via `mise run vuln`.

### Workflow / Action versions

| What | How it stays current |
| --- | --- |
| `uses: owner/action@sha` | Dependabot `github-actions` (weekly) + `pinact run --check` in CI |
| mise CLI `with.version` in workflows | **Not** updated by Dependabot; bump when raising mise (or omit `version` to let mise-action pick the newest release older than `minimum_release_age`, default 24h) |

Prefer SHA-pinned actions with a `# vX.Y.Z` comment (pinact). Do not switch to floating `@v5` tags.

DB schema changes:

```bash
mise run db:migrate
mise run db:check
```

## Local app

```bash
mise run db:migrate
mise run serve   # http://127.0.0.1:8000
```

Optional tracing stack: `docker compose up -d` (OTel collector + Jaeger).

## Conventions for agents

- Keep CGO enabled (`CGO_ENABLED=1`); tests need `sqlite3def` on PATH via mise.
- Do not weaken lint rules to hide new issues; fix code or add a specific `//nolint:linter // reason`.
- Prefer extending the existing clean-architecture packages under `internal/` (`domain`, `usecase`, `repository`, `app`, `adapter/web`) over inventing parallel layouts.
- Secrets and production deploys belong in env / GCP, not in commits.
