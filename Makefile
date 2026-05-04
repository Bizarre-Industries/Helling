# Helling Makefile
# Every target referenced in AGENTS.md is defined here.

SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c
.DEFAULT_GOAL := help

# ---- Configurable knobs --------------------------------------------------

GO            ?= go
BUN           ?= bun
GOLANGCI_LINT ?= golangci-lint
GOFUMPT       ?= gofumpt
GOIMPORTS     ?= goimports
DOCKER        ?= docker

VERSION       ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT        ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_TIME    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS       := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)

GO_PKGS_HELLINGD     := ./apps/hellingd/...
GO_PKGS_HELLING_CLI  := ./apps/helling-cli/...
GO_PKGS_HELLING_PROXY:= ./apps/helling-proxy/...
GO_TEST_FLAGS        := -race -count=1
GO_BUILD_FLAGS       := -trimpath -ldflags '$(LDFLAGS)'

OUT_DIR := ./bin

# ---- Help ----------------------------------------------------------------

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "Helling — make targets\n\n"} \
		/^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

# ---- One-shot bootstrap --------------------------------------------------

.PHONY: dev-setup
dev-setup: ## Install required tools, frontend deps, and git hooks
	@echo "→ installing Go tools (oapi-codegen, golangci-lint, gofumpt, goimports)"
	$(GO) install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
	$(GO) install mvdan.cc/gofumpt@latest
	$(GO) install golang.org/x/tools/cmd/goimports@latest
	@echo "→ ensure golangci-lint is installed (https://golangci-lint.run/welcome/install/)"
	@command -v $(GOLANGCI_LINT) >/dev/null || { echo "golangci-lint missing"; exit 1; }
	@if [ -d web ] && [ -f web/package.json ]; then \
		echo "→ installing frontend deps with bun"; \
		cd web && $(BUN) install; \
	fi
	@if [ -d .git ]; then \
		echo "→ installing git hooks"; \
		mkdir -p .git/hooks; \
		printf '#!/bin/sh\nexec make fmt-check lint\n' > .git/hooks/pre-commit; \
		chmod +x .git/hooks/pre-commit; \
	fi

# ---- Code generation -----------------------------------------------------

.PHONY: generate
generate: generate-go generate-web ## Regenerate all OpenAPI artifacts

.PHONY: generate-go
generate-go: ## Regenerate Go server and client code from OpenAPI
	@echo "→ generating hellingd server"
	cd apps/hellingd && $(GO) generate ./api/...
	@echo "→ generating helling-cli client"
	@if [ -d apps/helling-cli/internal/client ]; then \
		cd apps/helling-cli && $(GO) generate ./internal/client/...; \
	fi

.PHONY: generate-web
generate-web: ## Regenerate the TypeScript client (Orval)
	@if [ -d web ] && [ -f web/package.json ]; then \
		cd web && $(BUN) run gen:api; \
	else \
		echo "→ skipping: web/ not present"; \
	fi

.PHONY: check-generated
check-generated: generate ## Fail if generated artifacts drift from spec
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "✗ generated artifacts are out of date. Run 'make generate' and commit."; \
		git --no-pager diff --stat; \
		exit 1; \
	fi
	@echo "✓ generated artifacts are clean"

# ---- Format / lint -------------------------------------------------------

.PHONY: fmt
fmt: ## Format all Go and frontend code in place
	$(GOFUMPT) -w .
	$(GOIMPORTS) -w -local github.com/Bizarre-Industries/helling .
	@if [ -d web ] && [ -f web/package.json ]; then \
		cd web && $(BUN) run fmt || true; \
	fi

.PHONY: fmt-check
fmt-check: ## Verify formatting without modifying files
	@diff=$$($(GOFUMPT) -d .); if [ -n "$$diff" ]; then echo "$$diff"; echo "✗ run make fmt"; exit 1; fi
	@echo "✓ go formatting clean"

