.PHONY: install dev build build-ui build-dist watch clean run fmt tidy download-ui-tools require-env env-show

# ── .env auto-loading ────────────────────────────────────────────────────────
# `.env` is git-ignored; `.env.example` is the tracked template. The app
# reads every APP_* env var via viper.AutomaticEnv() (see cmd/app/main.go
# loadConfig), so a single `.env` is the place to keep secrets and per-
# developer overrides.
#
# Two layers of integration:
#
#   (a) `include $(ENV_FILE)` + bare `export` — make parses `.env`, so
#       `$(APP_SESSION_SECRET)` works in any recipe line below. (We
#       don't currently use it, but it keeps `.env` discoverable via
#       `make -p` and lets a future recipe reference any var directly.)
#
#   (b) `$(ENV_LOAD)` — prepended to recipes that run the app, sources
#       `.env` in a subshell with `set -a` so every var is auto-exported
#       into the spawned `go run` / ansible-playbook process. The Go app
#       then picks them up via its own viper.AutomaticEnv() layer.
#
# Already-exported shell vars win over `.env` (make's `include`
# semantics: env-vars override vars set in the included file).
ENV_FILE := .env
ifeq ($(wildcard $(ENV_FILE)),)
ENV_LOAD :=
else
# `$(CURDIR)/$(ENV_FILE)` (not bare `./.env`) so POSIX `.` resolves it
# as a path instead of searching $PATH, and so the source still works
# after a recipe's `cd ...` (e.g. `cd deploy/playbooks`).
ENV_LOAD := set -a; . $(CURDIR)/$(ENV_FILE); set +a;
include $(ENV_FILE)
export
endif

# Fail-loud guard for targets that need $(ENV_FILE) to be present.
# Used as a prerequisite by every target that consumes $(ENV_LOAD) so
# the error wording and exit behaviour stay consistent.
require-env:
	@if [ ! -f $(ENV_FILE) ]; then \
		echo "  ! $(ENV_FILE) not found — copy .env.example to .env first" >&2; \
		exit 1; \
	fi

# Print the effective vars loaded from .env (debug aid).
env-show: require-env
	@echo "Loaded from $(ENV_FILE):"
	@grep -vE '^[[:space:]]*(#|$$)' $(ENV_FILE) | sed 's/^/  /'

# ── Go production build settings ──────────────────────────────────────────
# Identity injected into the binary via -ldflags -X. main.go declares
# matching package-level vars (Version, Commit) so they can be printed
# at runtime for support / debugging.
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)

# Strip symbol table / DWARF (-s -w), wipe the build ID, and inject the
# two identity vars above. Quoted so spaces in the version string don't
# split the argument.
LDFLAGS := -s -w -buildid= \
           -X 'main.Version=$(VERSION)' \
           -X 'main.Commit=$(COMMIT)'

# -trimpath removes the absolute paths to the source tree and module cache
# from the binary. Combined with the LDFLAGS above, this gives reproducible
# builds and prevents leaking the build host's directory layout.
GO_BUILD_FLAGS := -trimpath


# Versions of the standalone CLI tools we ship.
TAILWIND_VERSION := 3.4.17
ESBUILD_VERSION  := 0.28.0

# CSS / JS source → output paths. The output is served by the running app
# from cmd/app/app/public/assets/ (embedded via //go:embed in main.go).
TAILWIND_INPUT  := tailwind.css
TAILWIND_OUTPUT := cmd/app/app/public/assets/app.css
TAILWIND_CONFIG := cmd/app/app/tailwind.config.js
ESBUILD_INPUT   := cmd/app/app/public/assets/app.js
ESBUILD_OUTPUT  := cmd/app/app/public/assets/app.min.js

# Shared curl flags for the two UI tool downloads: fail on HTTP error (-f),
# follow redirects (-L, GitHub release URL → release-assets host), show a
# progress bar (--progress-bar), and resume from any existing partial file
# (-C -).
CURL_FLAGS := -fL --progress-bar -C -

# Download tailwindcss and esbuild CLIs to ./bin/. Separated from build-ui
# so Docker / Make can cache this layer independently. Supports Linux and
# macOS on x64 and ARM64.
#
# Resolution order (per tool):
#   1. ./bin/<tool> already exists → use it
#   2. <tool> found in PATH → symlink it into ./bin/
#   3. Otherwise → download from upstream
#
# This order ensures Docker builds use the staged binary from COPY/RUN layers
# before falling back to PATH or network download, fixing offline builds.
define resolve-cli
@if [ -e bin/$1 ]; then \
	echo "Using bin/$1 (already present)"; \
elif BIN_PATH=$$(command -v $1 2>/dev/null); then \
	echo "Symlinking $$BIN_PATH -> bin/$1"; \
	ln -sf "$$BIN_PATH" bin/$1; \
