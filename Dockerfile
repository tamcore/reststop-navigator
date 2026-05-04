# syntax=docker/dockerfile:1

# Production Dockerfile for goreleaser (dockers_v2).
# The reststop-navigator binary is pre-built by goreleaser with the
# frontend embedded via //go:embed, so no separate frontend build stage
# is needed.

FROM gcr.io/distroless/static-debian12:nonroot

ARG TARGETPLATFORM

LABEL org.opencontainers.image.source="https://github.com/tamcore/reststop-navigator"
LABEL org.opencontainers.image.description="Reststop Navigator - find upcoming highway rest stops"
LABEL org.opencontainers.image.licenses="MIT"

COPY ${TARGETPLATFORM}/reststop-navigator /reststop-navigator

EXPOSE 8080

USER 65532:65532

ENTRYPOINT ["/reststop-navigator"]
