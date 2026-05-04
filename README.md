# Reststop Navigator

[![Tests](https://github.com/tamcore/reststop-navigator/actions/workflows/test.yaml/badge.svg)](https://github.com/tamcore/reststop-navigator/actions/workflows/test.yaml) [![E2E](https://github.com/tamcore/reststop-navigator/actions/workflows/e2e.yaml/badge.svg)](https://github.com/tamcore/reststop-navigator/actions/workflows/e2e.yaml) [![Go](https://img.shields.io/github/go-mod/go-version/tamcore/reststop-navigator)](https://github.com/tamcore/reststop-navigator/blob/master/go.mod)

A PWA that, given your live GPS, identifies which highway and direction you're driving and lists the upcoming rest stops — filterable by fuel, EV charging, food, toilets, 24/7 opening, and dog-friendliness.

When you've picked one, the detail page hands off to Google Maps, Apple Maps, or Waze with the rest stop as a destination.

## Coverage

MVP supports motorways and trunk roads in:

- Germany
- Austria
- Slovakia
- Czechia

## Architecture

- **Backend:** Go (chi router), Redis cache.
- **Frontend:** SvelteKit PWA, embedded into the Go binary.
- **Data:** OpenStreetMap via the Overpass API. Country datasets are refreshed weekly into Redis.
- **Deploy:** Helm chart + Kubernetes (Gateway API HTTPRoute), released via goreleaser.

No accounts. No tracking. The browser asks for geolocation; the backend never stores per-user data.

## Status

Pre-MVP. See `docs/` (TBD) for roadmap.
