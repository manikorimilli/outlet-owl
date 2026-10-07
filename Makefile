# OutletOwl: Go server (repository root) and React web app (web/).
# Every command lives here; `make help` lists them. CI runs these targets.
SHELL := /bin/bash
.DEFAULT_GOAL := help

# Tools installed with `go install` live in GOPATH/bin, which is often not on
# PATH; put it first so the pinned versions are the ones that run.
GOPATH_BIN := $(shell go env GOPATH 2>/dev/null)/bin
export PATH := $(GOPATH_BIN):$(PATH)
PNPM ?= pnpm
WEB := web
STATE := .bearing/state
SKIPPED := $(STATE)/.skipped
# The database the migrate and test-integration targets use is the server's:
# DATABASE_URL from the command line or the environment, else from .env read
# the way make dev reads it, else PostgreSQL on POSTGRES_PORT. Make does not
# read .env by itself, so a port moved there used to send make migrate to 5432.
POSTGRES_PORT ?= 5432
DOTENV_DATABASE_URL := $(shell [ -f .env ] && { set -a; . ./.env; set +a; printf '%s' "$$DATABASE_URL"; } 2>/dev/null)
DATABASE_URL ?= $(or $(DOTENV_DATABASE_URL),postgres://postgres:postgres@localhost:$(POSTGRES_PORT)/outlet_owl?sslmode=disable)
GO_FILES = $(shell find . -name '*.go' -not -path './web/*' -not -path './.git/*' -not -path './internal/store/*.sql.go')

# The offline gates `make check` runs, in order. vuln needs the network and
# test-integration needs PostgreSQL: CI runs both as jobs of their own.
GATES := go-fmt-check go-vet go-lint go-test web-format-check web-lint web-api-types-check web-typecheck web-test

# $(call skip,gate,tool): the tool is absent. Print it, record it, let the other
# gates run; `check` then fails. Never a silent pass.
define skip
{ mkdir -p $(STATE); echo "$(1): SKIPPED ($(2) not installed)"; echo "$(1) $(2)" >> $(SKIPPED); exit 0; }
endef

.PHONY: help setup hooks dev web-dev hash-password build check check-file fix vuln db db-down db-reset migrate migrate-down migrate-status sqlc doctor clean \
	go-fmt-check go-vet go-lint go-test test-integration web-format-check web-lint web-api-types web-api-types-check web-typecheck web-test web-build

help: ## List targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-18s %s\n", $$1, $$2}'

setup: ## Download Go modules, install web dependencies, install git hooks
	go mod download
	cd $(WEB) && $(PNPM) install --frozen-lockfile
	@$(MAKE) --no-print-directory hooks
	@echo "setup done; dev tools (sqlc, goose, golangci-lint, govulncheck) are listed in AGENTS.md"

hooks: ## Point git at .githooks (commit-msg, pre-commit, pre-push)
	bash .githooks/install.sh

dev: ## Run the Go server locally (reads .env if present; needs make db)
	@set -a; [ -f .env ] && . ./.env; set +a; go run ./cmd/api

web-dev: ## Run the Vite dev server on :5173 (forwards /api to :8080)
	cd $(WEB) && $(PNPM) run dev

# One read serves a terminal and a pipe: IFS= keeps leading and trailing
# spaces, and bash applies -s and -p only when stdin is a terminal.
hash-password: ## Print a bcrypt hash (cost 12) for the users file; the password is read without echo
	@IFS= read -rsp "Password: " pw; if [ -t 0 ]; then echo >&2; fi; \
	printf '%s' "$$pw" | go run ./cmd/hashpw

build: web-build ## Build the Go binary into bin/ and the web app into web/dist/
	CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=$$(git describe --tags --always --dirty)" -o bin/api ./cmd/api
	@echo "build: bin/api and $$(find $(WEB)/dist -type f | wc -l | tr -d ' ') files in $(WEB)/dist"

web-build: ## Typecheck and build the web app into web/dist/
	cd $(WEB) && $(PNPM) run build

# ---- Go gates ---------------------------------------------------------------

go-fmt-check: ## Fail if any Go file is unformatted
	@n=$$(echo $(GO_FILES) | wc -w | tr -d ' '); \
	[ "$$n" -gt 0 ] || { echo "go-fmt-check: 0 go files, nothing checked" >&2; exit 1; }; \
	command -v gofmt >/dev/null || $(call skip,go-fmt-check,gofmt); \
	files=$$(gofmt -l $(GO_FILES)); \
	[ -z "$$files" ] || { echo "unformatted (run make fix):"; echo "$$files"; exit 1; }; \
	echo "go-fmt-check: $$n files checked"

go-vet: ## go vet every package
	@n=$$(go list ./... | wc -l | tr -d ' '); \
	[ "$$n" -gt 0 ] || { echo "go-vet: 0 packages, nothing checked" >&2; exit 1; }; \
	go vet ./... && echo "go-vet: $$n packages checked"

go-lint: ## golangci-lint (default linters)
	@n=$$(go list ./... | wc -l | tr -d ' '); \
	command -v golangci-lint >/dev/null || $(call skip,go-lint,golangci-lint); \
	golangci-lint run ./... && echo "go-lint: $$n packages checked"

go-test: ## Go unit tests with the race detector
	@n=$$(go list ./... | wc -l | tr -d ' '); \
	[ "$$n" -gt 0 ] || { echo "go-test: 0 packages, nothing checked" >&2; exit 1; }; \
	set -o pipefail; go test -race -count=1 -cover ./... 2>&1 | tail -40 && echo "go-test: $$n packages checked"

test-integration: ## Integration tests in every package, each on its own database (needs make db and goose; a CI job, not part of check)
	@[ -n "$(DATABASE_URL)" ] || { echo "test-integration: DATABASE_URL is empty, nothing checked" >&2; exit 1; }; \
	command -v goose >/dev/null || { echo "test-integration: goose not installed (AGENTS.md, Toolchain), nothing checked" >&2; exit 1; }; \
	n=$$(go list -tags=integration ./... | wc -l | tr -d ' '); \
	set -o pipefail; DATABASE_URL="$(DATABASE_URL)" go test -race -count=1 -tags=integration ./... 2>&1 | tail -40 && echo "test-integration: $$n packages checked; per-run databases created through $(DATABASE_URL)"

# ---- Web gates --------------------------------------------------------------

web-format-check: ## Fail if any web file is unformatted (prettier)
	@[ -x $(WEB)/node_modules/.bin/prettier ] || $(call skip,web-format-check,prettier (run make setup)); \
	cd $(WEB) && $(PNPM) exec prettier --log-level warn --check . && echo "web-format-check: $$(git ls-files -co --exclude-standard . | wc -l | tr -d ' ') files in web/ checked"

web-lint: ## ESLint with zero warnings allowed
	@[ -x $(WEB)/node_modules/.bin/eslint ] || $(call skip,web-lint,eslint (run make setup)); \
	cd $(WEB) && $(PNPM) exec eslint . --max-warnings 0 && echo "web-lint: passed"

web-api-types: ## Regenerate web/src/lib/api-types.ts from api/openapi.yaml; commit it with the spec change
	@cd $(WEB) && $(PNPM) run --silent api-types && echo "web-api-types: src/lib/api-types.ts written from api/openapi.yaml"

# Tenet 7: the UI's types are generated from the spec. Generating and
# formatting happen in memory, so the committed file is compared, not touched.
web-api-types-check: ## Fail if web/src/lib/api-types.ts is not what api/openapi.yaml generates
	@[ -x $(WEB)/node_modules/.bin/openapi-typescript ] || $(call skip,web-api-types-check,openapi-typescript (run make setup)); \
	cd $(WEB) && [ -f src/lib/api-types.ts ] || { echo "web-api-types-check: src/lib/api-types.ts is missing; run make web-api-types" >&2; exit 1; }; \
	set -o pipefail; want=$$($(PNPM) exec openapi-typescript ../api/openapi.yaml | $(PNPM) exec prettier --stdin-filepath src/lib/api-types.ts) || \
		{ echo "web-api-types-check: generating from api/openapi.yaml failed" >&2; exit 1; }; \
	diff -u src/lib/api-types.ts <(printf '%s\n' "$$want") >/dev/null || { \
		echo "web-api-types-check: src/lib/api-types.ts does not match api/openapi.yaml; run make web-api-types and commit it" >&2; \
		diff -u src/lib/api-types.ts <(printf '%s\n' "$$want") | head -40 >&2; exit 1; }; \
	echo "web-api-types-check: src/lib/api-types.ts matches api/openapi.yaml ($$(wc -l < src/lib/api-types.ts | tr -d ' ') lines)"

web-typecheck: ## tsc --noEmit
	@[ -x $(WEB)/node_modules/.bin/tsc ] || $(call skip,web-typecheck,tsc (run make setup)); \
	cd $(WEB) && $(PNPM) exec tsc --noEmit -p tsconfig.json && echo "web-typecheck: passed"

web-test: ## Vitest unit tests (jsdom)
	@n=$$(cd $(WEB) && git ls-files -co --exclude-standard -- 'src/*.test.ts' 'src/*.test.tsx' | wc -l | tr -d ' '); \
	[ "$$n" -gt 0 ] || { echo "web-test: 0 test files, nothing checked" >&2; exit 1; }; \
	[ -x $(WEB)/node_modules/.bin/vitest ] || $(call skip,web-test,vitest (run make setup)); \
	set -o pipefail; cd $(WEB) && $(PNPM) exec vitest run 2>&1 | tail -20 && echo "web-test: $$n test files checked"

# ---- The gate ---------------------------------------------------------------

check: ## The offline gate: every gate in GATES, then the tally. CI runs exactly this.
	@mkdir -p $(STATE); rm -f $(SKIPPED) $(STATE)/.check-passed
	@for g in $(GATES); do $(MAKE) --no-print-directory $$g || { echo "check: $$g failed" >&2; exit 1; }; done
	@s=$$(cut -d' ' -f1 $(SKIPPED) 2>/dev/null | sort -u | wc -l | tr -d ' '); r=$$(( $(words $(GATES)) - s )); \
	echo "check: $$r gates run, $$s skipped"; \
	if [ "$$s" -eq 0 ]; then touch $(STATE)/.check-passed; echo "check: passed"; \
	else sed 's/^/  skipped: /' $(SKIPPED); echo "check: FAILED, $$s gate(s) skipped; install what is missing (make doctor)" >&2; exit 1; fi

check-file: ## Check one edited file, FILE=path (the Bearing edit hook runs this)
	@[ -n "$(FILE)" ] || { echo "check-file: FILE is empty, nothing checked" >&2; exit 1; }; \
	[ -f "$(FILE)" ] || { echo "check-file: $(FILE) does not exist, nothing checked" >&2; exit 1; }; \
	case "$(FILE)" in \
	  *.go) go vet "./$$(dirname "$(FILE)")" || exit 1;; \
	  web/*.ts|web/*.tsx|web/*.js) [ -x $(WEB)/node_modules/.bin/eslint ] || { echo "check-file: eslint not installed (make setup), $(FILE) not checked"; exit 0; }; \
	    (cd $(WEB) && ./node_modules/.bin/eslint --max-warnings 0 "$${FILE#web/}") || exit 1;; \
	  *) echo "check-file: no per-file check for $(FILE)"; exit 0;; \
	esac; \
	echo "check-file: 1 file checked"

fix: ## Apply every automatic fix (gofmt, golangci-lint --fix, prettier, eslint --fix)
	gofmt -w $(GO_FILES)
	@command -v golangci-lint >/dev/null && golangci-lint run --fix ./... || true
	cd $(WEB) && $(PNPM) exec prettier --log-level warn --write . && $(PNPM) exec eslint . --fix

vuln: ## govulncheck and pnpm audit (needs the network; a CI job, not part of check)
	@command -v govulncheck >/dev/null || { echo "vuln: govulncheck not installed" >&2; exit 1; }
	govulncheck ./...
	cd $(WEB) && $(PNPM) audit --audit-level=high
	@echo "vuln: Go modules and web packages checked"

# ---- Database ---------------------------------------------------------------

db: ## Start PostgreSQL and MailHog in Docker
	docker compose up -d --wait

db-down: ## Stop the containers (data kept)
	docker compose down

db-reset: ## Drop the local database volume and start again (destructive)
	docker compose down -v && docker compose up -d --wait

migrate: ## goose up (db/migrations)
	@n=$$(ls db/migrations/*.sql 2>/dev/null | wc -l | tr -d ' '); \
	[ "$$n" -gt 0 ] || { echo "migrate: 0 migrations in db/migrations, nothing applied" >&2; exit 1; }; \
	goose -dir db/migrations postgres "$(DATABASE_URL)" up

migrate-down: ## goose down one
	goose -dir db/migrations postgres "$(DATABASE_URL)" down

migrate-status: ## goose status
	goose -dir db/migrations postgres "$(DATABASE_URL)" status

sqlc: ## Regenerate typed queries into internal/store
	@n=$$(ls db/migrations/*.sql 2>/dev/null | wc -l | tr -d ' '); \
	[ "$$n" -gt 0 ] || { echo "sqlc: 0 migrations in db/migrations, nothing to generate yet" >&2; exit 1; }; \
	sqlc generate && echo "sqlc: generated into internal/store"

doctor: ## Environment diagnostics
	@echo "go:            $$(command -v go >/dev/null && go version || echo 'missing (Go 1.26.8, see AGENTS.md)')"
	@echo "node:          $$(command -v node >/dev/null && node --version || echo missing) (web/.nvmrc wants 24)"
	@echo "pnpm:          $$(command -v $(PNPM) >/dev/null && $(PNPM) --version 2>/dev/null || echo missing)"
	@echo "web deps:      $$([ -d $(WEB)/node_modules ] && echo installed || echo 'NOT INSTALLED (make setup)')"
	@echo "golangci-lint: $$(command -v golangci-lint >/dev/null && golangci-lint version 2>/dev/null | head -1 || echo missing)"
	@echo "govulncheck:   $$(command -v govulncheck >/dev/null && echo present || echo missing)"
	@echo "sqlc:          $$(command -v sqlc >/dev/null && sqlc version || echo missing)"
	@echo "goose:         $$(command -v goose >/dev/null && goose -version || echo missing)"
	@echo "docker:        $$(command -v docker >/dev/null && docker --version || echo missing)"
	@echo "hooksPath:     $$(git config core.hooksPath || echo 'NOT SET (make hooks)')"

clean: ## Remove build output
	rm -rf bin $(WEB)/dist $(WEB)/coverage $(STATE)/.check-passed $(SKIPPED)
