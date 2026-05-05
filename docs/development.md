# Development

How to clone, run, test, and contribute. Read [architecture.md](architecture.md) first if you haven't.

## Prerequisites

- **Go 1.26+** (`go.mod` is the source of truth)
- **Node 24+** + npm (frontend build & tests)
- **Docker + Docker Compose** (used for the full e2e stack and dev deploys)
- **Helm 3** + `kubectl` if you intend to deploy

Everything else (Playwright browsers, Go tools) is pulled by the relevant `make` / `npm` target on first run.

## Clone & run

```sh
git clone git@github.com:tamcore/reststop-navigator.git
cd reststop-navigator
```

### Backend-only (fastest feedback loop)

```sh
make build           # → bin/reststop-navigator (no frontend embedded)
./bin/reststop-navigator
# → :8080  /api/healthz returns {"status":"ok"}
```

The binary boots without Redis configured; it will refuse `/api/stops/upcoming` until you point it at one.

### Backend + frontend, docker-compose (matches CI / e2e)

```sh
docker compose -f docker-compose.dev.yaml up -d --build --wait
# Hits app on http://localhost:8080. Stubbed Overpass at :7000.
```

This is the same stack the e2e workflow runs. The `stubbed-overpass` service serves canned fixtures from `cmd/stubbedoverpass/fixtures/` so tests don't depend on the real Overpass instance.

### Production-mode local build (frontend embedded)

```sh
make build-prod      # builds the SvelteKit frontend then go build -tags prodfrontend
./bin/reststop-navigator
# → SPA served at /, API at /api/...
```

### Frontend dev server (hot reload, live reload)

```sh
cd web
npm install
npm run dev          # vite, http://localhost:5173, proxies /api → :8080
```

Run `make build` in another terminal so the API exists at `:8080`. The geolocation prompt only fires on `https://` or `localhost`, so `localhost:5173` is fine.

## Tests

| Command | What it covers |
|---|---|
| `make test` | All Go packages with race detector + coverage profile |
| `make coverage` | Per-function + total coverage report |
| `cd web && npm run check` | `svelte-check` + TypeScript |
| `cd web && npm run test:unit` | Vitest unit tests for stores, deep-link builders, API client |
| `cd web && npx playwright test` | E2E against the local dev stack (start it with `docker compose -f docker-compose.dev.yaml up -d --wait` first) |
| `make lint` | `go vet` + `golangci-lint` + `helm lint` + `goreleaser check` |

CI runs the same matrix on every push. Coverage gate is **≥ 80 %** for every `internal/...` package.

### TDD discipline (non-negotiable)

Per [AGENTS.md](../AGENTS.md): write the failing test first, then the implementation. Don't land `// TODO: add tests`. A typical Go change adds a `_test.go` table-driven test in the same commit (or one commit ahead) of the implementation.

## Replay against a real GPX trace

`cmd/replay` walks a GPX file `<trkpt>` by `<trkpt>`, derives heading + speed from time deltas, and hits a running backend's `/api/stops/upcoming` for each point — useful for verifying matching/ranking against a real drive without driving.

```sh
RESTSTOP_GPX_FIXTURE=~/Downloads/some-route.gpx go run ./cmd/replay
# Or against the deployed app:
RESTSTOP_GPX_FIXTURE=~/Downloads/some-route.gpx \
  go run ./cmd/replay -target https://restops.example.com
```

`*.gpx` is gitignored — personal data must never be committed.

## Common operations

```sh
# Update Go deps
go get -u ./... && go mod tidy

# Update frontend deps
cd web && npm update && npm audit

# Regenerate the embedded frontend after Svelte changes (production binary)
make build-prod

# Inspect the embedded FS
go run ./cmd/server &  # then `curl localhost:8080/_app/...` to verify assets
```

## Repo layout

```
cmd/server/                    # Go entrypoint + serve.Run
cmd/replay/                    # GPX-driven replay CLI
cmd/stubbedoverpass/           # E2E fixture server, canned Overpass JSON
internal/api/handlers/         # /api/stops/upcoming, /api/stops/detail, /api/healthz
internal/api/middleware/       # access log, security headers
internal/cache/                # Redis tile cache (lazy, 0.5° tiles)
internal/geo/                  # bearing, distance, match, ahead — pure functions
internal/overpass/             # HTTP client, queries, decode, enrich
internal/stops/                # service orchestration, filters, ranking
web/src/routes/                # +page.svelte (list), stop/[id]/+page.svelte
web/src/lib/components/        # StopCard, FilterChips, ThemeToggle, RoadShield
web/src/lib/stores/            # geo, filters, theme
web/src/lib/api/               # typed fetch wrapper
web/src/styles/global.css      # design-system tokens
charts/reststop-navigator/     # Helm chart (host + registry caller-supplied)
docs/                          # this directory
```

## Where to read next

- [architecture.md](architecture.md) — system overview + per-request data flow.
- [deployment.md](deployment.md) — release pipeline, GitOps notes, dev deploys.
- [AGENTS.md](../AGENTS.md) — workflow rules for AI coding agents.
