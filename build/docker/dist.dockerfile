ARG APP_NAME=yaitoo

FROM yaitoo-debian AS yaitoo-build

WORKDIR /yaitoo

# Pre-download the standalone tailwindcss and esbuild CLIs to ./bin/ so
# the rest of the build can use them via `make build-ui`. Cached as a
# separate layer that is invalidated only when the Makefile (which pins
# the tool versions) changes.
RUN mkdir -p bin && \
    curl -fsSLk -o bin/tailwindcss \
        https://yaitoo.cn/tailwindcss && \
    chmod +x bin/tailwindcss && \
    curl -fsSL -o bin/esbuild \
        https://cdn.jsdelivr.net/npm/@esbuild/linux-x64@0.28.0/bin/esbuild && \
    chmod +x bin/esbuild

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

# Export stage to allow docker build -o to output binaries directly
FROM scratch AS export-stage
ARG APP_NAME=yaitoo
COPY --from=yaitoo-build /yaitoo/${APP_NAME}-* /