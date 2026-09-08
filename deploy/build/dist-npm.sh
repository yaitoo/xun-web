#!/bin/bash
#
# Build a Debian (linux/amd64) deploy artefact via Docker buildx
# against `deploy/build/npm.Dockerfile` — the worked example for
# consumers of `imlangzi/yaitoo:npm` who also need to compile Go.
# Output lands in ./dist/ as `app-npm`. This is a DEMONSTRATION of
# the BuildKit cache-mount pattern documented in `README.md §14`;
# it is not the production xun-web build (use `dist-golang.sh` for
# that).
#
# Usage (from the repo root):
#   ./deploy/build/dist-npm.sh              # produces ./dist/app-npm
#   APP_NAME=… ./deploy/build/dist-npm.sh   # override binary name
#
# Requires Docker with buildx (`brew install docker-buildx` on macOS).

set -euo pipefail

APP_NAME=${APP_NAME:-yaitoo}
DIST_DIR=${DIST_DIR:-./dist}

# Always land in the repo root regardless of cwd. `$(dirname "$0")/..`
# would only get us to `deploy/`, since this script lives at
# `deploy/build/dist-npm.sh` — one more `..` reaches the repo root.
cd "$(dirname "$0")/../.."

mkdir -p "$DIST_DIR"

# `--platform=linux/amd64` is intentionally fixed: every target host
# runs Debian amd64.
docker buildx build --progress plain \
  --platform=linux/amd64 \
  --build-arg "APP_NAME=$APP_NAME" \
  --target export-stage \
  -f ./deploy/build/npm.Dockerfile . \
  -o "type=local,dest=$DIST_DIR"

# Rename the artefact to `app-npm` so it doesn't collide with the
# golang build's `./dist/app` output. `dist-npm.sh` and `dist-golang.sh`
# can be run sequentially (or interleaved) without overwriting each
# other as long as this rename is in place.
mv -f "$DIST_DIR/app" "$DIST_DIR/app-npm"

echo
echo "==> built artefact:"
ls -lh "$DIST_DIR"