else \
	echo "Downloading $2..."; \
	if ! curl $(CURL_FLAGS) -o bin/$1.tmp "$3"; then \
		echo "  ERROR: failed to download $2" >&2; \
		rm -f bin/$1.tmp; \
		exit 1; \
	fi; \
	mv bin/$1.tmp bin/$1; \
	chmod +x bin/$1; \
	echo "  $2 downloaded to bin/$1"; \
fi
endef

TAILWIND_PLATFORM := $(shell uname -s | tr '[:upper:]' '[:lower:]')-$(shell uname -m | sed 's/aarch64/arm64/; s/x86_64/x64/; s/^darwin/macos/')
TAILWIND_URL := https://github.com/tailwindlabs/tailwindcss/releases/download/v$(TAILWIND_VERSION)/tailwindcss-$(TAILWIND_PLATFORM)

ESBUILD_PLATFORM := $(shell uname -s | tr '[:upper:]' '[:lower:]')-$(shell uname -m | sed 's/aarch64/arm64/; s/x86_64/x64/')
ESBUILD_URL := https://cdn.jsdelivr.net/npm/@esbuild/$(ESBUILD_PLATFORM)@$(ESBUILD_VERSION)/bin/esbuild

download-ui-tools:
	@mkdir -p bin
	$(call resolve-cli,tailwindcss,tailwindcss v$(TAILWIND_VERSION),$(TAILWIND_URL))
	$(call resolve-cli,esbuild,esbuild v$(ESBUILD_VERSION),$(ESBUILD_URL))

# Build the admin UI CSS and JS bundles. Idempotent: re-runs overwrite the
# outputs. Tailwind is run with -c to point at the v3 config in
# cmd/app/app/. esbuild is only invoked if a source file exists — projects
# without any client-side JS can ship a CSS-only bundle.
build-ui: download-ui-tools
	@echo "Building admin UI CSS..."
	@bin/tailwindcss -c $(TAILWIND_CONFIG) -i $(TAILWIND_INPUT) -o $(TAILWIND_OUTPUT) --minify
	@if [ -f $(ESBUILD_INPUT) ]; then \
		echo "Building admin UI JS bundle..."; \
		bin/esbuild $(ESBUILD_INPUT) --bundle --minify --format=esm --target=es2020 --outfile=$(ESBUILD_OUTPUT); \
	else \
		echo "  no $(ESBUILD_INPUT); skipping esbuild"; \
	fi
	@echo "  build-ui done"

# Local build: compile the UI assets and the Go binary in the *current*
# environment, into ./bin/. Use this for development and for producing a
# binary that runs on the same OS/arch as the host.
#
# For a deployable distribution artefact (cross-compiled via Docker and
# packaged for Debian), use `make build-dist` instead — its output lands
# in ./dist/ and never overlaps with ./bin/.
build: build-ui
	go build $(GO_BUILD_FLAGS) -ldflags '$(LDFLAGS)' -o bin/app ./cmd/app

# Watch Tailwind CSS and rebuild on changes. esbuild has no watch in this
# minimal pipeline; re-run `make build` (or the build-ui target) when the
# JS source changes.
watch: download-ui-tools
	bin/tailwindcss -c $(TAILWIND_CONFIG) -i $(TAILWIND_INPUT) -o $(TAILWIND_OUTPUT) --watch

# Development: watch Tailwind in the background and start the Go app.
# `go run ./cmd/app` (directory form) is required — see the `run` target.
# `$(ENV_LOAD)` sources .env so APP_* vars reach the go process.
dev: download-ui-tools require-env
	@bin/tailwindcss -c $(TAILWIND_CONFIG) -i $(TAILWIND_INPUT) -o $(TAILWIND_OUTPUT) --watch & \
	  $(ENV_LOAD) go run ./cmd/app

# Run the application (assumes `make build-ui` has been run at least once).
# `go run ./cmd/app` (directory form) compiles the whole `package main`,
# not just main.go — `go run FILE` is file mode and would miss sibling
# sources like db.go, routes.go, middleware.go.
# `$(ENV_LOAD)` sources .env so APP_* vars reach the go process.
run: require-env
	$(ENV_LOAD) go run ./cmd/app

# Remove the downloaded CLI binaries and the generated UI assets. Leaves
# the .gitignored ./bin/ folder in place.
clean:
	rm -f bin/tailwindcss bin/esbuild
	rm -f $(TAILWIND_OUTPUT) $(ESBUILD_OUTPUT)

# Format Go code
fmt:
	go fmt ./...

# Tidy Go modules
tidy:
	go mod tidy

# Distribution build: run the Docker pipeline (./build/dist.sh) to
# produce the deployable package and export it via `buildx --output`
# into ./dist/. Kept strictly separate from `./bin/` (the output of the
# local `build` target) so the two never overwrite each other.
build-dist:
	@mkdir -p dist
	$(ENV_LOAD) ./build/dist.sh

# Convenience: fetch the UI tools without building anything.
install: download-ui-tools
