#!/bin/bash
set -e

DOCKER_REF=${DOCKER_REF:-latest}
APP_NAME=${APP_NAME:-yaitoo}
DIST_DIR=${DIST_DIR:-./dist}

# Resolve project root from this script's location so it works whether
# called as `./build/dist.sh` from the project root or as `./dist.sh`
# from the build/ directory.
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

cd "$PROJECT_ROOT"

mkdir -p "$DIST_DIR"

# Build and export in one shot (single stage). Output goes to ./dist/
# (kept separate from ./bin/ which holds the locally-compiled Go binary).
docker build --progress plain \
  --build-arg DOCKER_REF=$DOCKER_REF \
  --build-arg APP_NAME=$APP_NAME \
  --target export-stage \
  -f ./build/docker/dist.dockerfile . \
  -o "type=local,dest=$DIST_DIR"