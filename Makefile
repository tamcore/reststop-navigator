VERSION ?= dev
COMMIT  := $$(git rev-parse HEAD 2>/dev/null || echo unknown)
DATE    := $$(date -u +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || echo unknown)
BRANCH  := $$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo unknown)

LDFLAGS := -s -w

IMAGE_REGISTRY ?= reg.meh.wf
IMAGE_NAME     ?= reststop-navigator
IMAGE_TAG      ?= dev
DEPLOY_NS      ?= reststop-navigator
INGRESS_HOST   ?=
KUBE_CONTEXT   ?=
KUBECTL_CTX    := $(if $(KUBE_CONTEXT),--context $(KUBE_CONTEXT),)

.PHONY: help build test lint fmt vet golangci-lint helm-lint goreleaser-check coverage clean dev-deploy-k8s

help: ## Show this help message
	@echo "Reststop Navigator - Make targets:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build the reststop-navigator binary (without embedded frontend)
	@echo "Building reststop-navigator..."
	@go build -ldflags "$(LDFLAGS)" -o bin/reststop-navigator ./cmd/server
	@echo "Binary built: bin/reststop-navigator"

build-prod: ## Build the production binary with the SvelteKit frontend embedded
	@echo "Building frontend..."
	@cd web && npm ci --silent && npm run build
	@echo "Building reststop-navigator with embedded frontend..."
	@go build -ldflags "$(LDFLAGS)" -tags prodfrontend -o bin/reststop-navigator ./cmd/server
	@echo "Binary built: bin/reststop-navigator (with frontend)"

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

demo-tracks: ## Convert ~/Downloads/route-*.gpx → web/src/lib/data/tracks/*.json (gitignored)
	go run ./cmd/gpx2demo -src ~/Downloads -out web/src/lib/data/tracks

clean: ## Remove build artifacts
	rm -rf bin/ coverage.out coverage.html dist/

dev-deploy-k8s: ## Build dev image, push to IMAGE_REGISTRY, deploy to K8s namespace DEPLOY_NS
	@if [ -z "$(IMAGE_REGISTRY)" ]; then echo "ERROR: IMAGE_REGISTRY is not set. See AGENTS.md.local for dev deployment env vars."; exit 1; fi
	@if [ -z "$(INGRESS_HOST)" ]; then echo "ERROR: INGRESS_HOST is not set. See AGENTS.md.local for dev deployment env vars."; exit 1; fi
	@echo "Building dev image..."
	@docker build \
		--target app \
		-t $(IMAGE_REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG) \
		-f Dockerfile.dev \
		.
	@echo ""
	@echo "Pushing to $(IMAGE_REGISTRY)..."
	@docker push $(IMAGE_REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)
	@echo ""
	@IMAGE_DIGEST=$$(docker inspect $(IMAGE_REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG) --format='{{index .RepoDigests 0}}' | cut -d'@' -f2); \
	echo "Using digest: $$IMAGE_DIGEST"; \
	echo ""; \
	echo "Ensuring namespace $(DEPLOY_NS) exists..."; \
	kubectl $(KUBECTL_CTX) get namespace $(DEPLOY_NS) >/dev/null 2>&1 || kubectl $(KUBECTL_CTX) create namespace $(DEPLOY_NS); \
	echo ""; \
	echo "Deploying to namespace $(DEPLOY_NS)..."; \
	kubectl $(KUBECTL_CTX) -n $(DEPLOY_NS) delete deploy/reststop-navigator --ignore-not-found; \
	helm template reststop-navigator ./charts/reststop-navigator \
		--namespace $(DEPLOY_NS) \
		--set image.repository="$(IMAGE_REGISTRY)/$(IMAGE_NAME)" \
		--set image.tag="$(IMAGE_TAG)" \
		--set image.digest="$$IMAGE_DIGEST" \
		--set ingress.hosts[0]="$(INGRESS_HOST)" \
		--set ingress.tls[0].hosts[0]="$(INGRESS_HOST)" \
		$(HELM_EXTRA_ARGS) \
	| kubectl $(KUBECTL_CTX) apply -n $(DEPLOY_NS) -f - --wait
	@echo ""
	@echo "Deployment dispatched. Watch:"
	@echo "  kubectl -n $(DEPLOY_NS) rollout status deploy/reststop-navigator"
