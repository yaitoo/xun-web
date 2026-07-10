.PHONY: install dev build build-ui build-dist watch clean run fmt tidy download-ui-tools install-system require-env env-show

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
# If tailwindcss or esbuild are already in PATH, create symlinks instead of
# downloading. This allows using system-installed versions (via npm, apt, etc).
download-ui-tools:
	@mkdir -p bin
	@# Check for tailwindcss in PATH
	@if command -v tailwindcss >/dev/null 2>&1; then \
		if [ ! -e bin/tailwindcss ]; then \
			echo "Using tailwindcss from PATH: $$(command -v tailwindcss)"; \
			ln -sf $$(command -v tailwindcss) bin/tailwindcss; \
		fi; \
	elif [ ! -x bin/tailwindcss ]; then \
		OS=$$(uname -s | tr '[:upper:]' '[:lower:]'); \
		ARCH=$$(uname -m); \
		if [ "$$OS" = "darwin" ]; then \
			if [ "$$ARCH" = "arm64" ]; then \
				TAILWIND_PLATFORM="macos-arm64"; \
			else \
				TAILWIND_PLATFORM="macos-x64"; \
			fi; \
		elif [ "$$OS" = "linux" ]; then \
			if [ "$$ARCH" = "aarch64" ] || [ "$$ARCH" = "arm64" ]; then \
				TAILWIND_PLATFORM="linux-arm64"; \
			else \
				TAILWIND_PLATFORM="linux-x64"; \
			fi; \
		else \
			echo "  ERROR: Unsupported OS: $$OS" >&2; \
			exit 1; \
		fi; \
		echo "Downloading tailwindcss v$(TAILWIND_VERSION) for $$TAILWIND_PLATFORM..."; \
		if ! curl $(CURL_FLAGS) -o bin/tailwindcss.tmp \
			"https://github.com/tailwindlabs/tailwindcss/releases/download/v$(TAILWIND_VERSION)/tailwindcss-$$TAILWIND_PLATFORM"; then \
			echo "  ERROR: failed to download tailwindcss" >&2; \
			rm -f bin/tailwindcss.tmp; \
			exit 1; \
		fi; \
		mv bin/tailwindcss.tmp bin/tailwindcss; \
		chmod +x bin/tailwindcss; \
		echo "  tailwindcss downloaded"; \
	fi
	@# Check for esbuild in PATH
	@if command -v esbuild >/dev/null 2>&1; then \
		if [ ! -e bin/esbuild ]; then \
			echo "Using esbuild from PATH: $$(command -v esbuild)"; \
			ln -sf $$(command -v esbuild) bin/esbuild; \
		fi; \
	elif [ ! -x bin/esbuild ]; then \
		OS=$$(uname -s | tr '[:upper:]' '[:lower:]'); \
		ARCH=$$(uname -m); \
		if [ "$$OS" = "darwin" ]; then \
			if [ "$$ARCH" = "arm64" ]; then \
				ESBUILD_PACKAGE="@esbuild/darwin-arm64"; \
			else \
				ESBUILD_PACKAGE="@esbuild/darwin-x64"; \
			fi; \
		elif [ "$$OS" = "linux" ]; then \
			if [ "$$ARCH" = "aarch64" ] || [ "$$ARCH" = "arm64" ]; then \
				ESBUILD_PACKAGE="@esbuild/linux-arm64"; \
			else \
				ESBUILD_PACKAGE="@esbuild/linux-x64"; \
			fi; \
		else \
			echo "  ERROR: Unsupported OS: $$OS" >&2; \
			exit 1; \
		fi; \
		echo "Downloading esbuild v$(ESBUILD_VERSION) for $$ESBUILD_PACKAGE..."; \
		if ! curl $(CURL_FLAGS) -o bin/esbuild.tmp \
			"https://cdn.jsdelivr.net/npm/$$ESBUILD_PACKAGE@$(ESBUILD_VERSION)/bin/esbuild"; then \
			echo "  ERROR: failed to download esbuild" >&2; \
			rm -f bin/esbuild.tmp; \
			exit 1; \
		fi; \
		chmod +x bin/esbuild.tmp; \
		mv bin/esbuild.tmp bin/esbuild; \
		echo "  esbuild downloaded"; \
	fi

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

# Install downloaded UI tools to /usr/local/bin for system-wide access.
# This allows `make download-ui-tools` to use PATH versions instead of downloading.
# Requires sudo/write permissions to /usr/local/bin.
install-system: download-ui-tools
	@echo "Installing tailwindcss and esbuild to /usr/local/bin..."
	@if [ -x bin/tailwindcss ] && [ ! -L bin/tailwindcss ]; then \
		sudo cp bin/tailwindcss /usr/local/bin/tailwindcss; \
		echo "  tailwindcss installed to /usr/local/bin/tailwindcss"; \
	else \
		echo "  tailwindcss is a symlink or doesn't exist, skipping"; \
	fi
	@if [ -x bin/esbuild ] && [ ! -L bin/esbuild ]; then \
		sudo cp bin/esbuild /usr/local/bin/esbuild; \
		echo "  esbuild installed to /usr/local/bin/esbuild"; \
	else \
		echo "  esbuild is a symlink or doesn't exist, skipping"; \
	fi
	@echo "Done! Run 'make clean' then 'make download-ui-tools' to use system versions."
