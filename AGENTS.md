# Agent guide

Learning project: a small Go ToDo web app (SQLite + HTMX). Prefer small, reviewable changes. Architecture and security still need deliberate improvement; do not "enterprise-rewrite" in one pass.

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
mise run test
```

Go lint config lives in `.golangci.yml` (golangci-lint v2). Format with:

```bash
mise run fmt:go
```

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
