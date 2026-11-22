# Tasks

Simple ToDo web app for learning Go application development (SQLite, HTMX, OpenTelemetry).

## Requirements

- [mise](https://mise.jdx.dev/)

## Setup

```bash
export ADMIN_USERNAME=admin ADMIN_PASSWORD=admin
mise install
mise run db:migrate
mise run serve
```

Open http://127.0.0.1:8000

## Checks

```bash
mise run 'lint:*'
mise run vuln
mise run test
mise run fmt:go
```

Go lint rules: `.golangci.yml`. Agent notes: `AGENTS.md`.

Dependency PRs: Dependabot (`.github/dependabot.yml`). Vulnerability gate: `govulncheck` in CI (`mise run vuln`).

GitHub Actions pins are checked by Dependabot + `pinact` (`mise run lint:action`). The mise CLI version in workflows (`jdx/mise-action` → `with.version`) is pinned separately and needs a manual/agent bump when you want a newer mise.
