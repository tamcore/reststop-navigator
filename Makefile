VERSION ?= dev
COMMIT  := $$(git rev-parse HEAD 2>/dev/null || echo unknown)
DATE    := $$(date -u +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || echo unknown)
BRANCH  := $$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo unknown)

LDFLAGS := -s -w

.PHONY: help build test lint fmt vet golangci-lint helm-lint goreleaser-check coverage clean

help: ## Show this help message
	@echo "Reststop Navigator - Make targets:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build the reststop-navigator binary
	@echo "Building reststop-navigator..."
	@go build -ldflags "$(LDFLAGS)" -o bin/reststop-navigator ./cmd/server
	@echo "Binary built: bin/reststop-navigator"

fmt: ## Run go fmt
	go fmt ./...

vet: ## Run go vet
	go vet ./...

golangci-lint: ## Run golangci-lint via Docker
	docker run --rm -v "$(PWD)":"$(PWD)" -w "$(PWD)" golangci/golangci-lint:latest golangci-lint run --timeout=5m

helm-lint: ## Lint Helm chart (no-op until charts/ exists)
	@if [ -d charts/reststop-navigator ]; then helm lint ./charts/reststop-navigator -f ./charts/reststop-navigator/values.yaml; else echo "charts/reststop-navigator not present yet - skipping"; fi

goreleaser-check: ## Validate .goreleaser.yaml
	@if [ -f .goreleaser.yaml ]; then goreleaser check; else echo ".goreleaser.yaml not present yet - skipping"; fi

lint: fmt vet golangci-lint goreleaser-check helm-lint ## Run all linters and checks
	@echo "Linting complete!"

test: ## Run all tests with race detector and coverage
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...

coverage: test ## Print coverage by func and total
	@go tool cover -func=coverage.out

clean: ## Remove build artifacts
	rm -rf bin/ coverage.out coverage.html dist/
