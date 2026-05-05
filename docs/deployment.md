# Deployment

How a commit reaches production. Read [architecture.md](architecture.md) for the system shape and [development.md](development.md) for local builds.

## Two deploy paths

| Path | Trigger | Image | Manifests | Use when |
|---|---|---|---|---|
| **Release** | Tag push `v*` on `master` | GHCR (`ghcr.io/tamcore/reststop-navigator:<tag>`) | Helm chart on OCI (`ghcr.io/tamcore/charts/reststop-navigator`) | Anything user-facing — durable, GitOps-managed. |
| **Dev deploy** | `make dev-deploy-k8s` from the repo | Caller-supplied registry (set `IMAGE_REGISTRY`) | Helm-template + `kubectl apply` | Iterating on the deployed binary without cutting a tag. |

The release path is the source of truth. The dev path is for the author to spike unreleased changes; it bypasses any GitOps controller and is overwritten the next time the controller reconciles.

## Release pipeline (`.github/workflows/release.yaml`)

Triggered on tag pushes matching `v*`. Two jobs:

1. **`release`** — runs `goreleaser release --clean`:
   - Builds multi-arch (`linux/amd64`, `linux/arm64`) Go binaries with `-tags prodfrontend`, embedding the SvelteKit build.
   - Pushes Docker images to `ghcr.io/tamcore/reststop-navigator:<tag>` and `:latest`.
   - Creates a GitHub release with checksums + tarballs. Pre-release tags (`v*-alpha.*`, `v*-rc.*`) are auto-marked as "pre-release."
2. **`chart-release`** — packages and pushes the Helm chart:
   - `yq` rewrites `Chart.yaml` `version` and `appVersion` to the stripped tag (`v0.1.0` → `0.1.0`).
   - `helm package charts/reststop-navigator -u`.
   - `helm push` to `oci://ghcr.io/tamcore/charts`.
   - `helm template` is rendered and uploaded as `install.yaml` to the GitHub release for users who want bare manifests.

### Cutting a release

```sh
# Verify CI is green.
gh run list --branch master --limit 4

# Tag, push, watch.
git tag -a v0.1.0 -m "v0.1.0"
git push origin v0.1.0
gh run watch
```

Pre-releases use SemVer suffixes (`v0.1.0-alpha.0`, `v0.1.0-rc.1`, …). The `latest` tag still moves to pre-releases — that's fine while there's no stable release; revisit when the first stable cut happens.

## Production (GitOps)

Production deploys are managed by the user's GitOps controller, which pulls the released chart from `oci://ghcr.io/tamcore/charts/reststop-navigator` at a pinned version. The wiring (controller manifests, wrapper chart, per-cluster value overrides) lives in a **separate, private** infrastructure repo — see `AGENTS.md.local` for the details.

To bump production after a new release: bump the pinned chart version in that private repo, merge, then trigger a sync. Manual sync only for the alpha; auto-sync stays off until the project stabilises.

## Dev deploy (`make dev-deploy-k8s`)

For iterating on a change without cutting a tag. Reads required env vars from the caller (see `AGENTS.md.local` for the values; gitignored).

```sh
IMAGE_REGISTRY=<your-registry> \
INGRESS_HOST=<your-fqdn> \
make dev-deploy-k8s
```

The Makefile fails fast if either env var is unset. It:

1. Builds a multi-stage Docker image via `Dockerfile.dev` (frontend stage → backend stage with `-tags prodfrontend` → distroless final image).
2. Pushes to `${IMAGE_REGISTRY}/reststop-navigator:dev`.
3. Resolves the image digest, renders the chart with overrides:
   - `image.repository=${IMAGE_REGISTRY}/reststop-navigator`
   - `image.tag=dev`
   - `image.digest=sha256:...`
   - `ingress.hosts[0]=${INGRESS_HOST}`
   - `ingress.tls[0].hosts[0]=${INGRESS_HOST}`
4. `kubectl apply --wait`.

Watch the rollout:

```sh
kubectl --context <ctx> -n reststop-navigator rollout status deploy/reststop-navigator
```

A dev deploy is overwritten the next time the GitOps controller reconciles (it reverts to the chart's pinned image). That's by design — dev is for spikes, not durable changes.

## The chart in this repo (`charts/reststop-navigator/`)

Source of truth for the released chart. Defaults are intentionally empty for cluster-specific values:

| Value | Default | Notes |
|---|---|---|
| `image.repository` | `ghcr.io/tamcore/reststop-navigator` | Public GHCR image. |
| `image.tag` | `""` | Falls through to `Chart.AppVersion`. |
| `ingress.hosts` | `[]` | Caller must supply. |
| `ingress.tls` | `[]` | Caller must supply. |
| `ingress.certManagerIssuer` | `letsencrypt-prod` | cert-manager wires up TLS. |
| `redis.enabled` | `true` | In-chart single-replica Redis, no persistence. |
| `replicaCount` | `2` | PodDisruptionBudget keeps `minAvailable: 1`. |

Anything cluster-private (FQDNs, registry hosts) **must not** be defaulted in the chart — the privacy boundary in [AGENTS.md](../AGENTS.md) treats that as a release-blocker. Pass at deploy time via `--set` or a per-cluster values file.

## Pre-deploy gate

Per [AGENTS.md](../AGENTS.md): all CI workflows must be green (`test.yaml`, `e2e.yaml`, `commit-lint.yaml`) before:

- Running `make dev-deploy-k8s`.
- Tagging a release.

If CI is red, the next commit fixes it. No piling on. No deploying past a red branch even if local checks pass.

## Where to read next

- [architecture.md](architecture.md) — system overview + per-request data flow.
- [development.md](development.md) — clone, build, test, replay.
- [AGENTS.md](../AGENTS.md) — workflow rules.