.PHONY: lint
lint: ## Run static analysis
	cd apps/hellingd && $(GOLANGCI_LINT) run ./...
	@if [ -d apps/helling-cli ]; then cd apps/helling-cli && $(GOLANGCI_LINT) run ./...; fi
	@if [ -d apps/helling-proxy ]; then cd apps/helling-proxy && $(GOLANGCI_LINT) run ./...; fi

# ---- Tests ---------------------------------------------------------------

.PHONY: test
test: test-hellingd test-cli test-proxy ## Run unit tests for all Go modules

.PHONY: test-hellingd
test-hellingd:
	$(GO) test -tags devauth $(GO_PKGS_HELLINGD) $(GO_TEST_FLAGS)

.PHONY: test-cli
test-cli:
	@if [ -d apps/helling-cli ]; then \
		$(GO) test $(GO_PKGS_HELLING_CLI) $(GO_TEST_FLAGS); \
	fi

.PHONY: test-proxy
test-proxy:
	@if [ -f apps/helling-proxy/go.mod ]; then \
		$(GO) test $(GO_PKGS_HELLING_PROXY) $(GO_TEST_FLAGS); \
	fi

.PHONY: test-cover
test-cover: ## Run tests with coverage report
	$(GO) test -tags devauth -race -coverprofile=coverage.out ./apps/...
	$(GO) tool cover -func=coverage.out

# ---- Build ---------------------------------------------------------------

.PHONY: build
build: build-hellingd build-cli build-proxy ## Build all binaries to $(OUT_DIR)

.PHONY: build-hellingd
build-hellingd: $(OUT_DIR)
	$(GO) build $(GO_BUILD_FLAGS) -o $(OUT_DIR)/hellingd ./apps/hellingd

.PHONY: build-cli
build-cli: $(OUT_DIR)
	@if [ -d apps/helling-cli ]; then \
		$(GO) build $(GO_BUILD_FLAGS) -o $(OUT_DIR)/helling ./apps/helling-cli; \
	fi

.PHONY: build-proxy
build-proxy: $(OUT_DIR)
	@if [ -f apps/helling-proxy/go.mod ]; then \
		$(GO) build $(GO_BUILD_FLAGS) -o $(OUT_DIR)/helling-proxy ./apps/helling-proxy; \
	fi

$(OUT_DIR):
	mkdir -p $(OUT_DIR)

# ---- Web -----------------------------------------------------------------

.PHONY: web-dev
web-dev: ## Run the frontend dev server
	cd web && $(BUN) run dev

.PHONY: web-build
web-build: ## Build the production frontend bundle
	cd web && $(BUN) run build

# ---- Container -----------------------------------------------------------

.PHONY: docker
docker: ## Build the helling Docker image
	$(DOCKER) build -t helling:$(VERSION) -f deploy/Dockerfile .

# ---- Security ------------------------------------------------------------

.PHONY: security-fast
security-fast: ## Quick security checks (gitleaks + govulncheck)
	@command -v gitleaks >/dev/null && gitleaks detect --no-banner || echo "→ gitleaks not installed, skipping"
	@if command -v govulncheck >/dev/null; then \
		for mod in apps/hellingd apps/helling-cli; do \
			if [ -f "$$mod/go.mod" ]; then \
				echo "→ govulncheck $$mod"; \
				(cd "$$mod" && govulncheck ./...); \
			fi; \
		done; \
	else \
		echo "→ govulncheck not installed, skipping"; \
	fi

.PHONY: security
security: security-fast ## Full security scan (slower)
	cd apps/hellingd && $(GOLANGCI_LINT) run --enable=gosec ./...
	@if [ -f apps/helling-cli/go.mod ]; then cd apps/helling-cli && $(GOLANGCI_LINT) run --enable=gosec ./...; fi

# ---- Aggregate gates -----------------------------------------------------

.PHONY: check
check: fmt-check lint test ## CI-equivalent local gate

# ---- Cleanup -------------------------------------------------------------

.PHONY: clean
clean: ## Remove build outputs
	rm -rf $(OUT_DIR) coverage.out
	@if [ -d web/dist ]; then rm -rf web/dist; fi
