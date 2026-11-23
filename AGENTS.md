# Agent guide

Learning project: a small Go ToDo web app (SQLite + HTMX). Prefer small, reviewable changes. Architecture and security still need deliberate improvement; do not "enterprise-rewrite" in one pass.

## Principles

Build with the Go standard library as far as practical. Prefer `net/http`, `database/sql`, `html/template`, `log/slog`, and other stdlib packages over application frameworks or heavy middleware stacks. Keep third-party dependencies minimal and intentional (drivers, OTel, test helpers, etc.). Feature work and rearchitecture must keep this goal; do not introduce a web framework or similar abstraction layer unless explicitly asked.

Prefer self-explanatory code over narrative comments. Do not add comments that only restate context, intent, or history for the reader (there is no golangci-lint rule for this yet; treat it as a project convention). Allowed exceptions: GoDoc on exported APIs when useful, `//nolint` / `//go:` / build and codegen directives, and brief comments required by a tool or a non-obvious invariant that the code cannot express.

## Testing

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
- Prefer extending existing packages under `internal/` over new top-level layouts unless asked.
- Secrets and production deploys belong in env / GCP, not in commits.
