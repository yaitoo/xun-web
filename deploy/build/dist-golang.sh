#!/bin/bash
#
# Build the Debian (linux/amd64) deploy artefact via Docker buildx
# against `deploy/build/golang.Dockerfile` and dump it into ./dist/
# as `app`. Used by `ansible-playbook app.yml` to copy the binary
# onto each target host.
#
# For a worked example of the pnpm + Go BuildKit cache-mount pattern
# (using `deploy/build/npm.Dockerfile`), see `dist-npm.sh` in this
# directory. See `README.md §14` for the full pattern and rationale.
#
# Usage (from the repo root):
#   ./deploy/build/dist-golang.sh              # produces ./dist/app
#   APP_NAME=… ./deploy/build/dist-golang.sh   # override binary name
#
# Requires Docker with buildx (`brew install docker-buildx` on macOS).

set -euo pipefail

APP_NAME=${APP_NAME:-yaitoo}
DIST_DIR=${DIST_DIR:-./dist}

# Always land in the repo root regardless of cwd. `$(dirname "$0")/..`
# would only get us to `deploy/`, since this script now lives at
# `deploy/build/dist-golang.sh` — one more `..` reaches the repo root.
cd "$(dirname "$0")/../.."

mkdir -p "$DIST_DIR"

# `--platform=linux/amd64` is intentionally fixed: every target host
# runs Debian amd64.
docker buildx build --progress plain \
  --platform=linux/amd64 \
  --build-arg "APP_NAME=$APP_NAME" \
  --target export-stage \
  -f ./deploy/build/golang.Dockerfile . \
  -o "type=local,dest=$DIST_DIR"

echo
echo "==> built artefact:"
ls -lh "$DIST_DIR"