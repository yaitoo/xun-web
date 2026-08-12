#!/bin/bash
#
# Build the Debian (linux/amd64) deploy artefact via Docker buildx
# and dump it into ./dist/ as `app`. Used by `ansible-playbook
# app.yml` to copy the binary onto each target host.
#
# Usage (from the repo root):
#   ./build/dist.sh              # produces ./dist/app
#   APP_NAME=… ./build/dist.sh   # override binary name (rare)
#
# Requires Docker with buildx (`brew install docker-buildx` on macOS).

set -euo pipefail

APP_NAME=${APP_NAME:-yaitoo}
DIST_DIR=${DIST_DIR:-./dist}

# Always land in the repo root regardless of cwd. `cd ..` would walk
# OUT of the repo when invoked via `./build/dist.sh`.
cd "$(dirname "$0")/.."

mkdir -p "$DIST_DIR"

# `--platform=linux/amd64` is intentionally fixed: every target host
# runs Debian amd64.
docker buildx build --progress plain \
  --platform=linux/amd64 \
  --build-arg "APP_NAME=$APP_NAME" \
  --target export-stage \
  -f ./build/docker/dist.dockerfile . \
  -o "type=local,dest=$DIST_DIR"

echo
echo "==> built artefact:"
ls -lh "$DIST_DIR"