#!/bin/bash
set -e

DOCKER_REF=${DOCKER_REF:-latest}
APP_NAME=${APP_NAME:-yaitoo}
DIST_DIR=${DIST_DIR:-./dist}

# Move to project root so Dockerfile can COPY the whole tree
cd ..

mkdir -p "$DIST_DIR"

# Build and export in one shot (single stage). Output goes to ./dist/
# (kept separate from ./bin/ which holds the locally-compiled Go binary).
docker build --progress plain \
  --build-arg DOCKER_REF=$DOCKER_REF \
  --build-arg APP_NAME=$APP_NAME \
  --target export-stage \
  -f ./build/docker/dist.dockerfile . \
  -o "type=local,dest=$DIST_DIR"