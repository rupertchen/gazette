#!/usr/bin/env bash
set -Eeu -o pipefail

# Root directory of repository.
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"

# Build a cache-able runtime base image plus vendor'd dependencies.
docker build ${ROOT} \
    --file ${ROOT}/v2/build/Dockerfile \
    --target vendored \
    --tag liveramp/gazette-vendored:latest \
    --cache-from liveramp/gazette-vendored:lates

# Build and test Gazette. This image includes all Gazette source, compiled
# packages and binaries, and only completes after all tests pass.
docker build ${ROOT} \
    --file ${ROOT}/v2/build/Dockerfile \
    --target build \
    --tag liveramp/gazette-build:latest

# Create the `gazette` image, which plucks the `gazette`, `gazctl` and
# `run-consumer` onto a base runtime image.
docker build ${ROOT} \
    --file ${ROOT}/v2/build/Dockerfile \
    --target gazette \
    --tag liveramp/gazette:latest

# Create the `gazette-examples` image, which further plucks `stream-sum` and
# `word-count` example artifacts onto the `gazette` image.
docker build ${ROOT} \
    --file ${ROOT}/v2/build/Dockerfile \
    --target examples \
    --tag liveramp/gazette-examples:latest
