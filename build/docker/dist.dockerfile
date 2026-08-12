ARG APP_NAME=yaitoo

FROM imlangzi/yaitoo:golang AS yaitoo-build

WORKDIR /yaitoo

# Download Go dependencies (cached if mod/sum unchanged)
COPY ./go.mod .
COPY ./go.sum .
RUN go mod download


# Copy all sources
COPY . .

# Build UI assets (CSS via tailwindcss, JS via esbuild) then the Go
# binary. The Makefile's `build` target depends on `build-ui`, so a
# single `make build` produces both.
ENV GOCACHE=/root/.cache/go-build
RUN --mount=type=cache,target="/root/.cache/go-build" make build

# Export stage to allow docker build -o to output binaries directly.
# `make build` puts the binary at bin/app (not at the WORKDIR root),
# so the COPY source must reflect that — otherwise the `${APP_NAME}-*`
# glob matches nothing and `./dist/` ends up empty.
FROM scratch AS export-stage
ARG APP_NAME=yaitoo
COPY --from=yaitoo-build /yaitoo/bin/app /app