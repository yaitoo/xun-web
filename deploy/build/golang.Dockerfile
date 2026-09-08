ARG APP_NAME=yaitoo

FROM imlangzi/yaitoo:golang AS yaitoo-build

WORKDIR /yaitoo

# Layer 1: Go module manifests. Invalidates only on go.mod/go.sum change
# — not on every source-file edit. Keeping this COPY separate from
# `COPY . .` below is what lets the pre-warm layer survive source-only
# edits.
COPY ./go.mod ./go.sum ./

# Layer 2: go mod download. Stable `id=` namespaces are critical: without
# them, anonymous mounts may get a fresh namespace per RUN and defeat
# cross-RUN sharing. See README §14 for the full pattern and rationale.
RUN --mount=type=cache,id=gomod,target=/root/go/pkg/mod \
    go mod download

# Layer 3: source. Invalidates every RUN below it on any source change.
COPY . ./

# Pin $GOCACHE to the cache mount's target. Defensive: if the mount
# is ever misconfigured, `go build` still has a stable cache path.
ENV GOCACHE=/root/.cache/go-build

# Layer 4: build. Redeclare `id=gomod` and `id=gobuild` so `go build`
# reads the warm module + compile caches populated above. The cache
# mount is unmounted when the RUN exits, so we must re-mount here.
#
# The Makefile's `build` target depends on `build-ui`, so a single
# `make build` runs the full UI toolchain (tailwindcss + esbuild) and
# then the Go binary.
RUN --mount=type=cache,id=gomod,target=/root/go/pkg/mod \
    --mount=type=cache,id=gobuild,target=/root/.cache/go-build \
    make build

# Export stage to allow docker build -o to output binaries directly.
# `make build` puts the binary at bin/app (not at the WORKDIR root),
# so the COPY source must reflect that — otherwise `./dist/` ends up
# empty.
FROM scratch AS export-stage
ARG APP_NAME=yaitoo
COPY --from=yaitoo-build /yaitoo/bin/app /app
