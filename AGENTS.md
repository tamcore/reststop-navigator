# AGENTS.md

Guidance for AI coding agents (Claude Code, Codex, etc.) working in this repository.

`CLAUDE.md` is a symlink to this file — keep all agent guidance here, not there.

## Repository overview

Reststop Navigator is a PWA + Go backend that lists upcoming highway rest stops based on the driver's live GPS, filterable by amenity (fuel, EV, food, WC, 24/7, dog). MVP coverage: DE, AT, SK, CZ. Data source is OpenStreetMap via the Overpass API; Redis is the only stateful dependency. Frontend is SvelteKit, embedded into the Go binary via `//go:embed`.

## Local secrets file: AGENTS.md.local

Deployment-specific configuration (registry hosts, ingress FQDNs, kube context name, etc.) lives in **`AGENTS.md.local`** at the repo root. That file is **gitignored** and must never be committed. If you need to deploy a dev build to the user's cluster, read `AGENTS.md.local` for the env vars to set.

If `AGENTS.md.local` is missing, ask the user to populate it — do **not** infer or hardcode private values from prior sessions, history, or context summaries.

## Privacy boundary (non-negotiable)

The following are **PRIVATE** and must never appear in tracked files (Helm values, Makefiles, code comments, README, workflows, screenshots, commit messages):

- The user's wildcard DNS suffix and any subdomain thereof (the actual suffix is named only in `AGENTS.md.local`).
- Public IPs that resolve to user-owned infrastructure.
- Internal registry hostnames.
- The `kube-context` name used for the user's cluster.
- Anything else explicitly marked private in `AGENTS.md.local`.

If you find any of these in the working tree before committing, treat it as a release-blocking bug and remove it. Pass them only at deploy time via env vars / `--set` flags read from `AGENTS.md.local`.

## Documentation discipline

- **`README.md` and `AGENTS.md` must be kept up-to-date at all times.** Any change that affects how a user runs, builds, deploys, or interacts with the app updates `README.md` in the same commit. Any change that affects how an agent should approach the codebase updates `AGENTS.md` in the same commit.
- **Screenshots are part of the docs.** When the UI changes meaningfully, refresh `docs/screenshots/list.png` and `docs/screenshots/detail.png` (the two reference shots, both linked from `README.md`) so they reflect the current UI. Take new shots from the running app at the standard mobile viewport (390×844). Never commit a screenshot whose URL bar shows the user's private FQDN — use a local hostname or crop.

## Workflow rules

### Git
- Push directly to `master`. No feature branches, no PRs for this repo (per user instruction).
- Conventional Commits (`feat:`, `fix:`, `chore:`, `test:`, `docs:`, `refactor:`, `perf:`, `ci:`).
- One logical change per commit; small and reviewable.
- Push after each commit (`git push origin master`).

### TDD
- Write the failing test first, then the implementation. Coverage gate is **≥ 80 %** for `internal/...` packages, enforced in CI.
- Never land `// TODO: add tests`.

### CI / E2E gate before any deployment or release
- **All CI workflows must be green** (`test.yaml`, `e2e.yaml`, `commit-lint.yaml`) before:
  - Running `make dev-deploy-k8s`.
  - Tagging a release (`v*` tag triggering `release.yaml`).
- If a CI run is red, the next commit fixes it. No piling on. No deploying from a red branch.
- Verify locally before pushing when feasible: `make lint && make test` and `cd web && npm run check && npm run test:unit && npx playwright test` — but never deploy past a red remote CI run, even if local checks pass.

## Project layout

```
cmd/server/         # Go entrypoint + serve.Run
cmd/replay/         # GPX-driven replay CLI (offline verification)
internal/api/       # chi router, handlers, middleware (incl. access_log)
internal/cache/     # Redis tile cache (lazy, 0.5° tiles)
internal/geo/       # bearing, distance, match, ahead — pure functions
internal/overpass/  # Overpass HTTP client + queries + decode + enrich
internal/stops/     # service orchestration + filters + ranking
web/                # SvelteKit PWA (static adapter), embedded via //go:embed
charts/reststop-navigator/  # Helm chart (host/registry are caller-supplied)
docs/screenshots/   # README screenshots — keep current
```

## Deployment

Deployment values (`IMAGE_REGISTRY`, `INGRESS_HOST`, kube context, etc.) are caller-supplied. The chart defaults `ingress.hosts` and `ingress.tls` to empty so a public render reveals nothing. See `AGENTS.md.local` for the actual env vars.

Once env is sourced:
```sh
make dev-deploy-k8s
kubectl --context <ctx> -n reststop-navigator rollout status deploy/reststop-navigator
```

Releases (`v*` tag) flow through `goreleaser` → GHCR. Do not tag a release with red CI.

## Things to avoid

- Hardcoding the user's FQDN, IP, or registry host anywhere in tracked files.
- Adding `kubectl --context <name>` calls with the user's real context name baked in.
- Committing GPX files (personal data — already in `.gitignore`).
- Embedding screenshots that show the production hostname in the URL bar.
- Skipping CI before deploy "just this once".
