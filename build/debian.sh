#!/bin/bash

set -e

cd ..

docker build --progress plain -f ./build/docker/debian.dockerfile . -t yaitoo-debian:latest