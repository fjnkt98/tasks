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
mise run test
mise run fmt:go
```

Go lint rules: `.golangci.yml`. Agent notes: `AGENTS.md`.
