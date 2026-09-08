ARG APP_NAME=yaitoo

# Worked example: BuildKit cache-mount pattern for `imlangzi/yaitoo:npm`
# consumers that ALSO need to compile Go (e.g. a Go binary that embeds
# a JS-bundled UI, or a build that calls `pnpm run build` from `make`).
#
# This file is a DEMONSTRATION, not the production xun-web build —
# see `deploy/build/golang.Dockerfile` for the actual production build,
# and `README.md §14` for the full pattern and rationale.
#
# Key idea: `--mount=type=cache` is scoped per RUN. To share the pnpm
# and Go caches across RUNs (and across builds), declare stable
# `id=pnpm` / `id=gomod` / `id=gobuild` on every RUN that needs them.
# Anonymous mounts (no `id=`) may get a fresh namespace per RUN and
# defeat the cache.
#
# Layer ordering: front-load the slowest-changing deps first. pnpm-lock
# changes very rarely, so `pnpm install` sits at the top of the stack
# and is cached most often. Go modules change occasionally. Source
# code changes frequently — it sits at the bottom of the stack, where
# invalidation is cheap (the caches above are already warm).

FROM imlangzi/yaitoo:npm AS build

WORKDIR /app

# Layer 1: pnpm manifests. Invalidates only on pnpm-lock.yaml /
# package.json change. pnpm-lock changes very rarely — keep this COPY
# FIRST so the largest payload is reused most often.
COPY ./pnpm-lock.yaml ./package.json ./

# Layer 2: pnpm install. Mounts `id=pnpm` with a stable namespace so
# the populated store persists across RUNs and across builds.
RUN --mount=type=cache,id=pnpm,target=${PNPM_HOME} \
    pnpm install --frozen-lockfile

# Layer 3: Go module manifests. Invalidates only on go.mod/go.sum change.
COPY ./go.mod ./go.sum ./

# Layer 4: go mod download. Mounts `id=gomod` with a stable
# namespace. `go mod download` is module-fetching only — it does not
# invoke the compiler, so it doesn't need `id=gobuild`. Do NOT combine
# with `pnpm install` above: keeping them in separate RUNs preserves
# layer caching for source-only changes.
RUN --mount=type=cache,id=gomod,target=/root/go/pkg/mod \
    go mod download

# Layer 5: source. Invalidates every RUN below it on any source change.
COPY . ./

# Pin $GOCACHE to the cache mount's target. Defensive.
ENV GOCACHE=/root/.cache/go-build

# Layer 6: build. Redeclare ALL three `id=` namespaces so the build
# reads from all warm caches. `make build` may transitively call
# `pnpm run build`, which reads from `${PNPM_HOME}` — so `id=pnpm`
# must be re-mounted here.
RUN --mount=type=cache,id=gomod,target=/root/go/pkg/mod \
    --mount=type=cache,id=gobuild,target=/root/.cache/go-build \
    --mount=type=cache,id=pnpm,target=${PNPM_HOME} \
    make build

# Export stage to allow docker build -o to output binaries directly.
# `make build` puts the binary at bin/app (not at the WORKDIR root).
FROM scratch AS export-stage
ARG APP_NAME=yaitoo
COPY --from=build /app/bin/app /app
