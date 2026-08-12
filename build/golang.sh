#!/bin/bash

set -e

# Resolve project root from this script's location so it works whether
# called as `./build/golang.sh` from the project root or as `./golang.sh`
# from the build/ directory.
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

cd "$PROJECT_ROOT"

docker build --progress plain \
  -f ./build/docker/golang.dockerfile \
  . \
  -t imlangzi/yaitoo:golang